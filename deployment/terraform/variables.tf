variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "ssh_public_key" {
  description = "SSH public key used to access EC2"
  type        = string
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t2.micro"
}

variable "ami_id" {
  description = "Fixed Canonical Ubuntu 22.04 amd64 AMI ID in aws_region. For an existing server, use its current AMI ID."
  type        = string
  validation {
    condition     = can(regex("^ami-([0-9a-f]{8}|[0-9a-f]{17})$", var.ami_id))
    error_message = "Use an AMI ID such as ami-0123456789abcdef0."
  }
}

variable "subnet_id" {
  description = "Public subnet with an Internet Gateway route. For an existing server, use its current subnet ID."
  type        = string
  validation {
    condition     = can(regex("^subnet-([0-9a-f]{8}|[0-9a-f]{17})$", var.subnet_id))
    error_message = "Use a subnet ID such as subnet-0123456789abcdef0."
  }
}

variable "allowed_ssh_cidr" {
  description = "Your fixed public IPv4 address with /32. Only this address may reach SSH and the learning API."
  type        = string
  validation {
    condition     = can(cidrnetmask(var.allowed_ssh_cidr)) && can(regex("/32$", var.allowed_ssh_cidr))
    error_message = "Use one public IPv4 address with /32, such as 203.0.113.10/32."
  }
}
