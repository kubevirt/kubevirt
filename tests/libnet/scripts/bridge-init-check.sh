set -e

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

error() {
    echo "$*" > /dev/termination-log
    log "ERROR: $*"
    exit 1
}

BRIDGE={{BRIDGE}}
IFACE={{IFACE}}

log "Starting pre-flight checks for bridge setup"

# Check if the interface exists
log "Checking if interface $IFACE exists"
if ! nsenter -t 1 -n -m -- ip link show "$IFACE" > /dev/null 2>&1; then
    error "interface $IFACE not found on node"
fi
log "Interface $IFACE exists"

# Check if the bridge already exists
log "Checking if bridge $BRIDGE already exists"
if nsenter -t 1 -n -m -- ip link show "$BRIDGE" > /dev/null 2>&1; then
    error "bridge $BRIDGE already exists on node"
fi
log "Bridge $BRIDGE does not exist"

log "Pre-flight checks completed successfully"
