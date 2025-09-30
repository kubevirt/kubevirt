#!/bin/bash

# Copyright 2025 Cursor AI, Inc.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Create local directory for copying content
LOCAL_DIR="./debug-vol-content"
mkdir -p "$LOCAL_DIR"

echo "Getting PVCs with debug-vol prefix..."

# Get PVCs with debug-vol prefix, sort by AGE (newest first), take first 2
# Use --sort-by=.metadata.creationTimestamp to sort by actual creation time
LATEST_PVCS=$(kubectl get pvc --sort-by=.metadata.creationTimestamp | grep "debug-vol" | tail -2 | awk '{print $1}')

if [ -z "$LATEST_PVCS" ]; then
    echo "No PVCs with debug-vol prefix found"
    exit 1
fi

echo "Found latest PVCs:"
echo "$LATEST_PVCS"

# Process each PVC
for pvc in $LATEST_PVCS; do
    echo "Processing PVC: $pvc"
    
    # Create a temporary pod to mount the PVC
    POD_NAME="debug-copy-$pvc"
    
    # Create pod manifest
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: $POD_NAME
spec:
  containers:
  - name: debug-copy
    image: busybox
    command: ["sleep", "3600"]
    volumeMounts:
    - name: debug-vol
      mountPath: /debug
  volumes:
  - name: debug-vol
    persistentVolumeClaim:
      claimName: $pvc
  restartPolicy: Never
EOF

    # Wait for pod to be ready
    echo "Waiting for pod $POD_NAME to be ready..."
    kubectl wait --for=condition=Ready pod/$POD_NAME --timeout=60s
    
    if [ $? -eq 0 ]; then
        # Create local directory for this PVC
        PVC_DIR="$LOCAL_DIR/$pvc"
        mkdir -p "$PVC_DIR"
        
        # Copy content from the pod
        echo "Copying content from $pvc to $PVC_DIR..."
        kubectl cp $POD_NAME:/debug/ "$PVC_DIR/"
        
        echo "Content copied from $pvc to $PVC_DIR"
    else
        echo "Failed to start pod for $pvc"
    fi
    
    # Clean up the temporary pod
    echo "Cleaning up pod $POD_NAME..."
    kubectl delete pod $POD_NAME --ignore-not-found=true
done

echo "Done! Content copied to $LOCAL_DIR"
echo "Directory structure:"
ls -la "$LOCAL_DIR"
