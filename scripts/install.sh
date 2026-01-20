#!/bin/bash
#
#  ██████╗ ██████╗  ██████╗      ██╗███████╗ ██████╗████████╗
#  ██╔══██╗██╔══██╗██╔═══██╗     ██║██╔════╝██╔════╝╚══██╔══╝
#  ██████╔╝██████╔╝██║   ██║     ██║█████╗  ██║        ██║
#  ██╔═══╝ ██╔══██╗██║   ██║██   ██║██╔══╝  ██║        ██║
#  ██║     ██║  ██║╚██████╔╝╚█████╔╝███████╗╚██████╗   ██║
#  ╚═╝     ╚═╝  ╚═╝ ╚═════╝  ╚════╝ ╚══════╝ ╚═════╝   ╚═╝
#                     H O R I Z O N
#
# ProjectHorizon Installation Script
# https://github.com/hiratazx/projecthorizon
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Variables
VERSION="0.1.0"
INSTALL_PREFIX="/usr/local"
CONFIG_DIR="/etc/projecthorizon"
DATA_DIR="/var/lib/projecthorizon"
SERVICE_NAME="projecthorizon"
GITHUB_REPO="hiratazx/projecthorizon"
ARCH=$(uname -m)

# Helper functions
print_banner() {
    echo -e "${CYAN}"
    cat << "EOF"
  ██████╗ ██████╗  ██████╗      ██╗███████╗ ██████╗████████╗
  ██╔══██╗██╔══██╗██╔═══██╗     ██║██╔════╝██╔════╝╚══██╔══╝
  ██████╔╝██████╔╝██║   ██║     ██║█████╗  ██║        ██║
  ██╔═══╝ ██╔══██╗██║   ██║██   ██║██╔══╝  ██║        ██║
  ██║     ██║  ██║╚██████╔╝╚█████╔╝███████╗╚██████╗   ██║
  ╚═╝     ╚═╝  ╚═╝ ╚═════╝  ╚════╝ ╚══════╝ ╚═════╝   ╚═╝
                     H O R I Z O N
EOF
    echo -e "${NC}"
    echo -e "${GREEN}NAS Dashboard for Existing Linux Systems${NC}"
    echo ""
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[✗]${NC} $1"
    exit 1
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        error "Please run as root (sudo bash install.sh)"
    fi
}

# Check system architecture
check_arch() {
    info "Checking system architecture..."
    case ${ARCH} in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l|armhf)
            ARCH="arm"
            ;;
        *)
            error "Unsupported architecture: ${ARCH}"
            ;;
    esac
    success "Architecture: ${ARCH}"
}

# Check OS
check_os() {
    info "Checking operating system..."
    if [ ! -f /etc/os-release ]; then
        error "Cannot detect OS. /etc/os-release not found."
    fi
    
    . /etc/os-release
    OS_ID="${ID}"
    OS_VERSION="${VERSION_ID}"
    
    case ${OS_ID} in
        ubuntu|debian|raspbian)
            success "Detected: ${PRETTY_NAME}"
            ;;
        centos|rhel|fedora|rocky|almalinux)
            success "Detected: ${PRETTY_NAME}"
            ;;
        *)
            warn "Untested OS: ${PRETTY_NAME}. Proceeding anyway..."
            ;;
    esac
}

# Check and install Docker
check_docker() {
    info "Checking Docker installation..."
    
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version | awk '{print $3}' | tr -d ',')
        success "Docker installed: ${DOCKER_VERSION}"
    else
        warn "Docker not found. Installing Docker..."
        install_docker
    fi
    
    # Check if Docker is running
    if ! systemctl is-active --quiet docker; then
        info "Starting Docker service..."
        systemctl start docker
        systemctl enable docker
    fi
    success "Docker is running"
}

install_docker() {
    info "Installing Docker..."
    curl -fsSL https://get.docker.com | bash
    success "Docker installed successfully"
}

# Create directories
create_directories() {
    info "Creating directories..."
    mkdir -p ${CONFIG_DIR}/config
    mkdir -p ${DATA_DIR}/www
    success "Directories created"
}

# Download and install binary
install_binary() {
    info "Downloading ProjectHorizon v${VERSION}..."
    
    TMP_DIR=$(mktemp -d)
    cd ${TMP_DIR}
    
    # For local development, copy from build directory
    if [ -f "./build/horizon" ]; then
        cp ./build/horizon ${INSTALL_PREFIX}/bin/
    else
        # Download from releases (placeholder URL)
        DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/horizon-${VERSION}-linux-${ARCH}.tar.gz"
        
        if command -v wget &> /dev/null; then
            wget -q --show-progress "${DOWNLOAD_URL}" -O horizon.tar.gz || error "Download failed"
        elif command -v curl &> /dev/null; then
            curl -fsSL "${DOWNLOAD_URL}" -o horizon.tar.gz || error "Download failed"
        else
            error "Neither wget nor curl found"
        fi
        
        tar -xzf horizon.tar.gz
        cp sysroot/usr/local/bin/horizon ${INSTALL_PREFIX}/bin/
    fi
    
    chmod +x ${INSTALL_PREFIX}/bin/horizon
    success "Binary installed to ${INSTALL_PREFIX}/bin/horizon"
    
    cd - > /dev/null
    rm -rf ${TMP_DIR}
}

# Install default configuration
install_config() {
    info "Installing configuration..."
    
    if [ ! -f ${CONFIG_DIR}/config/storage.json ]; then
        cat > ${CONFIG_DIR}/config/storage.json << 'EOF'
{
  "volumes": [
    {
      "name": "Main Storage",
      "hostPath": "/media",
      "mountPath": "/media",
      "mode": "rw"
    }
  ],
  "settings": {
    "defaultPath": "/media"
  }
}
EOF
    fi
    success "Configuration installed"
}

# Install systemd service
install_service() {
    info "Installing systemd service..."
    
    cat > /etc/systemd/system/${SERVICE_NAME}.service << 'EOF'
[Unit]
Description=ProjectHorizon NAS Dashboard
Documentation=https://github.com/hiratazx/projecthorizon
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/var/lib/projecthorizon
ExecStart=/usr/local/bin/horizon
Restart=always
RestartSec=10
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=projecthorizon
Environment="PORT=8080"
Environment="GIN_MODE=release"

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable ${SERVICE_NAME}
    success "Service installed and enabled"
}

# Start service
start_service() {
    info "Starting ProjectHorizon..."
    systemctl start ${SERVICE_NAME}
    
    # Wait for service to start
    sleep 2
    
    if systemctl is-active --quiet ${SERVICE_NAME}; then
        success "ProjectHorizon started successfully"
    else
        error "Failed to start ProjectHorizon. Check: journalctl -u ${SERVICE_NAME}"
    fi
}

# Get IP addresses
get_ips() {
    ip -4 addr show | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | grep -v '127.0.0.1' | head -5
}

# Show success message
show_success() {
    PORT=8080
    echo ""
    echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}   ProjectHorizon has been installed successfully!${NC}"
    echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "   Dashboard is available at:"
    for IP in $(get_ips); do
        echo -e "   ${CYAN}➜ http://${IP}:${PORT}${NC}"
    done
    echo ""
    echo -e "   Useful commands:"
    echo -e "   ${YELLOW}systemctl status ${SERVICE_NAME}${NC}  - Check service status"
    echo -e "   ${YELLOW}journalctl -u ${SERVICE_NAME}${NC}     - View logs"
    echo -e "   ${YELLOW}horizon-uninstall${NC}                 - Uninstall"
    echo ""
    echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
}

# Main installation flow
main() {
    print_banner
    
    check_root
    check_arch
    check_os
    check_docker
    create_directories
    install_binary
    install_config
    install_service
    start_service
    show_success
}

# Run main
main "$@"
