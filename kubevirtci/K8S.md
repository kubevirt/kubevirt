# [kubevirtci](README.md): Getting Started with a multi-node Kubernetes Provider                              
                                                                                      
Download this repo                                                                    
```                                                                                   
git clone https://github.com/kubevirt/kubevirtci.git                                  
cd kubevirtci                                                                         
```                                                                                   
                                                                                      
Start multi node k8s cluster with 2 nics                                              
```                                                                                   
export KUBEVIRT_PROVIDER=k8s-1.33 KUBEVIRT_NUM_NODES=2 KUBEVIRT_NUM_SECONDARY_NICS=1
make cluster-up                                                                       
```                                                                                   
                                                                                      
Stop k8s cluster                                                                      
```
make cluster-down                                                                     
```

Use provider's kubectl client with kubectl.sh wrapper script               
```
export KUBEVIRTCI_TAG=`curl -L -Ss https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirtci/latest`
cluster-up/kubectl.sh get nodes                                            
cluster-up/kubectl.sh get pods --all-namespaces                            
```                                                                        
                                                                           
Use your own kubectl client by defining the KUBECONFIG environment variable
```
export KUBECONFIG=$(cluster-up/kubeconfig.sh)

kubectl get nodes
kubectl apply -f <some file>
```

## Global kubeconfig location

If you want the kubeconfig to be automatically copied to a specific location after cluster startup,
you can set the `GLOBAL_KUBECONFIG` environment variable:

```bash
export GLOBAL_KUBECONFIG=/path/to/your/kubeconfig
make cluster-up
```

SSH into a node                                                            
```                                                                        
cluster-up/ssh.sh node01                                                   
```                                                                        

Start single stack IPv6 cluster
```
export KUBEVIRT_SINGLE_STACK=true KUBEVIRT_PROVIDER=k8s-1.33
make cluster-up
```

## Attach to node console with screen and pty
```                                                  
# Attach to node01 console                           
docker exec -it ${KUBEVIRT_PROVIDER}-node01 screen /dev/pts/0
```                                                 
Use `vagrant:vagrant` for x86 and root:root for s390x to login.
Note: it is sometimes `/dev/pts/1` or `/dev/pts/2`, try them in case you don't get a prompt.

Make sure you don't leave open screens, else the next screen will be messed up.  
`screen -ls` shows the open screens.  
`screen -XS <session-id> quit` closes an open session.
Close all zombies and shutdown screen gracefully if you plan to open a new one instead.
Ctrl+A and Ctrl+D will detach your screen session and `screen -r <session-id>` reattach to a detached screen session.

## Container image cache
In order to have a local cache of container images:
1. Run your proxy (see for example https://github.com/rpardini/docker-registry-proxy)
2. Get the IP:PORT of the proxy and run `export KUBEVIRTCI_PROXY=http://<IP>:<PORT>`
3. Run `cluster-up` flow as usual
