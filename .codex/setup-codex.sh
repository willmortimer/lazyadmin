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

# Trust mise configuration immediately to prevent warnings
# Note: This warning may still appear during Codex's language runtime setup
# (before this script runs), but we trust it here to prevent warnings in our script
if command -v mise &> /dev/null; then
    log_info "Trusting mise configuration..."
    # Try to trust, but don't fail if it doesn't work (mise may not be activated yet)
    if mise trust 2>/dev/null; then
        log_success "mise configuration trusted (early)"
    else
        # Will trust again after mise activation
        log_info "Will trust mise config after activation"
    fi
fi

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
    
    # Check if dockerd is already running
    if pgrep -x dockerd > /dev/null 2>&1; then
        log_success "Docker daemon already running"
        # Verify it's accessible
        if docker info &>/dev/null 2>&1 || sudo docker info &>/dev/null 2>&1; then
            log_success "Docker daemon is accessible"
            return 0
        else
            log_warn "Docker daemon process found but not accessible"
        fi
    fi
    
    # Check if we have necessary capabilities for Docker-in-Docker
    log_info "Checking Docker-in-Docker requirements..."
    
    # Ensure Docker socket directory exists
    sudo mkdir -p /var/run
    DOCKERD_LOG="/tmp/dockerd.log"
    
    # Try to start dockerd with rootless mode first (if supported)
    log_info "Attempting to start Docker daemon..."
    log_info "Note: Docker-in-Docker requires privileged mode or specific capabilities"
    
    # Start dockerd with explicit socket path and log to file
    # Use vfs storage driver to avoid overlayfs issues in containers
    sudo dockerd \
        --host=unix:///var/run/docker.sock \
        --iptables=false \
        --ip-forward=false \
        --storage-driver=vfs \
        > "$DOCKERD_LOG" 2>&1 &
    
    DOCKERD_PID=$!
    
    # Give dockerd a moment to start
    log_info "Waiting for Docker daemon to initialize..."
    sleep 3
    
    # Check if process is still running
    if ! kill -0 "$DOCKERD_PID" 2>/dev/null; then
        log_warn "Docker daemon failed to start (this is expected in unprivileged containers)"
        if [ -f "$DOCKERD_LOG" ]; then
            log_info "Docker daemon log (last 15 lines):"
            tail -15 "$DOCKERD_LOG" | sed 's/^/  /'
            log_info ""
            log_info "Common issues:"
            log_info "  • Container needs --privileged flag or NET_ADMIN capability"
            log_info "  • Or use Docker socket mounting instead of Docker-in-Docker"
        fi
        return 1
    fi
    
    # Wait for Docker to be ready (max 30 seconds)
    log_info "Waiting for Docker daemon to be ready..."
    for i in {1..30}; do
        if docker info &>/dev/null 2>&1; then
            log_success "Docker daemon ready (accessible without sudo)"
            return 0
        elif sudo docker info &>/dev/null 2>&1; then
            log_success "Docker daemon ready (requires sudo)"
            # Fix socket permissions for current user
            if [ -S /var/run/docker.sock ]; then
                sudo chmod 666 /var/run/docker.sock 2>/dev/null || true
                # Retry without sudo
                sleep 1
                if docker info &>/dev/null 2>&1; then
                    log_success "Docker daemon now accessible without sudo"
                    return 0
                fi
            fi
            return 0
        fi
        
        # Check if dockerd is still running
        if ! kill -0 "$DOCKERD_PID" 2>/dev/null; then
            log_error "Docker daemon process died"
            if [ -f "$DOCKERD_LOG" ]; then
                log_info "Docker daemon log (last 20 lines):"
                tail -20 "$DOCKERD_LOG" | sed 's/^/  /'
            fi
            return 1
        fi
        
        # Show progress every 5 seconds
        if [ $((i % 5)) -eq 0 ]; then
            log_info "Still waiting... (${i}/30 seconds)"
        fi
        
        sleep 1
    done
    
    log_warn "Docker daemon may still be starting in background (PID: $DOCKERD_PID)"
    if [ -f "$DOCKERD_LOG" ]; then
        log_info "Check logs: tail -f $DOCKERD_LOG"
        log_info "Last 10 lines:"
        tail -10 "$DOCKERD_LOG" | sed 's/^/  /'
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
    
    # Trust mise configuration (already done at start, but ensure it's trusted)
    log_info "Ensuring mise configuration is trusted..."
    mise trust 2>/dev/null || true
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
        if ! setup_docker_container; then
            log_warn "Docker daemon setup failed, but continuing setup..."
            log_info "Docker CLI is installed, but daemon needs to be started manually"
            log_info "Or mount Docker socket: -v /var/run/docker.sock:/var/run/docker.sock"
        fi
    fi
    
    # Install development tools
    log_info "Installing development tools from .mise.toml..."
    log_info "This may take a few minutes..."
    MISE_INSTALL_LOG="/tmp/mise-install.log"
    if mise install > "$MISE_INSTALL_LOG" 2>&1; then
        log_success "Development tools installed"
    else
        INSTALL_EXIT=$?
        log_warn "mise install exited with code $INSTALL_EXIT"
        if grep -q "ERROR" "$MISE_INSTALL_LOG" 2>/dev/null; then
            log_warn "Some tools may have installation errors, check $MISE_INSTALL_LOG"
        else
            log_info "Installation completed (some warnings may be normal)"
        fi
    fi
    
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
        if docker info &>/dev/null 2>&1; then
            log_success "Docker daemon is accessible"
        elif sudo docker info &>/dev/null 2>&1; then
            log_warn "Docker daemon requires sudo"
            log_info "Fixing socket permissions..."
            if [ -S /var/run/docker.sock ]; then
                sudo chmod 666 /var/run/docker.sock 2>/dev/null || true
                if docker info &>/dev/null 2>&1; then
                    log_success "Docker daemon now accessible"
                else
                    log_warn "Docker CLI installed but daemon not accessible"
                    log_info "Check if dockerd is running: pgrep -x dockerd"
                    log_info "View logs: tail -f /tmp/dockerd.log"
                fi
            fi
        else
            log_warn "Docker CLI installed but daemon not accessible"
            if [ "$IS_CONTAINER" = "true" ]; then
                log_info "In containers, dockerd may need manual startup:"
                log_info "  sudo dockerd --host=unix:///var/run/docker.sock --iptables=false &"
                log_info "  Check logs: tail -f /tmp/dockerd.log"
            fi
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
    
    # Build CLI wrapper
    log_info "Building lazyadmin CLI wrapper..."
    if ! command -v go &>/dev/null; then
        log_warn "Go not found in PATH, skipping CLI wrapper build"
        log_info "Build it later with: go build -o lazyadmin ./cmd/lazyadmin-cli"
    else
        BUILD_LOG="/tmp/lazyadmin-build.log"
        log_info "Using Go: $(go version)"
        if go build -o lazyadmin ./cmd/lazyadmin-cli > "$BUILD_LOG" 2>&1; then
            log_success "CLI wrapper built: ./lazyadmin"
            rm -f "$BUILD_LOG"
        else
            BUILD_EXIT=$?
            log_warn "Failed to build CLI wrapper (exit code: $BUILD_EXIT)"
            if [ -f "$BUILD_LOG" ] && [ -s "$BUILD_LOG" ]; then
                log_info "Build errors:"
                head -20 "$BUILD_LOG" | sed 's/^/  /'
                if [ "$(wc -l < "$BUILD_LOG" 2>/dev/null || echo 0)" -gt 20 ]; then
                    log_info "  ... (see $BUILD_LOG for full output)"
                fi
            fi
            log_info "You can build it later with: go build -o lazyadmin ./cmd/lazyadmin-cli"
        fi
    fi
    
    echo ""
    log_success "Setup complete!"
    echo ""
    log_info "Next steps:"
    echo "  • Start dev stack: docker compose up backend postgres caddy -d"
    echo "  • Run TUI: ./lazyadmin (or: docker compose run --rm lazyadmin)"
    echo "  • Or run TUI natively: mise run tui"
    if [ "$IS_CONTAINER" = "true" ]; then
        echo "  • If Docker isn't working, check: tail -f /tmp/dockerd.log"
    fi
}

main "$@"

