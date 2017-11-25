
### KubeVirt deployment with Ansible

#### Inventory

The inventory is fairly simple. First make a copy of the example

```
cp inventory/hosts.ini{.example,}
```

Then just replace the kube-master1 with the FQDN or IP address of your kubernetes
master

#### Run ansible
```
ansible-playbook -i inventory/hosts.ini playbooks/main.yml
```
