#!/bin/bash
set -euxo pipefail

exec > >(tee /var/log/user-data.log | logger -t user-data -s 2>/dev/console) 2>&1

echo "=== EC2 bootstrap started ==="

export DEBIAN_FRONTEND=noninteractive

# Wait for networking
until curl -fsS https://archive.ubuntu.com >/dev/null; do
    echo "Waiting for internet..."
    sleep 5
done

echo "=== Updating apt ==="

apt-get update -y
apt-get install -y ca-certificates curl gnupg

echo "=== Adding Docker repository ==="

install -m 0755 -d /etc/apt/keyrings

curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
    -o /etc/apt/keyrings/docker.asc

chmod a+r /etc/apt/keyrings/docker.asc

echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
    https://download.docker.com/linux/ubuntu \
    $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
    > /etc/apt/sources.list.d/docker.list

echo "=== Installing Docker ==="

apt-get update -y

apt-get install -y \
    docker-ce \
    docker-ce-cli \
    containerd.io \
    docker-buildx-plugin \
    docker-compose-plugin

echo "=== Starting Docker ==="

systemctl enable docker
systemctl start docker

echo "=== Configuring ubuntu user ==="

usermod -aG docker ubuntu

echo "=== Verifying installation ==="

docker --version
docker compose version
docker buildx version

echo "=== EC2 bootstrap completed successfully ==="