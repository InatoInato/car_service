output "public_ip" {
  description = "Public IP address of the EC2 instance"
  value       = aws_eip.car_service.public_ip
}

output "ssh_command" {
  description = "SSH command for connecting to EC2"
  value       = "ssh -i <your-private-key> ubuntu@${aws_eip.car_service.public_ip}"
}

output "my_ip_detected" {
  # Keep the output name for existing scripts, but do not detect an IP at plan time.
  description = "Configured address allowed to access SSH and the API"
  value       = var.allowed_ssh_cidr
}

output "root_volume_id" {
  description = "Disk to identify when checking snapshots or planning recovery"
  value       = aws_instance.car_service.root_block_device[0].volume_id
}

output "snapshot_policy_id" {
  description = "Daily snapshot policy to check in the AWS console"
  value       = aws_dlm_lifecycle_policy.snapshots.id
}
