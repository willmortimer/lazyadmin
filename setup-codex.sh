#!/usr/bin/env bash
# Lazyadmin Codex/AI Dev Container Setup Script
# Optimized for container environments without systemd
# Run from /workspace/lazyadmin after cloning the repository

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}ℹ${NC} $1"; }
log_success() { echo -e "${GREEN}✓${NC} $1"; }
log_warn() { echo -e "${YELLOW}⚠${NC} $1"; }
log_error() { echo -e "${RED}✗${NC} $1"; }

# Detect environment
IS_CONTAINER=false
if [ -f /.dockerenv ] || [ -n "${CODESPACE_NAME:-}" ] || [ -n "${GITPOD_WORKSPACE_ID:-}" ]; then
    IS_CONTAINER=true
fi

log_info "Setting up lazyadmin development environment..."
log_info "Container environment: $IS_CONTAINER"

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
    VER=$VERSION_ID
else
    log_error "Cannot detect OS. Supported: Ubuntu, Debian, Fedora, CentOS/RHEL"
    exit 1
fi

log_info "Detected OS: $OS $VER"

# Ensure mise is available and activated
setup_mise() {
    log_info "Setting up mise..."
    
    if command -v mise &> /dev/null; then
        MISE_PATH=$(which mise)
        log_success "mise found at $MISE_PATH"
    else
        log_info "Installing mise..."
        curl -fsSL https://mise.run | sh
        export PATH="$HOME/.local/bin:$PATH"
        MISE_PATH="$HOME/.local/bin/mise"
    fi
    
    # Activate mise for current shell
    eval "$($MISE_PATH activate bash)" 2>/dev/null || true
    
    # Verify mise works
    if ! mise --version &>/dev/null; then
        log_error "mise activation failed"
        exit 1
    fi
    
    log_success "mise ready"
}

# Install system dependencies
install_dependencies() {
    log_info "Installing system dependencies..."
    
    case $OS in
        ubuntu|debian)
            sudo apt-get update -qq
            sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
                curl \
                git \
                build-essential \
                ca-certificates \
                gnupg \
                lsb-release \
                apt-transport-https \
                > /dev/null 2>&1
            ;;
        fedora)
            sudo dnf install -y -q \
                curl \
                git \
                gcc \
                gcc-c++ \
                make \
                ca-certificates \
                > /dev/null 2>&1
            ;;
        centos|rhel|rocky|almalinux)
            sudo yum install -y -q \
                curl \
                git \
                gcc \
                gcc-c++ \
                make \
                ca-certificates \
                > /dev/null 2>&1
            ;;
    esac
    
    log_success "System dependencies installed"
}

# Setup Docker for container environment
setup_docker_container() {
    log_info "Configuring Docker for container environment..."
    
    # Start dockerd if not running
    if ! pgrep -x dockerd > /dev/null 2>&1; then
        log_info "Starting Docker daemon..."
        sudo dockerd > /tmp/dockerd.log 2>&1 &
        DOCKERD_PID=$!
        
        # Wait for Docker to be ready (max 30 seconds)
        for i in {1..30}; do
            if docker info &>/dev/null || sudo docker info &>/dev/null; then
                log_success "Docker daemon ready"
                return 0
            fi
            sleep 1
        done
        
        log_warn "Docker daemon may still be starting in background"
    else
        log_success "Docker daemon already running"
    fi
    
    # Ensure Docker socket permissions
    if [ -S /var/run/docker.sock ]; then
        sudo chmod 666 /var/run/docker.sock 2>/dev/null || true
    fi
}

# Main setup flow
main() {
    setup_mise
    
    install_dependencies
    
    # Trust mise configuration
    log_info "Trusting mise configuration..."
    mise trust
    log_success "mise configuration trusted"
    
    # Link Docker plugin
    log_info "Linking Docker plugin..."
    if [ ! -d ".mise/plugins/docker" ]; then
        log_error "Docker plugin not found at .mise/plugins/docker"
        exit 1
    fi
    
    mise plugin link docker .mise/plugins/docker 2>/dev/null || true
    log_success "Docker plugin linked"
    
    # Install Docker
    log_info "Installing Docker..."
    if mise install docker@latest 2>&1 | grep -q "ERROR"; then
        log_warn "Docker installation had errors, but packages may be installed"
    else
        log_success "Docker installation completed"
    fi
    
    # Setup Docker for container environment
    if [ "$IS_CONTAINER" = "true" ]; then
        setup_docker_container
    fi
    
    # Install development tools
    log_info "Installing development tools from .mise.toml..."
    mise install
    log_success "Development tools installed"
    
    # Verify installations
    log_info "Verifying installations..."
    echo ""
    
    if mise --version &>/dev/null; then
        log_success "mise: $(mise --version | head -1)"
    fi
    
    if docker --version &>/dev/null || sudo docker --version &>/dev/null; then
        DOCKER_VER=$(docker --version 2>/dev/null || sudo docker --version 2>/dev/null)
        log_success "Docker: $DOCKER_VER"
        
        # Test Docker connectivity
        if docker info &>/dev/null || sudo docker info &>/dev/null; then
            log_success "Docker daemon is accessible"
        else
            log_warn "Docker CLI installed but daemon not accessible"
            log_info "In containers, you may need to start dockerd manually: sudo dockerd &"
        fi
    else
        log_warn "Docker not found"
    fi
    
    if go version &>/dev/null; then
        log_success "Go: $(go version)"
    fi
    
    if node --version &>/dev/null; then
        log_success "Node: $(node --version)"
    fi
    
    echo ""
    log_success "Setup complete!"
    echo ""
    log_info "Next steps:"
    echo "  • Start dev stack: mise run dev"
    echo "  • Run TUI: mise run tui"
    echo "  • If Docker isn't working: sudo dockerd &"
}

main "$@"

