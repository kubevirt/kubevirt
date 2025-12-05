
script_dir="$(dirname "$0")"
source "$script_dir/common.env"

create_vm() {
  "$script_dir/create-vm.sh" \
    -g "$RESOURCE_GROUP" \
    -S "$SUBSCRIPTION" \
    -v "$VM_NAME" \
    -s "$SIZE" \
    -l "$LOCATION" \
    -k "$SSH_KEY_PATH" \
    -i "$MSHV_IMAGE"
}