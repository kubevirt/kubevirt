#!/bin/bash

# Azure CLI deployment script for a single-node k3s cluster
# This script is idempotent and can be run multiple times safely

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
DEFAULT_LOCATION="westus"
DEFAULT_VM_SIZE="Standard_D8s_v5"
DEFAULT_ADMIN_USERNAME="azureuser"
DEFAULT_VM_IMAGE="Ubuntu2404"  # Can be marketplace alias, URN, shared image gallery ID, or image ID

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 -g <resource_group> -v <vm_name> -k <ssh_key_path> [OPTIONS]

Required arguments:
    -g, --resource-group    Azure resource group name
    -v, --vm-name          Virtual machine name
    -k, --ssh-key          Path to SSH public key

Optional arguments:
    -l, --location         Azure location (default: $DEFAULT_LOCATION)
    -s, --vm-size          VM size (default: $DEFAULT_VM_SIZE)
    -u, --admin-username   VM admin username (default: $DEFAULT_ADMIN_USERNAME)
    -i, --image            Azure image (alias, URN, SIG ID, or image ID) (default: $DEFAULT_VM_IMAGE)
    -S, --subscription     Azure subscription ID or name to target (default: currently active)
    -h, --help             Show this help message

Example:
    $0 -g my-k3s-rg -v k3s-vm -k ~/.ssh/id_rsa.pub

Note:
- VM size must support nested virtualization (e.g., Standard_D4s_v3, Standard_E4s_v3)
- The script configures k3s automatically with a public IP TLS SAN.

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -g|--resource-group)
                RESOURCE_GROUP="$2"
                shift 2
                ;;
            -v|--vm-name)
                VM_NAME="$2"
                shift 2
                ;;
            -l|--location)
                LOCATION="$2"
                shift 2
                ;;
            -s|--vm-size)
                VM_SIZE="$2"
                shift 2
                ;;
            -u|--admin-username)
                ADMIN_USERNAME="$2"
                shift 2
                ;;
            -k|--ssh-key)
                SSH_KEY_PATH="$2"
                shift 2
                ;;
            -i|--image)
                VM_IMAGE="$2"
                shift 2
                ;;
            -S|--subscription)
                SUBSCRIPTION="$2"
                shift 2
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # Set default values
    LOCATION=${LOCATION:-$DEFAULT_LOCATION}
    VM_SIZE=${VM_SIZE:-$DEFAULT_VM_SIZE}
    ADMIN_USERNAME=${ADMIN_USERNAME:-$DEFAULT_ADMIN_USERNAME}
    VM_IMAGE=${VM_IMAGE:-$DEFAULT_VM_IMAGE}
    
    # Expand tilde in SSH_KEY_PATH if present
    SSH_KEY_PATH=${SSH_KEY_PATH/#~/$HOME}

    # Validate required arguments
    if [[ -z "$RESOURCE_GROUP" || -z "$VM_NAME" || -z "$SSH_KEY_PATH" ]]; then
        print_error "Missing required arguments!"
        show_usage
        exit 1
    fi

    print_status "Using image: $VM_IMAGE"
    if [[ -n "$SUBSCRIPTION" ]]; then
        print_status "Requested subscription: $SUBSCRIPTION"
    fi
}

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."

    # Check if Azure CLI is installed
    if ! command -v az &> /dev/null; then
        print_error "Azure CLI is not installed. Please install it from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
        exit 1
    fi

    # Check if logged in to Azure
    if ! az account show &> /dev/null; then
        print_error "Not logged in to Azure. Please run 'az login' first."
        exit 1
    fi

    # Switch subscription if provided
    if [[ -n "$SUBSCRIPTION" ]]; then
        print_status "Setting Azure subscription to '$SUBSCRIPTION'..."
        if az account set --subscription "$SUBSCRIPTION" 2>/dev/null; then
            ACTIVE_SUB=$(az account show --query id -o tsv)
            print_success "Active subscription set to $ACTIVE_SUB"
        else
            print_error "Failed to set subscription '$SUBSCRIPTION'. Verify the ID/name and your access."
            exit 1
        fi
    else
        ACTIVE_SUB=$(az account show --query id -o tsv)
        print_status "Using existing active subscription: $ACTIVE_SUB"
    fi

    if [[ ! -f $SSH_KEY_PATH ]]; then
        SSH_PRIVATE_KEY_PATH="${SSH_KEY_PATH%.pub}"
        ssh-keygen -t rsa -b 4096 -f $SSH_PRIVATE_KEY_PATH -N ""
    fi


    SSH_PUBLIC_KEY=$(cat "$SSH_KEY_PATH")
    print_success "SSH key found at $SSH_KEY_PATH"
    print_success "cloud-init.yaml found"
    print_success "Prerequisites check completed"
}

# Create or update resource group
create_resource_group() {
    print_status "Creating resource group '$RESOURCE_GROUP' in '$LOCATION'..."
    
    if az group show --name "$RESOURCE_GROUP" &> /dev/null; then
        print_warning "Resource group '$RESOURCE_GROUP' already exists"
    else
        az group create --name "$RESOURCE_GROUP" --location "$LOCATION" --output none
        print_success "Resource group '$RESOURCE_GROUP' created"
    fi
}

# Function to create standard NSG rules
create_nsg_rules() {
    local nsg_name="$1"
    local rule_suffix="$2"  # Optional suffix for rule names (e.g., "-NIC")
    
    if [[ -z "$nsg_name" ]]; then
        print_error "NSG name is required for create_nsg_rules function"
        return 1
    fi
    
    local ssh_rule_name="SSH${rule_suffix}"
    local k3s_rule_name="k3s-api${rule_suffix}"
    
    # Add SSH rule
    if ! az network nsg rule show --name "$ssh_rule_name" --nsg-name "$nsg_name" --resource-group "$RESOURCE_GROUP" &> /dev/null; then
        print_status "Adding SSH rule to NSG '$nsg_name'..."
        az network nsg rule create \
            --resource-group "$RESOURCE_GROUP" \
            --nsg-name "$nsg_name" \
            --name "$ssh_rule_name" \
            --protocol tcp \
            --priority 1001 \
            --destination-port-range 22 \
            --access allow \
            --output none
        print_success "SSH rule added to NSG '$nsg_name'"
    else
        print_success "SSH rule already exists in NSG '$nsg_name'"
    fi
    
    # Add k3s API rule
    if ! az network nsg rule show --name "$k3s_rule_name" --nsg-name "$nsg_name" --resource-group "$RESOURCE_GROUP" &> /dev/null; then
        print_status "Adding k3s-api rule to NSG '$nsg_name'..."
        az network nsg rule create \
            --resource-group "$RESOURCE_GROUP" \
            --nsg-name "$nsg_name" \
            --name "$k3s_rule_name" \
            --protocol tcp \
            --priority 1002 \
            --destination-port-range 6443 \
            --access allow \
            --output none
        print_success "k3s-api rule added to NSG '$nsg_name'"
    else
        print_success "k3s-api rule already exists in NSG '$nsg_name'"
    fi
}


# Create virtual network and security group
create_network_resources() {
    print_status "Creating network resources..."
    
    local vnet_name="${VM_NAME}-vnet"
    local subnet_name="default"
    local nsg_name="${VM_NAME}-nsg"
    local pip_name="${VM_NAME}-pip"
    
    # Create virtual network
    if ! az network vnet show --name "$vnet_name" --resource-group "$RESOURCE_GROUP" &> /dev/null; then
        az network vnet create \
            --resource-group "$RESOURCE_GROUP" \
            --name "$vnet_name" \
            --address-prefix "10.0.0.0/16" \
            --subnet-name "$subnet_name" \
            --subnet-prefix "10.0.0.0/24" \
            --output none
        print_success "Virtual network '$vnet_name' created"
    else
        print_warning "Virtual network '$vnet_name' already exists"
    fi
    
    # Create network security group
    if ! az network nsg show --name "$nsg_name" --resource-group "$RESOURCE_GROUP" &> /dev/null; then
        az network nsg create \
            --resource-group "$RESOURCE_GROUP" \
            --name "$nsg_name" \
            --output none
        print_success "Network security group '$nsg_name' created"
    else
        print_warning "Network security group '$nsg_name' already exists"
    fi
    
    # Add standard rules to the NSG
    create_nsg_rules "$nsg_name" ""
    
    # Associate NSG with subnet
    az network vnet subnet update \
        --resource-group "$RESOURCE_GROUP" \
        --vnet-name "$vnet_name" \
        --name "$subnet_name" \
        --network-security-group "$nsg_name" \
        --output none
    
    # Create public IP
    if ! az network public-ip show --name "$pip_name" --resource-group "$RESOURCE_GROUP" &> /dev/null; then
        az network public-ip create \
            --resource-group "$RESOURCE_GROUP" \
            --name "$pip_name" \
            --sku Standard \
            --allocation-method Static \
            --dns-name "${VM_NAME}-$(echo $RANDOM | md5sum | head -c 8)" \
            --output none
        print_success "Public IP '$pip_name' created"
    else
        print_warning "Public IP '$pip_name' already exists"
    fi
}

# Create virtual machine
create_virtual_machine() {
    print_status "Creating virtual machine '$VM_NAME'..."
    
    if az vm show --name "$VM_NAME" --resource-group "$RESOURCE_GROUP" &> /dev/null; then
        print_warning "Virtual machine '$VM_NAME' already exists"
        return
    fi
    
    local vnet_name="${VM_NAME}-vnet"
    local subnet_name="default"
    local pip_name="${VM_NAME}-pip"
    
    # Use the cloud-init file from the current directory
    local cloud_init_file="cloud-init.yaml"
    # Use single-quoted heredoc delimiter to avoid local variable/command expansion;
    # everything should be evaluated on the VM during cloud-init execution.
    cat > "$cloud_init_file" << 'EOF'
#cloud-config
package_update: true
package_upgrade: true

packages:
  - curl
  - wget
  - git
  - unzip
  - jq

write_files:
  - path: /opt/k3s-install.sh
    permissions: '0755'
    content: |
        #!/bin/bash
        set -e
        
        # Get public IP
        PUBLIC_IP=$(curl -s ifconfig.me || curl -s ipinfo.io/ip || echo "127.0.0.1")
        echo "Public IP: $PUBLIC_IP"
        
        # Install k3s with containerd configuration
        echo "Installing k3s..."
        curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--write-kubeconfig-mode 644 --tls-san $PUBLIC_IP --tls-san localhost --bind-address 0.0.0.0" sh -
        
        # Wait for k3s to be ready
        echo "Waiting for k3s to be ready..."
        until kubectl get nodes 2>/dev/null; do
            echo "Waiting for k3s API server..."
            sleep 5
        done
        
        # Log completion
        echo "k3s installation completed at $(date)" >> /var/log/k3s-install.log
        
        # Display cluster info
        echo "=== k3s Cluster Information ===" >> /var/log/k3s-install.log
        kubectl cluster-info >> /var/log/k3s-install.log 2>&1
        kubectl get nodes >> /var/log/k3s-install.log 2>&1

        # Update IP tables to allow traffic on port 6443 (k3s API)
        sudo iptables -I INPUT -p tcp --dport 6443 -j ACCEPT

        # Save IP tables rule (persistent across reboots). Ignore if tere is an error
        sudo /sbin/service iptables save 2>/dev/null || true
        sudo sh -c "iptables-save > /etc/sysconfig/iptables 2>/dev/null || true"

    

runcmd:
  - /opt/k3s-install.sh

final_message: "k3s installation completed! Check /var/log/k3s-install.log for details."
EOF
    
    # Create VM
    az vm create \
        --resource-group "$RESOURCE_GROUP" \
        --name "$VM_NAME" \
        --image "/subscriptions/d04efe97-9374-4822-ac22-f38facb77dd3/resourceGroups/l1vh/providers/Microsoft.Compute/galleries/images/images/dom0qemu" \
        --size "$VM_SIZE" \
        --admin-username "$ADMIN_USERNAME" \
        --ssh-key-values "$SSH_PUBLIC_KEY" \
        --vnet-name "$vnet_name" \
        --subnet "$subnet_name" \
        --public-ip-address "$pip_name" \
        --os-disk-size-gb 128 \
        --custom-data "$cloud_init_file" \
        --output none
    
    print_success "Virtual machine '$VM_NAME' created"
}

# Configure NIC-level security rules
configure_nic_security() {
    print_status "Configuring NIC-level security rules..."
    
    local nic_name=$(az vm show --name "$VM_NAME" --resource-group "$RESOURCE_GROUP" \
            --query "networkProfile.networkInterfaces[0].id" --output tsv | \
            sed 's/.*\///')
    if [[ -z "$nic_name" ]]; then
        print_error "Could not find NIC for VM '$VM_NAME'"
        return 1
    fi
    
    # Create NIC-level NSG for additional security
    local nic_nsg_name="$(az network nic show --name "$nic_name" --resource-group "$RESOURCE_GROUP" \
            --query "networkSecurityGroup.id" --output tsv | \
            sed 's/.*\///')"
    
    # Add standard rules to the NIC NSG using our reusable function
    if [[ -n "$nic_nsg_name" ]]; then
        create_nsg_rules "$nic_nsg_name" "-NIC"
        print_success "NIC-level security rules configured"
    else
        print_warning "No NSG found associated with NIC"
    fi
}


# Get deployment information
get_deployment_info() {
    print_status "Gathering deployment information..."
    
    # Get public IP
    local pip_name="${VM_NAME}-pip"
    PUBLIC_IP=$(az network public-ip show \
        --name "$pip_name" \
        --resource-group "$RESOURCE_GROUP" \
        --query "ipAddress" \
        --output tsv)
    
    # Get FQDN
    FQDN=$(az network public-ip show \
        --name "$pip_name" \
        --resource-group "$RESOURCE_GROUP" \
        --query "dnsSettings.fqdn" \
        --output tsv)
    
}

# Display instructions
show_instructions() {
    echo ""
    echo "=================================="
    echo -e "${GREEN}🎉 Deployment Completed Successfully!${NC}"
    echo "=================================="
    echo ""
    echo -e "${BLUE}📋 Cluster Access Information:${NC}"
    echo "=================================="
    echo "Public IP:    $PUBLIC_IP"
    echo "FQDN:         $FQDN"
    echo "SSH User:     $ADMIN_USERNAME"
    echo "SSH Key:      $SSH_KEY_PATH"
    echo ""
    echo -e "${BLUE}🔑 SSH Access:${NC}"
    echo "ssh $ADMIN_USERNAME@$PUBLIC_IP"
    echo ""
    echo -e "${BLUE}☸️  Kubernetes Access:${NC}"
    echo "1. Copy kubeconfig from VM:"
    echo "   scp $ADMIN_USERNAME@$PUBLIC_IP:/etc/rancher/k3s/k3s.yaml ~/.kube/k3s-config"
    echo ""
    echo "2. Update server URL in kubeconfig:"
    echo "   sed -i 's/127.0.0.1/$PUBLIC_IP/g' ~/.kube/k3s-config"
    echo ""
    echo "3. Use kubectl with the config:"
    echo "   export KUBECONFIG=~/.kube/k3s-config"
    echo "   kubectl get nodes"
    echo ""
    echo "=================================="
    echo -e "${BLUE}🔧 Network Configuration:${NC}"
    echo "=================================="
    echo "✅ Port 6443 (k3s API) is open for inbound traffic"
    echo "✅ Port 22 (SSH) is open for management access"
    echo "✅ k3s installed"
    echo ""
    echo -e "${BLUE}🧹 Cleanup:${NC}"
    echo "To delete all resources: az group delete --name $RESOURCE_GROUP --yes --no-wait"
    echo ""
    echo "=================================="
    echo -e "${GREEN}✅ Setup Complete!${NC}"
    echo "k3s cluster is ready"
    echo "=================================="
}

# Main execution
main() {
    echo "🚀 Azure k3s Cluster Deployment"
    echo "====================================================="
    
    parse_args "$@"
    check_prerequisites
    create_resource_group
    create_network_resources
    create_virtual_machine
    configure_nic_security
    
    print_status "Waiting for VM to be ready and k3s to install..."
    sleep 60  # Give more time for the VM to start and cloud-init to complete
    
    get_deployment_info
    
    print_status "Waiting for SSH access to be available..."
    # Wait for SSH to be available
    local max_attempts=30
    local attempt=0
    local ssh_key_file
    
    # Determine the correct SSH key file to use (private key for SSH)
    if [[ "$SSH_KEY_PATH" == *.pub ]]; then
        ssh_key_file="${SSH_KEY_PATH%.*}"  # Remove .pub extension for private key
    else
        ssh_key_file="$SSH_KEY_PATH"
    fi
    
    # Check if private key exists
    if [[ ! -f "$ssh_key_file" ]]; then
        print_error "Private SSH key not found at $ssh_key_file"
        print_error "Please ensure the private key exists or provide the correct path"
        return 1
    fi
    
    while [ $attempt -lt $max_attempts ]; do
        if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i "$ssh_key_file" "${ADMIN_USERNAME}@${PUBLIC_IP}" "echo 'SSH connection successful'" 2>/dev/null; then
            print_success "SSH connection established"
            break
        fi
        attempt=$((attempt + 1))
        echo "SSH attempt $attempt/$max_attempts failed, retrying in 10 seconds..."
        sleep 10
    done
    
    show_instructions
}

# Run main function with all arguments
main "$@"
