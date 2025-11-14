# Mise Docker Plugin

Custom mise plugin for installing Docker CLI and Engine on Linux systems.

## Installation

### Option 1: Link the plugin and install via mise

```bash
# Link the plugin
mise plugin link docker .mise/plugins/docker

# Install Docker
mise install docker@latest
```

### Option 2: Use the mise task

```bash
# Run the installation task
mise run install-docker
```

## Supported Operating Systems

- Ubuntu
- Debian
- Fedora
- CentOS/RHEL
- Rocky Linux
- AlmaLinux

## What Gets Installed

- Docker Engine (`docker-ce`)
- Docker CLI (`docker-ce-cli`)
- Containerd (`containerd.io`)
- Docker Buildx plugin
- Docker Compose plugin

## Post-Installation

After installation, the script will:

1. Start the Docker service
2. Enable Docker to start on boot
3. Add your user to the `docker` group (requires logout/login to take effect)

## Verification

After installation, verify Docker is working:

```bash
docker --version
docker compose version
sudo docker run hello-world
```

## Notes

- This plugin requires `sudo` privileges
- You may need to log out and back in after installation for the docker group membership to take effect
- The plugin installs the latest stable version from Docker's official repositories
