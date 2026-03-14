set -e

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

BRIDGE={{BRIDGE}}
IFACE={{IFACE}}

log "Starting cleanup for bridge $BRIDGE"

# Detach interface from bridge
log "Detaching interface $IFACE from bridge $BRIDGE"
nsenter -t 1 -n -m -- ip link set "$IFACE" nomaster
log "Interface $IFACE detached successfully"

# Delete the bridge
log "Deleting bridge $BRIDGE"
nsenter -t 1 -n -m -- ip link delete "$BRIDGE"
log "Bridge $BRIDGE deleted successfully"

log "Cleanup completed successfully"
