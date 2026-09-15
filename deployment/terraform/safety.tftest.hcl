# These tests use fake AWS responses. They do not create cloud resources.
mock_provider "aws" {
  mock_data "aws_subnet" {
    defaults = { vpc_id = "vpc-0123456789abcdef0" }
  }
  mock_data "aws_ami" {
    defaults = { id = "ami-0123456789abcdef0" }
  }
  mock_data "aws_partition" {
    defaults = { partition = "aws" }
  }
}

variables {
  ami_id           = "ami-0123456789abcdef0"
  subnet_id        = "subnet-0123456789abcdef0"
  allowed_ssh_cidr = "203.0.113.10/32"
  ssh_public_key   = "mock-public-key"
}

run "safe_defaults" {
  command = plan
  assert {
    condition     = length(aws_security_group.car_service.ingress) == 2 && alltrue([for rule in aws_security_group.car_service.ingress : toset(rule.cidr_blocks) == toset([var.allowed_ssh_cidr])])
    error_message = "SSH and API must be limited to the chosen address."
  }
  assert {
    condition     = aws_instance.car_service.subnet_id == var.subnet_id && aws_security_group.car_service.vpc_id == data.aws_subnet.car_service.vpc_id
    error_message = "Instance and firewall must use the selected network."
  }
  assert {
    condition     = one([for filter in data.aws_ami.ubuntu.filter : filter if filter.name == "image-id"]).values == toset([var.ami_id])
    error_message = "AMI lookup must select the fixed ID."
  }
  assert {
    condition     = aws_instance.car_service.disable_api_termination && !aws_instance.car_service.root_block_device[0].delete_on_termination
    error_message = "Server termination and disk deletion need protection."
  }
  assert {
    condition     = aws_instance.car_service.root_block_device[0].tags["Backup"] == aws_dlm_lifecycle_policy.snapshots.policy_details[0].target_tags["Backup"]
    error_message = "The backup policy must select the server disk."
  }
  assert {
    condition     = one(aws_dlm_lifecycle_policy.snapshots.policy_details[0].schedule).create_rule[0].interval == 24 && one(aws_dlm_lifecycle_policy.snapshots.policy_details[0].schedule).retain_rule[0].count == 7 && aws_dlm_lifecycle_policy.snapshots.state == "ENABLED"
    error_message = "Daily snapshots must keep the last seven backups."
  }
}

run "reject_world_access" {
  command = plan
  variables { allowed_ssh_cidr = "0.0.0.0/0" }
  expect_failures = [var.allowed_ssh_cidr]
}

run "reject_bad_ip" {
  command = plan
  variables { allowed_ssh_cidr = "999.1.1.1/32" }
  expect_failures = [var.allowed_ssh_cidr]
}

run "reject_ipv6" {
  command = plan
  variables { allowed_ssh_cidr = "2001:db8::/32" }
  expect_failures = [var.allowed_ssh_cidr]
}

run "reject_bad_ami" {
  command = plan
  variables { ami_id = "latest" }
  expect_failures = [var.ami_id]
}

run "reject_bad_subnet" {
  command = plan
  variables { subnet_id = "default" }
  expect_failures = [var.subnet_id]
}
