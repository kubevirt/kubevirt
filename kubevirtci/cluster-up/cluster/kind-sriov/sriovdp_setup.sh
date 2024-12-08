#!/bin/bash

get_sriov_pci_root_addresses() {
  for dir in $(find /sys/devices/ -name sriov_totalvfs -exec dirname {} \;); do
    if [ $(cat $dir/sriov_numvfs) -gt 0 ]; then
      # use perl because sed doesn't support non-greedy matching
      basename $dir | perl -pe 's|(.*?:)(.*)|\2|'
    fi
  done
}

create_pci_string() {
  local quoted_values=($(echo "${pci_addresses[@]}" | xargs printf "\"%s\" "  ))
  local quoted_as_string=${quoted_values[@]}
  if [ "$quoted_as_string" = "\"\"" ]; then
    pci_string=""
  else
    pci_string=${quoted_as_string// /, }
  fi
}

sriov_device_plugin() {
  pci_addresses=$(get_sriov_pci_root_addresses)
  create_pci_string

  cat <<EOF > /etc/pcidp/config.json
{
    "resourceList":
    [
        {
            "resourceName": "sriov",
            "rootDevices": [$pci_string],
            "sriovMode": true,
            "deviceType": "vfio"
        }
    ]
}
EOF
}

mkdir -p /etc/pcidp
sriov_device_plugin
