set -e

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

BRIDGE={{BRIDGE}}
IFACE={{IFACE}}

log "Starting bridge setup for $BRIDGE with interface $IFACE"

# Create the bridge
log "Creating bridge $BRIDGE"
nsenter -t 1 -n -m -- ip link add "$BRIDGE" type bridge
log "Bridge $BRIDGE created successfully"

# Check if interface already has a master
current_master=$(nsenter -t 1 -n -m -- ip link show "$IFACE" | grep -oP 'master \K\S+' || true)
if [ -n "$current_master" ]; then
    log "WARNING: Interface $IFACE already has master: $current_master"
    log "Changing master from $current_master to $BRIDGE"
fi

# Set the interface as a slave to the bridge
log "Attaching interface $IFACE to bridge $BRIDGE"
nsenter -t 1 -n -m -- ip link set "$IFACE" master "$BRIDGE"
log "Interface $IFACE attached to bridge $BRIDGE successfully"

# Bring up the bridge
log "Bringing up bridge $BRIDGE"
nsenter -t 1 -n -m -- ip link set "$BRIDGE" up
log "Bridge $BRIDGE is up and running"

log "Bridge setup completed successfully"
tail -f /dev/null
