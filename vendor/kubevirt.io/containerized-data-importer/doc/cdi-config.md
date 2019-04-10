# CDI User Configuration

CDI Config is a custom resource defining user configuration for CDI.
The configuration is created when CDI is deployed.

Currently it is used only for holding Upload Proxy URL details.

## Configuration Options

| Name                    | Default value         |                                                     |
|-------------------------|-----------------------|-----------------------------------------------------|
| uploadProxyURLOverride  | nil                   | A user defined URL for Upload Proxy service.        |
| scratchSpaceStorageClass| nil                   | The storage class used to create scratch space      |

## Configuration Status Fields

| Name                    | Default value         |                                                     |
|-------------------------|-----------------------|-----------------------------------------------------|
| uploadProxyURL          | nil                   | updated when a new Ingress or Route (Openshift) is created. If `uploadProxyURLOverride` is set, Ingress/Route URL will be ignored and `uploadProxyURL` will be updated with the user defined URL. |

