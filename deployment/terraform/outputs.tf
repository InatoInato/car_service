output "public_ip" {
  description = "Public IP address of the EC2 instance"
  value       = aws_eip.car_service.public_ip
}

output "ssh_command" {
  description = "SSH command for connecting to EC2"
  value       = "ssh -i <your-private-key> ubuntu@${aws_eip.car_service.public_ip}"
}

output "my_ip_detected" {
  description = "IP address allowed to access SSH"
  value       = local.my_cidr
}