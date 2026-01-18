#!/bin/bash
#
# ProjectHorizon Uninstall Script
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

SERVICE_NAME="projecthorizon"
INSTALL_PREFIX="/usr/local"
CONFIG_DIR="/etc/projecthorizon"
DATA_DIR="/var/lib/projecthorizon"

info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}[✗]${NC} Please run as root (sudo horizon-uninstall)"
    exit 1
fi

echo -e "${CYAN}"
cat << "EOF"
  ██████╗ ██████╗  ██████╗      ██╗███████╗ ██████╗████████╗
  ██╔══██╗██╔══██╗██╔═══██╗     ██║██╔════╝██╔════╝╚══██╔══╝
  ██████╔╝██████╔╝██║   ██║     ██║█████╗  ██║        ██║
  ██╔═══╝ ██╔══██╗██║   ██║██   ██║██╔══╝  ██║        ██║
  ██║     ██║  ██║╚██████╔╝╚█████╔╝███████╗╚██████╗   ██║
  ╚═╝     ╚═╝  ╚═╝ ╚═════╝  ╚════╝ ╚══════╝ ╚═════╝   ╚═╝
                     H O R I Z O N
                    
                    Uninstall Script
EOF
echo -e "${NC}"

echo ""
read -p "Are you sure you want to uninstall ProjectHorizon? [y/N] " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Uninstall cancelled."
    exit 0
fi

echo ""

# Stop and disable service
info "Stopping service..."
systemctl stop ${SERVICE_NAME} 2>/dev/null || true
systemctl disable ${SERVICE_NAME} 2>/dev/null || true
success "Service stopped"

# Remove service file
info "Removing service file..."
rm -f /etc/systemd/system/${SERVICE_NAME}.service
systemctl daemon-reload
success "Service file removed"

# Remove binary
info "Removing binary..."
rm -f ${INSTALL_PREFIX}/bin/horizon
rm -f ${INSTALL_PREFIX}/bin/horizon-uninstall
success "Binary removed"

# Ask about config and data
echo ""
read -p "Remove configuration files? (${CONFIG_DIR}) [y/N] " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf ${CONFIG_DIR}
    success "Configuration removed"
else
    warn "Configuration preserved at ${CONFIG_DIR}"
fi

read -p "Remove data files? (${DATA_DIR}) [y/N] " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf ${DATA_DIR}
    success "Data removed"
else
    warn "Data preserved at ${DATA_DIR}"
fi

echo ""
echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}   ProjectHorizon has been uninstalled.${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
echo ""
