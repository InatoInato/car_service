#!/bin/bash
set -Eeuo pipefail
umask 022

# Cloud-init also captures stdout/stderr in /var/log/cloud-init-output.log.
# Avoid xtrace: future registry credentials must not be expanded into logs.
exec > >(tee -a /var/log/user-data.log) 2>&1
stage="initialization"
trap 'status=$?; echo "ERROR: bootstrap failed during ${stage} at line ${LINENO} (exit ${status})" >&2; exit "$status"' ERR

swap_tmp=""
key_tmp=""
docker_tmp=""
cleanup() {
    local temporary_file
    for temporary_file in "$swap_tmp" "$key_tmp" "$docker_tmp"; do
        if [[ -n "$temporary_file" && -f "$temporary_file" ]]; then
            unlink "$temporary_file" || true
        fi
    done
}
trap cleanup EXIT

retry() {
    local attempt
    for attempt in 1 2 3; do
        if "$@"; then
            return 0
        fi
        if (( attempt < 3 )); then
            echo "Retrying ${stage} (attempt $((attempt + 1))/3)..." >&2
            sleep 10
        fi
    done
    return 1
}

echo "=== EC2 Docker host bootstrap started ==="
export DEBIAN_FRONTEND=noninteractive

stage="checking operating system"
# shellcheck source=/dev/null
source /etc/os-release
if [[ "${ID}" != ubuntu || "${VERSION_ID}" != 22.04 ]]; then
    echo "This bootstrap targets the Ubuntu 22.04 AMI selected by Terraform." >&2
    exit 1
fi

stage="configuring swap"
# t2.micro has 1 GiB RAM. Swap cushions short spikes; it is not extra RAM.
# Keep existing swap and never reformat an unknown file on a rerun.
swap_path=/var/car-service.swap
if [[ -z "$(swapon --show --noheadings)" ]]; then
    if [[ ! -e "$swap_path" && ! -L "$swap_path" ]]; then
        available_kib=$(df -Pk /var | awk 'NR == 2 {print $4}')
        if (( available_kib < 3 * 1024 * 1024 )); then
            echo "Need 3 GiB free on /var: 1 GiB swap plus 2 GiB headroom." >&2
            exit 1
        fi
        # A temporary file keeps an interrupted allocation from becoming swap.
        swap_tmp=$(mktemp /var/car-service.swap.XXXXXX)
        dd if=/dev/zero of="$swap_tmp" bs=1M count=1024 status=none
        chmod 0600 "$swap_tmp"
        mkswap "$swap_tmp"
        mv "$swap_tmp" "$swap_path"
    fi
    if [[ ! -f "$swap_path" || -L "$swap_path" ]] || \
        [[ "$(blkid -p -s TYPE -o value "$swap_path")" != swap ]]; then
        echo "Refusing to use unrecognized swap file: $swap_path" >&2
        exit 1
    fi
    chmod 0600 "$swap_path"
    swapon "$swap_path"
fi
if swapon --show=NAME --noheadings | grep -Fxq "$swap_path"; then
    if ! awk -v path="$swap_path" '$1 == path && $3 == "swap" {found=1} END {exit !found}' /etc/fstab; then
        printf '%s none swap sw 0 0\n' "$swap_path" >> /etc/fstab
    fi
    printf 'vm.swappiness=10\n' > /etc/sysctl.d/90-car-service-swap.conf
    sysctl -p /etc/sysctl.d/90-car-service-swap.conf
fi

# APT can bootstrap networking dependencies itself. Bound retries, repository
# timeouts and lock waits rather than waiting forever for a curl probe.
apt_options=(
    -o Acquire::Retries=3
    -o Acquire::http::Timeout=30
    -o Acquire::https::Timeout=30
    -o DPkg::Lock::Timeout=120
    -o APT::Update::Error-Mode=any
)
stage="installing repository prerequisites"
retry apt-get "${apt_options[@]}" update
retry apt-get "${apt_options[@]}" install -y --no-install-recommends ca-certificates curl jq

stage="adding Docker repository"
install -m 0755 -d /etc/apt/keyrings
key_tmp=$(mktemp /etc/apt/keyrings/docker.asc.XXXXXX)
retry curl -fsSL --connect-timeout 10 --max-time 60 \
    https://download.docker.com/linux/ubuntu/gpg -o "$key_tmp"
chmod 0644 "$key_tmp"
mv "$key_tmp" /etc/apt/keyrings/docker.asc
printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu %s stable\n' \
    "$(dpkg --print-architecture)" "$VERSION_CODENAME" \
    > /etc/apt/sources.list.d/docker.list

stage="installing Docker"
retry apt-get "${apt_options[@]}" update
retry apt-get "${apt_options[@]}" install -y --no-install-recommends \
    docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
apt-get clean

stage="configuring Docker for a small host"
install -m 0755 -d /etc/docker
docker_tmp=$(mktemp /etc/docker/daemon.json.XXXXXX)
# Merge defaults without overriding existing operator settings on a rerun.
# Prefer pulling CI-built images: compiling on t2.micro consumes CPU credits.
defaults='{"log-driver":"local","log-opts":{"max-size":"10m","max-file":"3"},"max-concurrent-downloads":1,"max-concurrent-uploads":1}'
if [[ -e /etc/docker/daemon.json ]]; then
    # Log options belong to the selected driver; preserve an existing choice.
    jq --argjson defaults "$defaults" '
        if type != "object" then error("Docker config must be an object") else . end
        | if has("log-driver") or has("log-opts")
          then ($defaults | del(."log-driver", ."log-opts")) * .
          else $defaults * . end
    ' /etc/docker/daemon.json > "$docker_tmp"
else
    printf '%s\n' "$defaults" | jq . > "$docker_tmp"
fi
dockerd --validate --config-file "$docker_tmp"
docker_config_changed=false
if ! cmp -s "$docker_tmp" /etc/docker/daemon.json; then
    if [[ -e /etc/docker/daemon.json ]]; then
        chmod --reference=/etc/docker/daemon.json "$docker_tmp"
    else
        chmod 0644 "$docker_tmp"
    fi
    mv "$docker_tmp" /etc/docker/daemon.json
    docker_config_changed=true
else
    unlink "$docker_tmp"
fi

stage="starting Docker"
systemctl enable docker
if [[ "$docker_config_changed" == true ]]; then
    systemctl restart docker
else
    systemctl start docker
fi

stage="configuring ubuntu user"
# Docker group membership grants root-equivalent access; restrict SSH access.
usermod -aG docker ubuntu

stage="verifying Docker daemon"
systemctl is-active --quiet docker
retry timeout 20 docker info >/dev/null
docker --version
docker compose version
docker buildx version
free -m
echo "=== Docker host ready; deploy the app with its environment and migrations separately ==="
