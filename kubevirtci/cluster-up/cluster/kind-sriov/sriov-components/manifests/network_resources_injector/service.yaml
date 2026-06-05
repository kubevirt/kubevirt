apiVersion: v1
kind: Service
metadata:
  name: network-resources-injector-service
  namespace: kube-system
spec:
  ports:
  - port: 443
    targetPort: 8443
  selector:
    app: network-resources-injector
