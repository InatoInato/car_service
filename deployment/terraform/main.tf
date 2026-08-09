terraform {
  required_version = ">= 1.9.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }

    http = {
      source  = "hashicorp/http"
      version = "~> 3.5"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

resource "aws_security_group" "car_service" {
  name        = "car-service-sg"
  description = "Security group for car service"

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [local.my_cidr]
  }

  ingress {
    description = "Car Service API"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
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

  key_name               = aws_key_pair.car_service.key_name
  vpc_security_group_ids = [aws_security_group.car_service.id]

  user_data = file("${path.module}/user_data.sh")

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