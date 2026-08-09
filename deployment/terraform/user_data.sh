#!/bin/bash

set -euxo pipefail

exec > /var/log/user-data.log 2>&1

apt-get update
apt-get install -y docker.io docker-compose-plugin

systemctl enable docker
systemctl start docker

usermod -aG docker ubuntu

echo "Docker installation completed"