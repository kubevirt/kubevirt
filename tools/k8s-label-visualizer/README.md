# Tools for k8s labels

The `main.py` script
- draws relationship graph between k8s objects by using "app.kubernetes.io/managed-by" label
- draws component graph between k8s objects by using "app.kubernetes.io/component" label
- prints cpu and memory consumption per component by sending queries to prometheus and using "app.kubernetes.io/component" label 

***Requirements***
```
dnf install graphviz
```

***Installation***
```
python3 -m venv ./venv
source ./venv/bin/activate
pip3 install -r requirements.txt
```

 
***Usage***
```
python3 main.py  --namespace kubevirt-hyperconverged --conf conf.json --output out
```
