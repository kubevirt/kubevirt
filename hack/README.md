## Operator Test Script
**WARNING** The operator test script is not a permanent solution to deploying
KubeVirt component operators.  It's a stop-gap solution to provide automation
for launching component operators.

The hyperconverged-cluster-operator (HCO) will slowly replace the contents of
this script.

**Configure**
The ```config``` file contains configurable options.

**Launch**
The script requires `python-jinja2`. ```yum install -y python-jinja2```

```bash
./hack/operator-test.sh
```
