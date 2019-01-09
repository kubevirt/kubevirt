## Manual Design

The below diagram illustrates the short term goal of this project.  For our current
work we will not be focused on automation, implying that executing each step of
the import process will be done manually. User Flow (Current) provides explanation
of each step show in the diagram.

### Manual User Flow
Steps are identified by role according the colored shape. Each step must be performed in
the order they are number unless otherwise specified.

0. An admin stores the data in a network accessible location.
1. The admin must create the Golden PVC API Object in the kubernetes cluster. This step
kicks of the automated provisioning of a Persistent Volume.
2. The Dynamic Provisioner create the Persistent Volume API Object.
3. (In parallel to 2) The Dynamic Provisioner provisions the backing storage.
4. The admin creates the Endpoint Secret API Object.
5. The admin then creates the Data Import Pod.  (Prereqs: 1 & 4)
6. On startup, the Data Import Pod mounts the Secret and the PVC.  It then begins
streaming data from object store to the Golden Image location via the mounted PVC.

On completion of step 6, the pod will exit and the import is ended.

![Topology](../diagrams/cdi-manual-deploy.png)

### Components:
**Object Store:** Arbitrary url-based storage location.  Currently we support
http and S3 protocols.

**Images Namespace:** Restricted/private Namespace for Golden Persistent
Volume Claims. Data Importer Pod and it’s Endpoint Secret reside in this namespace.

**Dynamic Provisioner:** Existing storage provisoner(s) which create
the Golden Persistent Volume that reference an empty cluster storage volume.
Creation begins automatically when the Golden PVC is created by an admin.

**Golden PV:** Long-lived Persistent Volume created by the Dynamic Provisioner and
written to by the Data Import Pod.  References the Golden Image volume in storage.

**Golden PVC:** Long-lived claim manually created by an admin in the Images namespace.
Linked to the Dynamic Provisioner via a reference to the storage class and automatically
bound to a dynamically created Golden PV.  The "default" provisioner and storage class
is used in the example.  However, the importer pod should support any dynamic provisioner
that provides mountable volumes.

**Storage Class:** Long-lived, default Storage Class which links Persistent
Volume Claims to the default Dynamic Provisioner(s). Referenced by the golden PVC.
The example makes use of the "default" provisioner. However, any provisioner that
manages mountable volumes should be compatible.

**Endpoint Secret:** Short-lived secret in Images Namespace that must be defined
and created by an admin.  The Secret must contain the url, object path (bucket/object),
access key id and secret key required to make requests from the store.  The Secret
is mounted by the Data Import Pod

**Data Import Pod:** Short-lived Pod in Images Namespace.  The Pod Spec must be defined
by an admin to reference to Endpoint Secret and the Golden PVC.  On start, the Pod will
mount both and run the data import binary that is baked into the container.  The copy process
will consume values stored in the secret as environmental variables and stream data from
the url endpoint to the Golden PV. On completions (whether success or failure) the pod will exit.
