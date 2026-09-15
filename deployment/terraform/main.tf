terraform {
  required_version = ">= 1.10.0, < 2.0.0"

  # Local state for this single-operator EC2 project; no S3 bucket required.

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

resource "aws_security_group" "car_service" {
  name        = "car-service-sg"
  description = "Security group for car service"
  vpc_id      = data.aws_subnet.car_service.vpc_id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.allowed_ssh_cidr]
  }

  ingress {
    description = "Car Service API"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = [var.allowed_ssh_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "car-service-sg"
  }
}

resource "aws_key_pair" "car_service" {
  key_name   = "car-service"
  public_key = var.ssh_public_key
}

resource "aws_instance" "car_service" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = var.instance_type
  subnet_id     = var.subnet_id

  # The chosen subnet must have an Internet Gateway route for bootstrap.
  associate_public_ip_address = true

  key_name               = aws_key_pair.car_service.key_name
  vpc_security_group_ids = [aws_security_group.car_service.id]

  user_data = file("${path.module}/user_data.sh")

  disable_api_termination = true

  root_block_device {
    # Keep the disk if an operator deliberately terminates the instance.
    delete_on_termination = false
    tags = {
      Backup = "car-service-daily"
    }
  }

  lifecycle {
    prevent_destroy = true

    # User-data is for first boot. Updating this file must not restart a server
    # without running the new script. Existing hosts need a planned update.
    ignore_changes = [user_data]
  }

  tags = {
    Name = "car-service"
  }
}

resource "aws_eip" "car_service" {
  instance = aws_instance.car_service.id
  domain   = "vpc"

  tags = {
    Name = "car-service-eip"
  }
}

# AWS takes disk snapshots without running a backup job on the small server.
resource "aws_iam_role" "snapshots" {
  name_prefix = "car-service-snapshots-"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "dlm.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "snapshots" {
  role       = aws_iam_role.snapshots.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/service-role/AWSDataLifecycleManagerServiceRole"
}

resource "aws_dlm_lifecycle_policy" "snapshots" {
  description        = "Daily car service disk snapshots - keep the last seven"
  execution_role_arn = aws_iam_role.snapshots.arn
  state              = "ENABLED"

  policy_details {
    resource_types = ["VOLUME"]
    target_tags = {
      Backup = "car-service-daily"
    }

    schedule {
      name      = "daily"
      copy_tags = true
      create_rule {
        interval      = 24
        interval_unit = "HOURS"
        times         = ["03:00"]
      }
      retain_rule {
        count = 7
      }
    }
  }

  depends_on = [aws_iam_role_policy_attachment.snapshots]
}
