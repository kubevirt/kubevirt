# How To Authenticate Private App-Registry Repositories

If you have an app-registry repository that is backed by authentication, you can specify an authentication token in a Secret. To do this, create a Secret in the same namespace as your Operator Source:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: marketplacesecret
  namespace: openshift-marketplace
type: Opaque
stringData:
    token: "basic yourtokenhere=="
```

Then, to associate that secret with a registry, simply add a reference to the secret in the OperatorSource spec:

```yaml
apiVersion: "operators.coreos.com/v1"
kind: "OperatorSource"
metadata:
  name: "certified-operators"
  namespace: "openshift-marketplace"
  labels:
    opsrc-provider: certified
spec:
  type: appregistry
  endpoint: "https://quay.io/cnr"
  registryNamespace: "certified-operators"
  displayName: "Certified Operators"
  publisher: "Red Hat"
  authorizationToken:
    secretName: marketplacesecret
```

That's it! When downloading the repository from the app-registry, Marketplace will pass along the authentication token.
