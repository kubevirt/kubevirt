package components

var CRDsValidation map[string]string = map[string]string{
	"datavolumetemplatespec": `openAPIV3Schema:
  nullable: true
  properties:
    apiVersion:
      description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
      type: string
    kind:
      description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
      type: string
    metadata:
      nullable: true
      type: object
      x-kubernetes-preserve-unknown-fields: true
    spec:
      description: DataVolumeSpec contains the DataVolume specification.
      properties:
        contentType:
          description: 'DataVolumeContentType options: "kubevirt", "archive"'
          enum:
          - kubevirt
          - archive
          type: string
        pvc:
          description: PVC is the PVC specification
          properties:
            accessModes:
              description: 'AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1'
              items:
                type: string
              type: array
            dataSource:
              description: This field requires the VolumeSnapshotDataSource alpha feature gate to be enabled and currently VolumeSnapshot is the only supported data source. If the provisioner can support VolumeSnapshot data source, it will create a new volume and data will be restored to the volume at the same time. If the provisioner does not support VolumeSnapshot data source, volume will not be created and the failure will be reported as an event. In the future, we plan to support more data source types and the behavior of the provisioner may change.
              properties:
                apiGroup:
                  description: APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.
                  type: string
                kind:
                  description: Kind is the type of resource being referenced
                  type: string
                name:
                  description: Name is the name of resource being referenced
                  type: string
              required:
              - kind
              - name
              type: object
            resources:
              description: 'Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources'
              properties:
                limits:
                  additionalProperties:
                    anyOf:
                    - type: integer
                    - type: string
                    pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                    x-kubernetes-int-or-string: true
                  description: 'Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/'
                  type: object
                requests:
                  additionalProperties:
                    anyOf:
                    - type: integer
                    - type: string
                    pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                    x-kubernetes-int-or-string: true
                  description: 'Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/'
                  type: object
              type: object
            selector:
              description: A label query over volumes to consider for binding.
              properties:
                matchExpressions:
                  description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                  items:
                    description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                    properties:
                      key:
                        description: key is the label key that the selector applies to.
                        type: string
                      operator:
                        description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                        type: string
                      values:
                        description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                        items:
                          type: string
                        type: array
                    required:
                    - key
                    - operator
                    type: object
                  type: array
                matchLabels:
                  additionalProperties:
                    type: string
                  description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                  type: object
              type: object
            storageClassName:
              description: 'Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1'
              type: string
            volumeMode:
              description: volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec. This is a beta feature.
              type: string
            volumeName:
              description: VolumeName is the binding reference to the PersistentVolume backing this claim.
              type: string
          type: object
        source:
          description: Source is the src of the data for the requested DataVolume
          properties:
            blank:
              description: DataVolumeBlankImage provides the parameters to create a new raw blank image for the PVC
              type: object
            http:
              description: DataVolumeSourceHTTP can be either an http or https endpoint, with an optional basic auth user name and password, and an optional configmap containing additional CAs
              properties:
                certConfigMap:
                  description: CertConfigMap is a configmap reference, containing a Certificate Authority(CA) public key, and a base64 encoded pem certificate
                  type: string
                secretRef:
                  description: SecretRef A Secret reference, the secret should contain accessKeyId (user name) base64 encoded, and secretKey (password) also base64 encoded
                  type: string
                url:
                  description: URL is the URL of the http(s) endpoint
                  type: string
              required:
              - url
              type: object
            imageio:
              description: DataVolumeSourceImageIO provides the parameters to create a Data Volume from an imageio source
              properties:
                certConfigMap:
                  description: CertConfigMap provides a reference to the CA cert
                  type: string
                diskId:
                  description: DiskID provides id of a disk to be imported
                  type: string
                secretRef:
                  description: SecretRef provides the secret reference needed to access the ovirt-engine
                  type: string
                url:
                  description: URL is the URL of the ovirt-engine
                  type: string
              required:
              - diskId
              - url
              type: object
            pvc:
              description: DataVolumeSourcePVC provides the parameters to create a Data Volume from an existing PVC
              properties:
                name:
                  description: The name of the source PVC
                  type: string
                namespace:
                  description: The namespace of the source PVC
                  type: string
              required:
              - name
              - namespace
              type: object
            registry:
              description: DataVolumeSourceRegistry provides the parameters to create a Data Volume from an registry source
              properties:
                certConfigMap:
                  description: CertConfigMap provides a reference to the Registry certs
                  type: string
                secretRef:
                  description: SecretRef provides the secret reference needed to access the Registry source
                  type: string
                url:
                  description: URL is the url of the Docker registry source
                  type: string
              required:
              - url
              type: object
            s3:
              description: DataVolumeSourceS3 provides the parameters to create a Data Volume from an S3 source
              properties:
                secretRef:
                  description: SecretRef provides the secret reference needed to access the S3 source
                  type: string
                url:
                  description: URL is the url of the S3 source
                  type: string
              required:
              - url
              type: object
            upload:
              description: DataVolumeSourceUpload provides the parameters to create a Data Volume by uploading the source
              type: object
            vddk:
              description: DataVolumeSourceVDDK provides the parameters to create a Data Volume from a Vmware source
              properties:
                backingFile:
                  description: BackingFile is the path to the virtual hard disk to migrate from vCenter/ESXi
                  type: string
                secretRef:
                  description: SecretRef provides a reference to a secret containing the username and password needed to access the vCenter or ESXi host
                  type: string
                thumbprint:
                  description: Thumbprint is the certificate thumbprint of the vCenter or ESXi host
                  type: string
                url:
                  description: URL is the URL of the vCenter or ESXi host with the VM to migrate
                  type: string
                uuid:
                  description: UUID is the UUID of the virtual machine that the backing file is attached to in vCenter/ESXi
                  type: string
              type: object
          type: object
      required:
      - pvc
      - source
      type: object
    status:
      description: DataVolumeTemplateDummyStatus is here simply for backwards compatibility with a previous API.
      nullable: true
      type: object
  required:
  - spec
  type: object
`,
	"kubevirt": `openAPIV3Schema:
  description: KubeVirt represents the object deploying all KubeVirt resources
  properties:
    apiVersion:
      description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
      type: string
    kind:
      description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
      type: string
    metadata:
      type: object
    spec:
      description: '---'
      properties:
        certificateRotateStrategy:
          description: '---'
          properties:
            selfSigned:
              description: '---'
              properties:
                caOverlapInterval:
                  type: string
                caRotateInterval:
                  type: string
                certRotateInterval:
                  type: string
              type: object
          type: object
        configuration:
          description: holds kubevirt configurations. same as the virt-configMap
          properties:
            cpuModel:
              type: string
            cpuRequest:
              anyOf:
              - type: integer
              - type: string
              pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
              x-kubernetes-int-or-string: true
            developerConfiguration:
              description: DeveloperConfiguration holds developer options
              properties:
                cpuAllocationRatio:
                  type: number
                featureGates:
                  items:
                    type: string
                  type: array
                memoryOvercommit:
                  type: integer
                nodeSelectors:
                  additionalProperties:
                    type: string
                  type: object
                pvcTolerateLessSpaceUpToPercent:
                  type: integer
                useEmulation:
                  type: boolean
              type: object
            emulatedMachines:
              items:
                type: string
              type: array
            imagePullPolicy:
              description: PullPolicy describes a policy for if/when to pull a container image
              type: string
            machineType:
              type: string
            memBalloonStatsPeriod:
              format: int32
              type: integer
            migrations:
              description: MigrationConfiguration holds migration options
              properties:
                allowAutoConverge:
                  type: boolean
                allowPostCopy:
                  type: boolean
                bandwidthPerMigration:
                  anyOf:
                  - type: integer
                  - type: string
                  pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                  x-kubernetes-int-or-string: true
                completionTimeoutPerGiB:
                  format: int64
                  type: integer
                nodeDrainTaintKey:
                  type: string
                parallelMigrationsPerCluster:
                  format: int32
                  type: integer
                parallelOutboundMigrationsPerNode:
                  format: int32
                  type: integer
                progressTimeout:
                  format: int64
                  type: integer
                unsafeMigrationOverride:
                  type: boolean
              type: object
            network:
              description: NetworkConfiguration holds network options
              properties:
                defaultNetworkInterface:
                  type: string
                permitBridgeInterfaceOnPodNetwork:
                  type: boolean
                permitSlirpInterface:
                  type: boolean
              type: object
            ovmfPath:
              type: string
            selinuxLauncherType:
              type: string
            smbios:
              description: '---'
              properties:
                family:
                  type: string
                manufacturer:
                  type: string
                product:
                  type: string
                sku:
                  type: string
                version:
                  type: string
              type: object
            supportedGuestAgentVersions:
              items:
                type: string
              type: array
          type: object
        customizeComponents:
          properties:
            patches:
              items:
                properties:
                  patch:
                    type: string
                  resourceName:
                    type: string
                  resourceType:
                    type: string
                  type:
                    type: string
                type: object
              type: array
              x-kubernetes-list-type: atomic
          type: object
        imagePullPolicy:
          description: The ImagePullPolicy to use.
          type: string
        imageRegistry:
          description: The image registry to pull the container images from Defaults to the same registry the operator's container image is pulled from.
          type: string
        imageTag:
          description: The image tag to use for the continer images installed. Defaults to the same tag as the operator's container image.
          type: string
        infra:
          description: selectors and tolerations that should apply to KubeVirt infrastructure components
          properties:
            nodePlacement:
              description: nodePlacement decsribes scheduling confiuguration for specific KubeVirt components
              properties:
                affinity:
                  description: affinity enables pod affinity/anti-affinity placement expanding the types of constraints that can be expressed with nodeSelector. affinity is going to be applied to the relevant kind of pods in parallel with nodeSelector See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity
                  properties:
                    nodeAffinity:
                      description: Describes node affinity scheduling rules for the pod.
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.
                          items:
                            description: An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).
                            properties:
                              preference:
                                description: A node selector term, associated with the corresponding weight.
                                properties:
                                  matchExpressions:
                                    description: A list of node selector requirements by node's labels.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchFields:
                                    description: A list of node selector requirements by node's fields.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                type: object
                              weight:
                                description: Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - preference
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.
                          properties:
                            nodeSelectorTerms:
                              description: Required. A list of node selector terms. The terms are ORed.
                              items:
                                description: A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.
                                properties:
                                  matchExpressions:
                                    description: A list of node selector requirements by node's labels.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchFields:
                                    description: A list of node selector requirements by node's fields.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                type: object
                              type: array
                          required:
                          - nodeSelectorTerms
                          type: object
                      type: object
                    podAffinity:
                      description: Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                          items:
                            description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                            properties:
                              podAffinityTerm:
                                description: Required. A pod affinity term, associated with the corresponding weight.
                                properties:
                                  labelSelector:
                                    description: A label query over a set of resources, in this case pods.
                                    properties:
                                      matchExpressions:
                                        description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                        items:
                                          description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                          properties:
                                            key:
                                              description: key is the label key that the selector applies to.
                                              type: string
                                            operator:
                                              description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                              type: string
                                            values:
                                              description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                              items:
                                                type: string
                                              type: array
                                          required:
                                          - key
                                          - operator
                                          type: object
                                        type: array
                                      matchLabels:
                                        additionalProperties:
                                          type: string
                                        description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                        type: object
                                    type: object
                                  namespaces:
                                    description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                    items:
                                      type: string
                                    type: array
                                  topologyKey:
                                    description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                    type: string
                                required:
                                - topologyKey
                                type: object
                              weight:
                                description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - podAffinityTerm
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                          items:
                            description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                            properties:
                              labelSelector:
                                description: A label query over a set of resources, in this case pods.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                    items:
                                      description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: key is the label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                    type: object
                                type: object
                              namespaces:
                                description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                items:
                                  type: string
                                type: array
                              topologyKey:
                                description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                type: string
                            required:
                            - topologyKey
                            type: object
                          type: array
                      type: object
                    podAntiAffinity:
                      description: Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                          items:
                            description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                            properties:
                              podAffinityTerm:
                                description: Required. A pod affinity term, associated with the corresponding weight.
                                properties:
                                  labelSelector:
                                    description: A label query over a set of resources, in this case pods.
                                    properties:
                                      matchExpressions:
                                        description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                        items:
                                          description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                          properties:
                                            key:
                                              description: key is the label key that the selector applies to.
                                              type: string
                                            operator:
                                              description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                              type: string
                                            values:
                                              description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                              items:
                                                type: string
                                              type: array
                                          required:
                                          - key
                                          - operator
                                          type: object
                                        type: array
                                      matchLabels:
                                        additionalProperties:
                                          type: string
                                        description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                        type: object
                                    type: object
                                  namespaces:
                                    description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                    items:
                                      type: string
                                    type: array
                                  topologyKey:
                                    description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                    type: string
                                required:
                                - topologyKey
                                type: object
                              weight:
                                description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - podAffinityTerm
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                          items:
                            description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                            properties:
                              labelSelector:
                                description: A label query over a set of resources, in this case pods.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                    items:
                                      description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: key is the label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                    type: object
                                type: object
                              namespaces:
                                description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                items:
                                  type: string
                                type: array
                              topologyKey:
                                description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                type: string
                            required:
                            - topologyKey
                            type: object
                          type: array
                      type: object
                  type: object
                nodeSelector:
                  additionalProperties:
                    type: string
                  description: 'nodeSelector is the node selector applied to the relevant kind of pods It specifies a map of key-value pairs: for the pod to be eligible to run on a node, the node must have each of the indicated key-value pairs as labels (it can have additional labels as well). See https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector'
                  type: object
                tolerations:
                  description: tolerations is a list of tolerations applied to the relevant kind of pods See https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ for more info. These are additional tolerations other than default ones.
                  items:
                    description: The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.
                    properties:
                      effect:
                        description: Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.
                        type: string
                      key:
                        description: Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.
                        type: string
                      operator:
                        description: Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.
                        type: string
                      tolerationSeconds:
                        description: TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.
                        format: int64
                        type: integer
                      value:
                        description: Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.
                        type: string
                    type: object
                  type: array
              type: object
          type: object
        monitorAccount:
          description: The name of the Prometheus service account that needs read-access to KubeVirt endpoints Defaults to prometheus-k8s
          type: string
        monitorNamespace:
          description: The namespace Prometheus is deployed in Defaults to openshift-monitor
          type: string
        productName:
          description: Designate the apps.kubevirt.io/part-of label for KubeVirt components. Useful if KubeVirt is included as part of a product. If ProductName is not specified, the part-of label will be omitted.
          type: string
        productVersion:
          description: Designate the apps.kubevirt.io/version label for KubeVirt components. Useful if KubeVirt is included as part of a product. If ProductVersion is not specified, KubeVirt's version will be used.
          type: string
        uninstallStrategy:
          description: Specifies if kubevirt can be deleted if workloads are still present. This is mainly a precaution to avoid accidental data loss
          type: string
        workloads:
          description: selectors and tolerations that should apply to KubeVirt workloads
          properties:
            nodePlacement:
              description: nodePlacement decsribes scheduling confiuguration for specific KubeVirt components
              properties:
                affinity:
                  description: affinity enables pod affinity/anti-affinity placement expanding the types of constraints that can be expressed with nodeSelector. affinity is going to be applied to the relevant kind of pods in parallel with nodeSelector See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity
                  properties:
                    nodeAffinity:
                      description: Describes node affinity scheduling rules for the pod.
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.
                          items:
                            description: An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).
                            properties:
                              preference:
                                description: A node selector term, associated with the corresponding weight.
                                properties:
                                  matchExpressions:
                                    description: A list of node selector requirements by node's labels.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchFields:
                                    description: A list of node selector requirements by node's fields.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                type: object
                              weight:
                                description: Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - preference
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.
                          properties:
                            nodeSelectorTerms:
                              description: Required. A list of node selector terms. The terms are ORed.
                              items:
                                description: A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.
                                properties:
                                  matchExpressions:
                                    description: A list of node selector requirements by node's labels.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchFields:
                                    description: A list of node selector requirements by node's fields.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                type: object
                              type: array
                          required:
                          - nodeSelectorTerms
                          type: object
                      type: object
                    podAffinity:
                      description: Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                          items:
                            description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                            properties:
                              podAffinityTerm:
                                description: Required. A pod affinity term, associated with the corresponding weight.
                                properties:
                                  labelSelector:
                                    description: A label query over a set of resources, in this case pods.
                                    properties:
                                      matchExpressions:
                                        description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                        items:
                                          description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                          properties:
                                            key:
                                              description: key is the label key that the selector applies to.
                                              type: string
                                            operator:
                                              description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                              type: string
                                            values:
                                              description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                              items:
                                                type: string
                                              type: array
                                          required:
                                          - key
                                          - operator
                                          type: object
                                        type: array
                                      matchLabels:
                                        additionalProperties:
                                          type: string
                                        description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                        type: object
                                    type: object
                                  namespaces:
                                    description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                    items:
                                      type: string
                                    type: array
                                  topologyKey:
                                    description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                    type: string
                                required:
                                - topologyKey
                                type: object
                              weight:
                                description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - podAffinityTerm
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                          items:
                            description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                            properties:
                              labelSelector:
                                description: A label query over a set of resources, in this case pods.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                    items:
                                      description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: key is the label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                    type: object
                                type: object
                              namespaces:
                                description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                items:
                                  type: string
                                type: array
                              topologyKey:
                                description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                type: string
                            required:
                            - topologyKey
                            type: object
                          type: array
                      type: object
                    podAntiAffinity:
                      description: Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                          items:
                            description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                            properties:
                              podAffinityTerm:
                                description: Required. A pod affinity term, associated with the corresponding weight.
                                properties:
                                  labelSelector:
                                    description: A label query over a set of resources, in this case pods.
                                    properties:
                                      matchExpressions:
                                        description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                        items:
                                          description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                          properties:
                                            key:
                                              description: key is the label key that the selector applies to.
                                              type: string
                                            operator:
                                              description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                              type: string
                                            values:
                                              description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                              items:
                                                type: string
                                              type: array
                                          required:
                                          - key
                                          - operator
                                          type: object
                                        type: array
                                      matchLabels:
                                        additionalProperties:
                                          type: string
                                        description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                        type: object
                                    type: object
                                  namespaces:
                                    description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                    items:
                                      type: string
                                    type: array
                                  topologyKey:
                                    description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                    type: string
                                required:
                                - topologyKey
                                type: object
                              weight:
                                description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - podAffinityTerm
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                          items:
                            description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                            properties:
                              labelSelector:
                                description: A label query over a set of resources, in this case pods.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                    items:
                                      description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: key is the label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                    type: object
                                type: object
                              namespaces:
                                description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                items:
                                  type: string
                                type: array
                              topologyKey:
                                description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                type: string
                            required:
                            - topologyKey
                            type: object
                          type: array
                      type: object
                  type: object
                nodeSelector:
                  additionalProperties:
                    type: string
                  description: 'nodeSelector is the node selector applied to the relevant kind of pods It specifies a map of key-value pairs: for the pod to be eligible to run on a node, the node must have each of the indicated key-value pairs as labels (it can have additional labels as well). See https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector'
                  type: object
                tolerations:
                  description: tolerations is a list of tolerations applied to the relevant kind of pods See https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ for more info. These are additional tolerations other than default ones.
                  items:
                    description: The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.
                    properties:
                      effect:
                        description: Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.
                        type: string
                      key:
                        description: Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.
                        type: string
                      operator:
                        description: Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.
                        type: string
                      tolerationSeconds:
                        description: TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.
                        format: int64
                        type: integer
                      value:
                        description: Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.
                        type: string
                    type: object
                  type: array
              type: object
          type: object
      type: object
    status:
      description: KubeVirtStatus represents information pertaining to a KubeVirt deployment.
      properties:
        conditions:
          items:
            description: KubeVirtCondition represents a condition of a KubeVirt deployment
            properties:
              lastProbeTime:
                format: date-time
                nullable: true
                type: string
              lastTransitionTime:
                format: date-time
                nullable: true
                type: string
              message:
                type: string
              reason:
                type: string
              status:
                type: string
              type:
                type: string
            required:
            - status
            - type
            type: object
          type: array
        observedDeploymentConfig:
          type: string
        observedDeploymentID:
          type: string
        observedKubeVirtRegistry:
          type: string
        observedKubeVirtVersion:
          type: string
        operatorVersion:
          type: string
        phase:
          description: KubeVirtPhase is a label for the phase of a KubeVirt deployment at the current time.
          type: string
        targetDeploymentConfig:
          type: string
        targetDeploymentID:
          type: string
        targetKubeVirtRegistry:
          type: string
        targetKubeVirtVersion:
          type: string
      type: object
  required:
  - spec
  type: object
`,
	"virtualmachine": `openAPIV3Schema:
  description: VirtualMachine handles the VirtualMachines that are not running or are in a stopped state The VirtualMachine contains the template to create the VirtualMachineInstance. It also mirrors the running state of the created VirtualMachineInstance in its status.
  properties:
    apiVersion:
      description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
      type: string
    kind:
      description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
      type: string
    metadata:
      type: object
    spec:
      description: Spec contains the specification of VirtualMachineInstance created
      properties:
        dataVolumeTemplates:
          description: dataVolumeTemplates is a list of dataVolumes that the VirtualMachineInstance template can reference. DataVolumes in this list are dynamically created for the VirtualMachine and are tied to the VirtualMachine's life-cycle.
          items:
            nullable: true
            properties:
              apiVersion:
                description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
                type: string
              kind:
                description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
                type: string
              metadata:
                nullable: true
                type: object
                x-kubernetes-preserve-unknown-fields: true
              spec:
                description: DataVolumeSpec contains the DataVolume specification.
                properties:
                  contentType:
                    description: 'DataVolumeContentType options: "kubevirt", "archive"'
                    enum:
                    - kubevirt
                    - archive
                    type: string
                  pvc:
                    description: PVC is the PVC specification
                    properties:
                      accessModes:
                        description: 'AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1'
                        items:
                          type: string
                        type: array
                      dataSource:
                        description: This field requires the VolumeSnapshotDataSource alpha feature gate to be enabled and currently VolumeSnapshot is the only supported data source. If the provisioner can support VolumeSnapshot data source, it will create a new volume and data will be restored to the volume at the same time. If the provisioner does not support VolumeSnapshot data source, volume will not be created and the failure will be reported as an event. In the future, we plan to support more data source types and the behavior of the provisioner may change.
                        properties:
                          apiGroup:
                            description: APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.
                            type: string
                          kind:
                            description: Kind is the type of resource being referenced
                            type: string
                          name:
                            description: Name is the name of resource being referenced
                            type: string
                        required:
                        - kind
                        - name
                        type: object
                      resources:
                        description: 'Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources'
                        properties:
                          limits:
                            additionalProperties:
                              anyOf:
                              - type: integer
                              - type: string
                              pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                              x-kubernetes-int-or-string: true
                            description: 'Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/'
                            type: object
                          requests:
                            additionalProperties:
                              anyOf:
                              - type: integer
                              - type: string
                              pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                              x-kubernetes-int-or-string: true
                            description: 'Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/'
                            type: object
                        type: object
                      selector:
                        description: A label query over volumes to consider for binding.
                        properties:
                          matchExpressions:
                            description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                            items:
                              description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                              properties:
                                key:
                                  description: key is the label key that the selector applies to.
                                  type: string
                                operator:
                                  description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                  type: string
                                values:
                                  description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                  items:
                                    type: string
                                  type: array
                              required:
                              - key
                              - operator
                              type: object
                            type: array
                          matchLabels:
                            additionalProperties:
                              type: string
                            description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                            type: object
                        type: object
                      storageClassName:
                        description: 'Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1'
                        type: string
                      volumeMode:
                        description: volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec. This is a beta feature.
                        type: string
                      volumeName:
                        description: VolumeName is the binding reference to the PersistentVolume backing this claim.
                        type: string
                    type: object
                  source:
                    description: Source is the src of the data for the requested DataVolume
                    properties:
                      blank:
                        description: DataVolumeBlankImage provides the parameters to create a new raw blank image for the PVC
                        type: object
                      http:
                        description: DataVolumeSourceHTTP can be either an http or https endpoint, with an optional basic auth user name and password, and an optional configmap containing additional CAs
                        properties:
                          certConfigMap:
                            description: CertConfigMap is a configmap reference, containing a Certificate Authority(CA) public key, and a base64 encoded pem certificate
                            type: string
                          secretRef:
                            description: SecretRef A Secret reference, the secret should contain accessKeyId (user name) base64 encoded, and secretKey (password) also base64 encoded
                            type: string
                          url:
                            description: URL is the URL of the http(s) endpoint
                            type: string
                        required:
                        - url
                        type: object
                      imageio:
                        description: DataVolumeSourceImageIO provides the parameters to create a Data Volume from an imageio source
                        properties:
                          certConfigMap:
                            description: CertConfigMap provides a reference to the CA cert
                            type: string
                          diskId:
                            description: DiskID provides id of a disk to be imported
                            type: string
                          secretRef:
                            description: SecretRef provides the secret reference needed to access the ovirt-engine
                            type: string
                          url:
                            description: URL is the URL of the ovirt-engine
                            type: string
                        required:
                        - diskId
                        - url
                        type: object
                      pvc:
                        description: DataVolumeSourcePVC provides the parameters to create a Data Volume from an existing PVC
                        properties:
                          name:
                            description: The name of the source PVC
                            type: string
                          namespace:
                            description: The namespace of the source PVC
                            type: string
                        required:
                        - name
                        - namespace
                        type: object
                      registry:
                        description: DataVolumeSourceRegistry provides the parameters to create a Data Volume from an registry source
                        properties:
                          certConfigMap:
                            description: CertConfigMap provides a reference to the Registry certs
                            type: string
                          secretRef:
                            description: SecretRef provides the secret reference needed to access the Registry source
                            type: string
                          url:
                            description: URL is the url of the Docker registry source
                            type: string
                        required:
                        - url
                        type: object
                      s3:
                        description: DataVolumeSourceS3 provides the parameters to create a Data Volume from an S3 source
                        properties:
                          secretRef:
                            description: SecretRef provides the secret reference needed to access the S3 source
                            type: string
                          url:
                            description: URL is the url of the S3 source
                            type: string
                        required:
                        - url
                        type: object
                      upload:
                        description: DataVolumeSourceUpload provides the parameters to create a Data Volume by uploading the source
                        type: object
                      vddk:
                        description: DataVolumeSourceVDDK provides the parameters to create a Data Volume from a Vmware source
                        properties:
                          backingFile:
                            description: BackingFile is the path to the virtual hard disk to migrate from vCenter/ESXi
                            type: string
                          secretRef:
                            description: SecretRef provides a reference to a secret containing the username and password needed to access the vCenter or ESXi host
                            type: string
                          thumbprint:
                            description: Thumbprint is the certificate thumbprint of the vCenter or ESXi host
                            type: string
                          url:
                            description: URL is the URL of the vCenter or ESXi host with the VM to migrate
                            type: string
                          uuid:
                            description: UUID is the UUID of the virtual machine that the backing file is attached to in vCenter/ESXi
                            type: string
                        type: object
                    type: object
                required:
                - pvc
                - source
                type: object
              status:
                description: DataVolumeTemplateDummyStatus is here simply for backwards compatibility with a previous API.
                nullable: true
                type: object
            required:
            - spec
            type: object
          type: array
        runStrategy:
          description: Running state indicates the requested running state of the VirtualMachineInstance mutually exclusive with Running
          type: string
        running:
          description: Running controls whether the associatied VirtualMachineInstance is created or not Mutually exclusive with RunStrategy
          type: boolean
        template:
          description: Template is the direct specification of VirtualMachineInstance
          properties:
            metadata:
              nullable: true
              type: object
              x-kubernetes-preserve-unknown-fields: true
            spec:
              description: VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
              properties:
                affinity:
                  description: If affinity is specifies, obey all the affinity rules
                  properties:
                    nodeAffinity:
                      description: Describes node affinity scheduling rules for the pod.
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.
                          items:
                            description: An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).
                            properties:
                              preference:
                                description: A node selector term, associated with the corresponding weight.
                                properties:
                                  matchExpressions:
                                    description: A list of node selector requirements by node's labels.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchFields:
                                    description: A list of node selector requirements by node's fields.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                type: object
                              weight:
                                description: Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - preference
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.
                          properties:
                            nodeSelectorTerms:
                              description: Required. A list of node selector terms. The terms are ORed.
                              items:
                                description: A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.
                                properties:
                                  matchExpressions:
                                    description: A list of node selector requirements by node's labels.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchFields:
                                    description: A list of node selector requirements by node's fields.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                type: object
                              type: array
                          required:
                          - nodeSelectorTerms
                          type: object
                      type: object
                    podAffinity:
                      description: Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                          items:
                            description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                            properties:
                              podAffinityTerm:
                                description: Required. A pod affinity term, associated with the corresponding weight.
                                properties:
                                  labelSelector:
                                    description: A label query over a set of resources, in this case pods.
                                    properties:
                                      matchExpressions:
                                        description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                        items:
                                          description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                          properties:
                                            key:
                                              description: key is the label key that the selector applies to.
                                              type: string
                                            operator:
                                              description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                              type: string
                                            values:
                                              description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                              items:
                                                type: string
                                              type: array
                                          required:
                                          - key
                                          - operator
                                          type: object
                                        type: array
                                      matchLabels:
                                        additionalProperties:
                                          type: string
                                        description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                        type: object
                                    type: object
                                  namespaces:
                                    description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                    items:
                                      type: string
                                    type: array
                                  topologyKey:
                                    description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                    type: string
                                required:
                                - topologyKey
                                type: object
                              weight:
                                description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - podAffinityTerm
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                          items:
                            description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                            properties:
                              labelSelector:
                                description: A label query over a set of resources, in this case pods.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                    items:
                                      description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: key is the label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                    type: object
                                type: object
                              namespaces:
                                description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                items:
                                  type: string
                                type: array
                              topologyKey:
                                description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                type: string
                            required:
                            - topologyKey
                            type: object
                          type: array
                      type: object
                    podAntiAffinity:
                      description: Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                          items:
                            description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                            properties:
                              podAffinityTerm:
                                description: Required. A pod affinity term, associated with the corresponding weight.
                                properties:
                                  labelSelector:
                                    description: A label query over a set of resources, in this case pods.
                                    properties:
                                      matchExpressions:
                                        description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                        items:
                                          description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                          properties:
                                            key:
                                              description: key is the label key that the selector applies to.
                                              type: string
                                            operator:
                                              description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                              type: string
                                            values:
                                              description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                              items:
                                                type: string
                                              type: array
                                          required:
                                          - key
                                          - operator
                                          type: object
                                        type: array
                                      matchLabels:
                                        additionalProperties:
                                          type: string
                                        description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                        type: object
                                    type: object
                                  namespaces:
                                    description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                    items:
                                      type: string
                                    type: array
                                  topologyKey:
                                    description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                    type: string
                                required:
                                - topologyKey
                                type: object
                              weight:
                                description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - podAffinityTerm
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                          items:
                            description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                            properties:
                              labelSelector:
                                description: A label query over a set of resources, in this case pods.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                    items:
                                      description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: key is the label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                    type: object
                                type: object
                              namespaces:
                                description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                items:
                                  type: string
                                type: array
                              topologyKey:
                                description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                type: string
                            required:
                            - topologyKey
                            type: object
                          type: array
                      type: object
                  type: object
                dnsConfig:
                  description: Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS configuration based on DNSPolicy.
                  properties:
                    nameservers:
                      description: A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed.
                      items:
                        type: string
                      type: array
                    options:
                      description: A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy.
                      items:
                        description: PodDNSConfigOption defines DNS resolver options of a pod.
                        properties:
                          name:
                            description: Required.
                            type: string
                          value:
                            type: string
                        type: object
                      type: array
                    searches:
                      description: A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed.
                      items:
                        type: string
                      type: array
                  type: object
                dnsPolicy:
                  description: Set DNS policy for the pod. Defaults to "ClusterFirst". Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'.
                  type: string
                domain:
                  description: Specification of the desired behavior of the VirtualMachineInstance on the host.
                  properties:
                    chassis:
                      description: Chassis specifies the chassis info passed to the domain.
                      properties:
                        asset:
                          type: string
                        manufacturer:
                          type: string
                        serial:
                          type: string
                        sku:
                          type: string
                        version:
                          type: string
                      type: object
                    clock:
                      description: Clock sets the clock and timers of the vmi.
                      properties:
                        timer:
                          description: Timer specifies whih timers are attached to the vmi.
                          properties:
                            hpet:
                              description: HPET (High Precision Event Timer) - multiple timers with periodic interrupts.
                              properties:
                                present:
                                  description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                  type: boolean
                                tickPolicy:
                                  description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup", "merge", "discard".
                                  type: string
                              type: object
                            hyperv:
                              description: Hyperv (Hypervclock) - lets guests read the host’s wall clock time (paravirtualized). For windows guests.
                              properties:
                                present:
                                  description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                  type: boolean
                              type: object
                            kvm:
                              description: "KVM \t(KVM clock) - lets guests read the host’s wall clock time (paravirtualized). For linux guests."
                              properties:
                                present:
                                  description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                  type: boolean
                              type: object
                            pit:
                              description: PIT (Programmable Interval Timer) - a timer with periodic interrupts.
                              properties:
                                present:
                                  description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                  type: boolean
                                tickPolicy:
                                  description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup", "discard".
                                  type: string
                              type: object
                            rtc:
                              description: RTC (Real Time Clock) - a continuously running timer with periodic interrupts.
                              properties:
                                present:
                                  description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                  type: boolean
                                tickPolicy:
                                  description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup".
                                  type: string
                                track:
                                  description: Track the guest or the wall clock.
                                  type: string
                              type: object
                          type: object
                        timezone:
                          description: Timezone sets the guest clock to the specified timezone. Zone name follows the TZ environment variable format (e.g. 'America/New_York').
                          type: string
                        utc:
                          description: UTC sets the guest clock to UTC on each boot. If an offset is specified, guest changes to the clock will be kept during reboots and are not reset.
                          properties:
                            offsetSeconds:
                              description: OffsetSeconds specifies an offset in seconds, relative to UTC. If set, guest changes to the clock will be kept during reboots and not reset.
                              type: integer
                          type: object
                      type: object
                    cpu:
                      description: CPU allow specified the detailed CPU topology inside the vmi.
                      properties:
                        cores:
                          description: Cores specifies the number of cores inside the vmi. Must be a value greater or equal 1.
                          format: int32
                          type: integer
                        dedicatedCpuPlacement:
                          description: DedicatedCPUPlacement requests the scheduler to place the VirtualMachineInstance on a node with enough dedicated pCPUs and pin the vCPUs to it.
                          type: boolean
                        features:
                          description: Features specifies the CPU features list inside the VMI.
                          items:
                            description: CPUFeature allows specifying a CPU feature.
                            properties:
                              name:
                                description: Name of the CPU feature
                                type: string
                              policy:
                                description: 'Policy is the CPU feature attribute which can have the following attributes: force    - The virtual CPU will claim the feature is supported regardless of it being supported by host CPU. require  - Guest creation will fail unless the feature is supported by the host CPU or the hypervisor is able to emulate it. optional - The feature will be supported by virtual CPU if and only if it is supported by host CPU. disable  - The feature will not be supported by virtual CPU. forbid   - Guest creation will fail if the feature is supported by host CPU. Defaults to require'
                                type: string
                            required:
                            - name
                            type: object
                          type: array
                        isolateEmulatorThread:
                          description: IsolateEmulatorThread requests one more dedicated pCPU to be allocated for the VMI to place the emulator thread on it.
                          type: boolean
                        model:
                          description: Model specifies the CPU model inside the VMI. List of available models https://github.com/libvirt/libvirt/tree/master/src/cpu_map. It is possible to specify special cases like "host-passthrough" to get the same CPU as the node and "host-model" to get CPU closest to the node one. Defaults to host-model.
                          type: string
                        sockets:
                          description: Sockets specifies the number of sockets inside the vmi. Must be a value greater or equal 1.
                          format: int32
                          type: integer
                        threads:
                          description: Threads specifies the number of threads inside the vmi. Must be a value greater or equal 1.
                          format: int32
                          type: integer
                      type: object
                    devices:
                      description: Devices allows adding disks, network interfaces, and others
                      properties:
                        autoattachGraphicsDevice:
                          description: Whether to attach the default graphics device or not. VNC will not be available if set to false. Defaults to true.
                          type: boolean
                        autoattachMemBalloon:
                          description: Whether to attach the Memory balloon device with default period. Period can be adjusted in virt-config. Defaults to true.
                          type: boolean
                        autoattachPodInterface:
                          description: Whether to attach a pod network interface. Defaults to true.
                          type: boolean
                        autoattachSerialConsole:
                          description: Whether to attach the default serial console or not. Serial console access will not be available if set to false. Defaults to true.
                          type: boolean
                        blockMultiQueue:
                          description: Whether or not to enable virtio multi-queue for block devices
                          type: boolean
                        disks:
                          description: Disks describes disks, cdroms, floppy and luns which are connected to the vmi.
                          items:
                            properties:
                              bootOrder:
                                description: BootOrder is an integer value > 0, used to determine ordering of boot devices. Lower values take precedence. Each disk or interface that has a boot order must have a unique value. Disks without a boot order are not tried if a disk with a boot order exists.
                                type: integer
                              cache:
                                description: Cache specifies which kvm disk cache mode should be used.
                                type: string
                              cdrom:
                                description: Attach a volume as a cdrom to the vmi.
                                properties:
                                  bus:
                                    description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                                    type: string
                                  readonly:
                                    description: ReadOnly. Defaults to true.
                                    type: boolean
                                  tray:
                                    description: Tray indicates if the tray of the device is open or closed. Allowed values are "open" and "closed". Defaults to closed.
                                    type: string
                                type: object
                              dedicatedIOThread:
                                description: dedicatedIOThread indicates this disk should have an exclusive IO Thread. Enabling this implies useIOThreads = true. Defaults to false.
                                type: boolean
                              disk:
                                description: Attach a volume as a disk to the vmi.
                                properties:
                                  bus:
                                    description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                                    type: string
                                  pciAddress:
                                    description: 'If specified, the virtual disk will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10'
                                    type: string
                                  readonly:
                                    description: ReadOnly. Defaults to false.
                                    type: boolean
                                type: object
                              floppy:
                                description: Attach a volume as a floppy to the vmi.
                                properties:
                                  readonly:
                                    description: ReadOnly. Defaults to false.
                                    type: boolean
                                  tray:
                                    description: Tray indicates if the tray of the device is open or closed. Allowed values are "open" and "closed". Defaults to closed.
                                    type: string
                                type: object
                              io:
                                description: 'IO specifies which QEMU disk IO mode should be used. Supported values are: native, default, threads.'
                                type: string
                              lun:
                                description: Attach a volume as a LUN to the vmi.
                                properties:
                                  bus:
                                    description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                                    type: string
                                  readonly:
                                    description: ReadOnly. Defaults to false.
                                    type: boolean
                                type: object
                              name:
                                description: Name is the device name
                                type: string
                              serial:
                                description: Serial provides the ability to specify a serial number for the disk device.
                                type: string
                              tag:
                                description: If specified, disk address and its tag will be provided to the guest via config drive metadata
                                type: string
                            required:
                            - name
                            type: object
                          type: array
                        filesystems:
                          description: Filesystems describes filesystem which is connected to the vmi.
                          items:
                            properties:
                              name:
                                description: Name is the device name
                                type: string
                              virtiofs:
                                description: Virtiofs is supported
                                type: object
                            required:
                            - name
                            - virtiofs
                            type: object
                          type: array
                        gpus:
                          description: Whether to attach a GPU device to the vmi.
                          items:
                            properties:
                              deviceName:
                                type: string
                              name:
                                description: Name of the GPU device as exposed by a device plugin
                                type: string
                            required:
                            - deviceName
                            - name
                            type: object
                          type: array
                        inputs:
                          description: Inputs describe input devices
                          items:
                            properties:
                              bus:
                                description: 'Bus indicates the bus of input device to emulate. Supported values: virtio, usb.'
                                type: string
                              name:
                                description: Name is the device name
                                type: string
                              type:
                                description: 'Type indicated the type of input device. Supported values: tablet.'
                                type: string
                            required:
                            - name
                            - type
                            type: object
                          type: array
                        interfaces:
                          description: Interfaces describe network interfaces which are added to the vmi.
                          items:
                            properties:
                              bootOrder:
                                description: BootOrder is an integer value > 0, used to determine ordering of boot devices. Lower values take precedence. Each interface or disk that has a boot order must have a unique value. Interfaces without a boot order are not tried.
                                type: integer
                              bridge:
                                type: object
                              dhcpOptions:
                                description: If specified the network interface will pass additional DHCP options to the VMI
                                properties:
                                  bootFileName:
                                    description: If specified will pass option 67 to interface's DHCP server
                                    type: string
                                  ntpServers:
                                    description: If specified will pass the configured NTP server to the VM via DHCP option 042.
                                    items:
                                      type: string
                                    type: array
                                  privateOptions:
                                    description: 'If specified will pass extra DHCP options for private use, range: 224-254'
                                    items:
                                      description: DHCPExtraOptions defines Extra DHCP options for a VM.
                                      properties:
                                        option:
                                          description: Option is an Integer value from 224-254 Required.
                                          type: integer
                                        value:
                                          description: Value is a String value for the Option provided Required.
                                          type: string
                                      required:
                                      - option
                                      - value
                                      type: object
                                    type: array
                                  tftpServerName:
                                    description: If specified will pass option 66 to interface's DHCP server
                                    type: string
                                type: object
                              macAddress:
                                description: 'Interface MAC address. For example: de:ad:00:00:be:af or DE-AD-00-00-BE-AF.'
                                type: string
                              masquerade:
                                type: object
                              model:
                                description: 'Interface model. One of: e1000, e1000e, ne2k_pci, pcnet, rtl8139, virtio. Defaults to virtio. TODO:(ihar) switch to enums once opengen-api supports them. See: https://github.com/kubernetes/kube-openapi/issues/51'
                                type: string
                              name:
                                description: Logical name of the interface as well as a reference to the associated networks. Must match the Name of a Network.
                                type: string
                              pciAddress:
                                description: 'If specified, the virtual network interface will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10'
                                type: string
                              ports:
                                description: List of ports to be forwarded to the virtual machine.
                                items:
                                  description: Port repesents a port to expose from the virtual machine. Default protocol TCP. The port field is mandatory
                                  properties:
                                    name:
                                      description: If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.
                                      type: string
                                    port:
                                      description: Number of port to expose for the virtual machine. This must be a valid port number, 0 < x < 65536.
                                      format: int32
                                      type: integer
                                    protocol:
                                      description: Protocol for port. Must be UDP or TCP. Defaults to "TCP".
                                      type: string
                                  required:
                                  - port
                                  type: object
                                type: array
                              slirp:
                                type: object
                              sriov:
                                type: object
                              tag:
                                description: If specified, the virtual network interface address and its tag will be provided to the guest via config drive
                                type: string
                            required:
                            - name
                            type: object
                          type: array
                        networkInterfaceMultiqueue:
                          description: If specified, virtual network interfaces configured with a virtio bus will also enable the vhost multiqueue feature for network devices. The number of queues created depends on additional factors of the VirtualMachineInstance, like the number of guest CPUs.
                          type: boolean
                        rng:
                          description: Whether to have random number generator from host
                          type: object
                        watchdog:
                          description: Watchdog describes a watchdog device which can be added to the vmi.
                          properties:
                            i6300esb:
                              description: i6300esb watchdog device.
                              properties:
                                action:
                                  description: The action to take. Valid values are poweroff, reset, shutdown. Defaults to reset.
                                  type: string
                              type: object
                            name:
                              description: Name of the watchdog.
                              type: string
                          required:
                          - name
                          type: object
                      type: object
                    features:
                      description: Features like acpi, apic, hyperv, smm.
                      properties:
                        acpi:
                          description: ACPI enables/disables ACPI inside the guest. Defaults to enabled.
                          properties:
                            enabled:
                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                              type: boolean
                          type: object
                        apic:
                          description: Defaults to the machine type setting.
                          properties:
                            enabled:
                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                              type: boolean
                            endOfInterrupt:
                              description: EndOfInterrupt enables the end of interrupt notification in the guest. Defaults to false.
                              type: boolean
                          type: object
                        hyperv:
                          description: Defaults to the machine type setting.
                          properties:
                            evmcs:
                              description: EVMCS Speeds up L2 vmexits, but disables other virtualization features. Requires vapic. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            frequencies:
                              description: Frequencies improves the TSC clock source handling for Hyper-V on KVM. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            ipi:
                              description: IPI improves performances in overcommited environments. Requires vpindex. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            reenlightenment:
                              description: Reenlightenment enables the notifications on TSC frequency changes. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            relaxed:
                              description: Relaxed instructs the guest OS to disable watchdog timeouts. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            reset:
                              description: Reset enables Hyperv reboot/reset for the vmi. Requires synic. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            runtime:
                              description: Runtime improves the time accounting to improve scheduling in the guest. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            spinlocks:
                              description: Spinlocks allows to configure the spinlock retry attempts.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                                spinlocks:
                                  description: Retries indicates the number of retries. Must be a value greater or equal 4096. Defaults to 4096.
                                  format: int32
                                  type: integer
                              type: object
                            synic:
                              description: SyNIC enables the Synthetic Interrupt Controller. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            synictimer:
                              description: SyNICTimer enables Synthetic Interrupt Controller Timers, reducing CPU load. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            tlbflush:
                              description: TLBFlush improves performances in overcommited environments. Requires vpindex. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            vapic:
                              description: VAPIC improves the paravirtualized handling of interrupts. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            vendorid:
                              description: VendorID allows setting the hypervisor vendor id. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                                vendorid:
                                  description: VendorID sets the hypervisor vendor id, visible to the vmi. String up to twelve characters.
                                  type: string
                              type: object
                            vpindex:
                              description: VPIndex enables the Virtual Processor Index to help windows identifying virtual processors. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                          type: object
                        kvm:
                          description: Configure how KVM presence is exposed to the guest.
                          properties:
                            hidden:
                              description: Hide the KVM hypervisor from standard MSR based discovery. Defaults to false
                              type: boolean
                          type: object
                        smm:
                          description: SMM enables/disables System Management Mode. TSEG not yet implemented.
                          properties:
                            enabled:
                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                              type: boolean
                          type: object
                      type: object
                    firmware:
                      description: Firmware.
                      properties:
                        bootloader:
                          description: Settings to control the bootloader that is used.
                          properties:
                            bios:
                              description: If set (default), BIOS will be used.
                              properties:
                                useSerial:
                                  description: If set, the BIOS output will be transmitted over serial
                                  type: boolean
                              type: object
                            efi:
                              description: If set, EFI will be used instead of BIOS.
                              properties:
                                secureBoot:
                                  description: If set, SecureBoot will be enabled and the OVMF roms will be swapped for SecureBoot-enabled ones. Requires SMM to be enabled. Defaults to true
                                  type: boolean
                              type: object
                          type: object
                        serial:
                          description: The system-serial-number in SMBIOS
                          type: string
                        uuid:
                          description: UUID reported by the vmi bios. Defaults to a random generated uid.
                          type: string
                      type: object
                    ioThreadsPolicy:
                      description: 'Controls whether or not disks will share IOThreads. Omitting IOThreadsPolicy disables use of IOThreads. One of: shared, auto'
                      type: string
                    machine:
                      description: Machine type.
                      properties:
                        type:
                          description: QEMU machine type is the actual chipset of the VirtualMachineInstance.
                          type: string
                      required:
                      - type
                      type: object
                    memory:
                      description: Memory allow specifying the VMI memory features.
                      properties:
                        guest:
                          anyOf:
                          - type: integer
                          - type: string
                          description: Guest allows to specifying the amount of memory which is visible inside the Guest OS. The Guest must lie between Requests and Limits from the resources section. Defaults to the requested memory in the resources section if not specified.
                          pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                          x-kubernetes-int-or-string: true
                        hugepages:
                          description: Hugepages allow to use hugepages for the VirtualMachineInstance instead of regular memory.
                          properties:
                            pageSize:
                              description: PageSize specifies the hugepage size, for x86_64 architecture valid values are 1Gi and 2Mi.
                              type: string
                          type: object
                      type: object
                    resources:
                      description: Resources describes the Compute Resources required by this vmi.
                      properties:
                        limits:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Limits describes the maximum amount of compute resources allowed. Valid resource keys are "memory" and "cpu".
                          type: object
                        overcommitGuestOverhead:
                          description: Don't ask the scheduler to take the guest-management overhead into account. Instead put the overhead only into the container's memory limit. This can lead to crashes if all memory is in use on a node. Defaults to false.
                          type: boolean
                        requests:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Requests is a description of the initial vmi resources. Valid resource keys are "memory" and "cpu".
                          type: object
                      type: object
                  required:
                  - devices
                  type: object
                evictionStrategy:
                  description: EvictionStrategy can be set to "LiveMigrate" if the VirtualMachineInstance should be migrated instead of shut-off in case of a node drain.
                  type: string
                hostname:
                  description: Specifies the hostname of the vmi If not specified, the hostname will be set to the name of the vmi, if dhcp or cloud-init is configured properly.
                  type: string
                livenessProbe:
                  description: 'Periodic probe of VirtualMachineInstance liveness. VirtualmachineInstances will be stopped if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                  properties:
                    failureThreshold:
                      description: Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.
                      format: int32
                      type: integer
                    httpGet:
                      description: HTTPGet specifies the http request to perform.
                      properties:
                        host:
                          description: Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.
                          type: string
                        httpHeaders:
                          description: Custom headers to set in the request. HTTP allows repeated headers.
                          items:
                            description: HTTPHeader describes a custom header to be used in HTTP probes
                            properties:
                              name:
                                description: The header field name
                                type: string
                              value:
                                description: The header field value
                                type: string
                            required:
                            - name
                            - value
                            type: object
                          type: array
                        path:
                          description: Path to access on the HTTP server.
                          type: string
                        port:
                          anyOf:
                          - type: integer
                          - type: string
                          description: Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                          x-kubernetes-int-or-string: true
                        scheme:
                          description: Scheme to use for connecting to the host. Defaults to HTTP.
                          type: string
                      required:
                      - port
                      type: object
                    initialDelaySeconds:
                      description: 'Number of seconds after the VirtualMachineInstance has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                      format: int32
                      type: integer
                    periodSeconds:
                      description: How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.
                      format: int32
                      type: integer
                    successThreshold:
                      description: Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness. Minimum value is 1.
                      format: int32
                      type: integer
                    tcpSocket:
                      description: 'TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported TODO: implement a realistic TCP lifecycle hook'
                      properties:
                        host:
                          description: 'Optional: Host name to connect to, defaults to the pod IP.'
                          type: string
                        port:
                          anyOf:
                          - type: integer
                          - type: string
                          description: Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                          x-kubernetes-int-or-string: true
                      required:
                      - port
                      type: object
                    timeoutSeconds:
                      description: 'Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                      format: int32
                      type: integer
                  type: object
                networks:
                  description: List of networks that can be attached to a vm's virtual interface.
                  items:
                    description: Network represents a network type and a resource that should be connected to the vm.
                    properties:
                      multus:
                        description: Represents the multus cni network.
                        properties:
                          default:
                            description: Select the default network and add it to the multus-cni.io/default-network annotation.
                            type: boolean
                          networkName:
                            description: 'References to a NetworkAttachmentDefinition CRD object. Format: <networkName>, <namespace>/<networkName>. If namespace is not specified, VMI namespace is assumed.'
                            type: string
                        required:
                        - networkName
                        type: object
                      name:
                        description: 'Network name. Must be a DNS_LABEL and unique within the vm. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names'
                        type: string
                      pod:
                        description: Represents the stock pod network interface.
                        properties:
                          vmNetworkCIDR:
                            description: CIDR for vm network. Default 10.0.2.0/24 if not specified.
                            type: string
                        type: object
                    required:
                    - name
                    type: object
                  type: array
                nodeSelector:
                  additionalProperties:
                    type: string
                  description: 'NodeSelector is a selector which must be true for the vmi to fit on a node. Selector which must match a node''s labels for the vmi to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/'
                  type: object
                priorityClassName:
                  description: If specified, indicates the pod's priority. If not specified, the pod priority will be default or zero if there is no default.
                  type: string
                readinessProbe:
                  description: 'Periodic probe of VirtualMachineInstance service readiness. VirtualmachineInstances will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                  properties:
                    failureThreshold:
                      description: Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.
                      format: int32
                      type: integer
                    httpGet:
                      description: HTTPGet specifies the http request to perform.
                      properties:
                        host:
                          description: Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.
                          type: string
                        httpHeaders:
                          description: Custom headers to set in the request. HTTP allows repeated headers.
                          items:
                            description: HTTPHeader describes a custom header to be used in HTTP probes
                            properties:
                              name:
                                description: The header field name
                                type: string
                              value:
                                description: The header field value
                                type: string
                            required:
                            - name
                            - value
                            type: object
                          type: array
                        path:
                          description: Path to access on the HTTP server.
                          type: string
                        port:
                          anyOf:
                          - type: integer
                          - type: string
                          description: Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                          x-kubernetes-int-or-string: true
                        scheme:
                          description: Scheme to use for connecting to the host. Defaults to HTTP.
                          type: string
                      required:
                      - port
                      type: object
                    initialDelaySeconds:
                      description: 'Number of seconds after the VirtualMachineInstance has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                      format: int32
                      type: integer
                    periodSeconds:
                      description: How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.
                      format: int32
                      type: integer
                    successThreshold:
                      description: Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness. Minimum value is 1.
                      format: int32
                      type: integer
                    tcpSocket:
                      description: 'TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported TODO: implement a realistic TCP lifecycle hook'
                      properties:
                        host:
                          description: 'Optional: Host name to connect to, defaults to the pod IP.'
                          type: string
                        port:
                          anyOf:
                          - type: integer
                          - type: string
                          description: Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                          x-kubernetes-int-or-string: true
                      required:
                      - port
                      type: object
                    timeoutSeconds:
                      description: 'Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                      format: int32
                      type: integer
                  type: object
                schedulerName:
                  description: If specified, the VMI will be dispatched by specified scheduler. If not specified, the VMI will be dispatched by default scheduler.
                  type: string
                subdomain:
                  description: If specified, the fully qualified vmi hostname will be "<hostname>.<subdomain>.<pod namespace>.svc.<cluster domain>". If not specified, the vmi will not have a domainname at all. The DNS entry will resolve to the vmi, no matter if the vmi itself can pick up a hostname.
                  type: string
                terminationGracePeriodSeconds:
                  description: Grace period observed after signalling a VirtualMachineInstance to stop after which the VirtualMachineInstance is force terminated.
                  format: int64
                  type: integer
                tolerations:
                  description: If toleration is specified, obey all the toleration rules.
                  items:
                    description: The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.
                    properties:
                      effect:
                        description: Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.
                        type: string
                      key:
                        description: Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.
                        type: string
                      operator:
                        description: Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.
                        type: string
                      tolerationSeconds:
                        description: TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.
                        format: int64
                        type: integer
                      value:
                        description: Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.
                        type: string
                    type: object
                  type: array
                volumes:
                  description: List of volumes that can be mounted by disks belonging to the vmi.
                  items:
                    description: Volume represents a named volume in a vmi.
                    properties:
                      cloudInitConfigDrive:
                        description: 'CloudInitConfigDrive represents a cloud-init Config Drive user-data source. The Config Drive data will be added as a disk to the vmi. A proper cloud-init installation is required inside the guest. More info: https://cloudinit.readthedocs.io/en/latest/topics/datasources/configdrive.html'
                        properties:
                          networkData:
                            description: NetworkData contains config drive inline cloud-init networkdata.
                            type: string
                          networkDataBase64:
                            description: NetworkDataBase64 contains config drive cloud-init networkdata as a base64 encoded string.
                            type: string
                          networkDataSecretRef:
                            description: NetworkDataSecretRef references a k8s secret that contains config drive networkdata.
                            properties:
                              name:
                                description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                type: string
                            type: object
                          secretRef:
                            description: UserDataSecretRef references a k8s secret that contains config drive userdata.
                            properties:
                              name:
                                description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                type: string
                            type: object
                          userData:
                            description: UserData contains config drive inline cloud-init userdata.
                            type: string
                          userDataBase64:
                            description: UserDataBase64 contains config drive cloud-init userdata as a base64 encoded string.
                            type: string
                        type: object
                      cloudInitNoCloud:
                        description: 'CloudInitNoCloud represents a cloud-init NoCloud user-data source. The NoCloud data will be added as a disk to the vmi. A proper cloud-init installation is required inside the guest. More info: http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html'
                        properties:
                          networkData:
                            description: NetworkData contains NoCloud inline cloud-init networkdata.
                            type: string
                          networkDataBase64:
                            description: NetworkDataBase64 contains NoCloud cloud-init networkdata as a base64 encoded string.
                            type: string
                          networkDataSecretRef:
                            description: NetworkDataSecretRef references a k8s secret that contains NoCloud networkdata.
                            properties:
                              name:
                                description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                type: string
                            type: object
                          secretRef:
                            description: UserDataSecretRef references a k8s secret that contains NoCloud userdata.
                            properties:
                              name:
                                description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                type: string
                            type: object
                          userData:
                            description: UserData contains NoCloud inline cloud-init userdata.
                            type: string
                          userDataBase64:
                            description: UserDataBase64 contains NoCloud cloud-init userdata as a base64 encoded string.
                            type: string
                        type: object
                      configMap:
                        description: 'ConfigMapSource represents a reference to a ConfigMap in the same namespace. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/'
                        properties:
                          name:
                            description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                            type: string
                          optional:
                            description: Specify whether the ConfigMap or it's keys must be defined
                            type: boolean
                          volumeLabel:
                            description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                            type: string
                        type: object
                      containerDisk:
                        description: 'ContainerDisk references a docker image, embedding a qcow or raw disk. More info: https://kubevirt.gitbooks.io/user-guide/registry-disk.html'
                        properties:
                          image:
                            description: Image is the name of the image with the embedded disk.
                            type: string
                          imagePullPolicy:
                            description: 'Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images'
                            type: string
                          imagePullSecret:
                            description: ImagePullSecret is the name of the Docker registry secret required to pull the image. The secret must already exist.
                            type: string
                          path:
                            description: Path defines the path to disk file in the container
                            type: string
                        required:
                        - image
                        type: object
                      dataVolume:
                        description: DataVolume represents the dynamic creation a PVC for this volume as well as the process of populating that PVC with a disk image.
                        properties:
                          name:
                            description: Name represents the name of the DataVolume in the same namespace
                            type: string
                        required:
                        - name
                        type: object
                      downwardAPI:
                        description: DownwardAPI represents downward API about the pod that should populate this volume
                        properties:
                          fields:
                            description: Fields is a list of downward API volume file
                            items:
                              description: DownwardAPIVolumeFile represents information to create the file containing the pod field
                              properties:
                                fieldRef:
                                  description: 'Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.'
                                  properties:
                                    apiVersion:
                                      description: Version of the schema the FieldPath is written in terms of, defaults to "v1".
                                      type: string
                                    fieldPath:
                                      description: Path of the field to select in the specified API version.
                                      type: string
                                  required:
                                  - fieldPath
                                  type: object
                                mode:
                                  description: 'Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.'
                                  format: int32
                                  type: integer
                                path:
                                  description: 'Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the ''..'' path. Must be utf-8 encoded. The first item of the relative path must not start with ''..'''
                                  type: string
                                resourceFieldRef:
                                  description: 'Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.'
                                  properties:
                                    containerName:
                                      description: 'Container name: required for volumes, optional for env vars'
                                      type: string
                                    divisor:
                                      anyOf:
                                      - type: integer
                                      - type: string
                                      description: Specifies the output format of the exposed resources, defaults to "1"
                                      pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                                      x-kubernetes-int-or-string: true
                                    resource:
                                      description: 'Required: resource to select'
                                      type: string
                                  required:
                                  - resource
                                  type: object
                              required:
                              - path
                              type: object
                            type: array
                          volumeLabel:
                            description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                            type: string
                        type: object
                      emptyDisk:
                        description: 'EmptyDisk represents a temporary disk which shares the vmis lifecycle. More info: https://kubevirt.gitbooks.io/user-guide/disks-and-volumes.html'
                        properties:
                          capacity:
                            anyOf:
                            - type: integer
                            - type: string
                            description: Capacity of the sparse disk.
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                        required:
                        - capacity
                        type: object
                      ephemeral:
                        description: Ephemeral is a special volume source that "wraps" specified source and provides copy-on-write image on top of it.
                        properties:
                          persistentVolumeClaim:
                            description: 'PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. Directly attached to the vmi via qemu. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                            properties:
                              claimName:
                                description: 'ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                                type: string
                              readOnly:
                                description: Will force the ReadOnly setting in VolumeMounts. Default false.
                                type: boolean
                            required:
                            - claimName
                            type: object
                        type: object
                      hostDisk:
                        description: HostDisk represents a disk created on the cluster level
                        properties:
                          capacity:
                            anyOf:
                            - type: integer
                            - type: string
                            description: Capacity of the sparse disk
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          path:
                            description: The path to HostDisk image located on the cluster
                            type: string
                          shared:
                            description: Shared indicate whether the path is shared between nodes
                            type: boolean
                          type:
                            description: Contains information if disk.img exists or should be created allowed options are 'Disk' and 'DiskOrCreate'
                            type: string
                        required:
                        - path
                        - type
                        type: object
                      name:
                        description: 'Volume''s name. Must be a DNS_LABEL and unique within the vmi. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names'
                        type: string
                      persistentVolumeClaim:
                        description: 'PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. Directly attached to the vmi via qemu. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                        properties:
                          claimName:
                            description: 'ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                            type: string
                          readOnly:
                            description: Will force the ReadOnly setting in VolumeMounts. Default false.
                            type: boolean
                        required:
                        - claimName
                        type: object
                      secret:
                        description: 'SecretVolumeSource represents a reference to a secret data in the same namespace. More info: https://kubernetes.io/docs/concepts/configuration/secret/'
                        properties:
                          optional:
                            description: Specify whether the Secret or it's keys must be defined
                            type: boolean
                          secretName:
                            description: 'Name of the secret in the pod''s namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret'
                            type: string
                          volumeLabel:
                            description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                            type: string
                        type: object
                      serviceAccount:
                        description: 'ServiceAccountVolumeSource represents a reference to a service account. There can only be one volume of this type! More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/'
                        properties:
                          serviceAccountName:
                            description: 'Name of the service account in the pod''s namespace to use. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/'
                            type: string
                        type: object
                    required:
                    - name
                    type: object
                  type: array
              required:
              - domain
              type: object
          type: object
      required:
      - template
      type: object
    status:
      description: Status holds the current state of the controller and brief information about its associated VirtualMachineInstance
      properties:
        conditions:
          description: Hold the state information of the VirtualMachine and its VirtualMachineInstance
          items:
            description: VirtualMachineCondition represents the state of VirtualMachine
            properties:
              lastProbeTime:
                format: date-time
                nullable: true
                type: string
              lastTransitionTime:
                format: date-time
                nullable: true
                type: string
              message:
                type: string
              reason:
                type: string
              status:
                type: string
              type:
                type: string
            required:
            - status
            - type
            type: object
          type: array
        created:
          description: Created indicates if the virtual machine is created in the cluster
          type: boolean
        ready:
          description: Ready indicates if the virtual machine is running and ready
          type: boolean
        snapshotInProgress:
          description: SnapshotInProgress is the name of the VirtualMachineSnapshot currently executing
          type: string
        stateChangeRequests:
          description: StateChangeRequests indicates a list of actions that should be taken on a VMI e.g. stop a specific VMI then start a new one.
          items:
            properties:
              action:
                description: Indicates the type of action that is requested. e.g. Start or Stop
                type: string
              data:
                additionalProperties:
                  type: string
                description: Provides additional data in order to perform the Action
                type: object
              uid:
                description: Indicates the UUID of an existing Virtual Machine Instance that this change request applies to -- if applicable
                type: string
            required:
            - action
            type: object
          type: array
        volumeSnapshotStatuses:
          description: VolumeSnapshotStatuses indicates a list of statuses whether snapshotting is supported by each volume.
          items:
            properties:
              enabled:
                description: True if the volume supports snapshotting
                type: boolean
              name:
                description: Volume name
                type: string
              reason:
                description: Empty if snapshotting is enabled, contains reason otherwise
                type: string
            required:
            - enabled
            - name
            type: object
          type: array
      type: object
  required:
  - spec
  type: object
`,
	"virtualmachineinstance": `openAPIV3Schema:
  description: VirtualMachineInstance is *the* VirtualMachineInstance Definition. It represents a virtual machine in the runtime environment of kubernetes.
  properties:
    apiVersion:
      description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
      type: string
    kind:
      description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
      type: string
    metadata:
      type: object
    spec:
      description: VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
      properties:
        affinity:
          description: If affinity is specifies, obey all the affinity rules
          properties:
            nodeAffinity:
              description: Describes node affinity scheduling rules for the pod.
              properties:
                preferredDuringSchedulingIgnoredDuringExecution:
                  description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.
                  items:
                    description: An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).
                    properties:
                      preference:
                        description: A node selector term, associated with the corresponding weight.
                        properties:
                          matchExpressions:
                            description: A list of node selector requirements by node's labels.
                            items:
                              description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                              properties:
                                key:
                                  description: The label key that the selector applies to.
                                  type: string
                                operator:
                                  description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                  type: string
                                values:
                                  description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                  items:
                                    type: string
                                  type: array
                              required:
                              - key
                              - operator
                              type: object
                            type: array
                          matchFields:
                            description: A list of node selector requirements by node's fields.
                            items:
                              description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                              properties:
                                key:
                                  description: The label key that the selector applies to.
                                  type: string
                                operator:
                                  description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                  type: string
                                values:
                                  description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                  items:
                                    type: string
                                  type: array
                              required:
                              - key
                              - operator
                              type: object
                            type: array
                        type: object
                      weight:
                        description: Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.
                        format: int32
                        type: integer
                    required:
                    - preference
                    - weight
                    type: object
                  type: array
                requiredDuringSchedulingIgnoredDuringExecution:
                  description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.
                  properties:
                    nodeSelectorTerms:
                      description: Required. A list of node selector terms. The terms are ORed.
                      items:
                        description: A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.
                        properties:
                          matchExpressions:
                            description: A list of node selector requirements by node's labels.
                            items:
                              description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                              properties:
                                key:
                                  description: The label key that the selector applies to.
                                  type: string
                                operator:
                                  description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                  type: string
                                values:
                                  description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                  items:
                                    type: string
                                  type: array
                              required:
                              - key
                              - operator
                              type: object
                            type: array
                          matchFields:
                            description: A list of node selector requirements by node's fields.
                            items:
                              description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                              properties:
                                key:
                                  description: The label key that the selector applies to.
                                  type: string
                                operator:
                                  description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                  type: string
                                values:
                                  description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                  items:
                                    type: string
                                  type: array
                              required:
                              - key
                              - operator
                              type: object
                            type: array
                        type: object
                      type: array
                  required:
                  - nodeSelectorTerms
                  type: object
              type: object
            podAffinity:
              description: Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).
              properties:
                preferredDuringSchedulingIgnoredDuringExecution:
                  description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                  items:
                    description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                    properties:
                      podAffinityTerm:
                        description: Required. A pod affinity term, associated with the corresponding weight.
                        properties:
                          labelSelector:
                            description: A label query over a set of resources, in this case pods.
                            properties:
                              matchExpressions:
                                description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                items:
                                  description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                  properties:
                                    key:
                                      description: key is the label key that the selector applies to.
                                      type: string
                                    operator:
                                      description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                      type: string
                                    values:
                                      description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                      items:
                                        type: string
                                      type: array
                                  required:
                                  - key
                                  - operator
                                  type: object
                                type: array
                              matchLabels:
                                additionalProperties:
                                  type: string
                                description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                type: object
                            type: object
                          namespaces:
                            description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                            items:
                              type: string
                            type: array
                          topologyKey:
                            description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                            type: string
                        required:
                        - topologyKey
                        type: object
                      weight:
                        description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                        format: int32
                        type: integer
                    required:
                    - podAffinityTerm
                    - weight
                    type: object
                  type: array
                requiredDuringSchedulingIgnoredDuringExecution:
                  description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                  items:
                    description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                    properties:
                      labelSelector:
                        description: A label query over a set of resources, in this case pods.
                        properties:
                          matchExpressions:
                            description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                            items:
                              description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                              properties:
                                key:
                                  description: key is the label key that the selector applies to.
                                  type: string
                                operator:
                                  description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                  type: string
                                values:
                                  description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                  items:
                                    type: string
                                  type: array
                              required:
                              - key
                              - operator
                              type: object
                            type: array
                          matchLabels:
                            additionalProperties:
                              type: string
                            description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                            type: object
                        type: object
                      namespaces:
                        description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                        items:
                          type: string
                        type: array
                      topologyKey:
                        description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                        type: string
                    required:
                    - topologyKey
                    type: object
                  type: array
              type: object
            podAntiAffinity:
              description: Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).
              properties:
                preferredDuringSchedulingIgnoredDuringExecution:
                  description: The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                  items:
                    description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                    properties:
                      podAffinityTerm:
                        description: Required. A pod affinity term, associated with the corresponding weight.
                        properties:
                          labelSelector:
                            description: A label query over a set of resources, in this case pods.
                            properties:
                              matchExpressions:
                                description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                items:
                                  description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                  properties:
                                    key:
                                      description: key is the label key that the selector applies to.
                                      type: string
                                    operator:
                                      description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                      type: string
                                    values:
                                      description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                      items:
                                        type: string
                                      type: array
                                  required:
                                  - key
                                  - operator
                                  type: object
                                type: array
                              matchLabels:
                                additionalProperties:
                                  type: string
                                description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                type: object
                            type: object
                          namespaces:
                            description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                            items:
                              type: string
                            type: array
                          topologyKey:
                            description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                            type: string
                        required:
                        - topologyKey
                        type: object
                      weight:
                        description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                        format: int32
                        type: integer
                    required:
                    - podAffinityTerm
                    - weight
                    type: object
                  type: array
                requiredDuringSchedulingIgnoredDuringExecution:
                  description: If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                  items:
                    description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                    properties:
                      labelSelector:
                        description: A label query over a set of resources, in this case pods.
                        properties:
                          matchExpressions:
                            description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                            items:
                              description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                              properties:
                                key:
                                  description: key is the label key that the selector applies to.
                                  type: string
                                operator:
                                  description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                  type: string
                                values:
                                  description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                  items:
                                    type: string
                                  type: array
                              required:
                              - key
                              - operator
                              type: object
                            type: array
                          matchLabels:
                            additionalProperties:
                              type: string
                            description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                            type: object
                        type: object
                      namespaces:
                        description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                        items:
                          type: string
                        type: array
                      topologyKey:
                        description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                        type: string
                    required:
                    - topologyKey
                    type: object
                  type: array
              type: object
          type: object
        dnsConfig:
          description: Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS configuration based on DNSPolicy.
          properties:
            nameservers:
              description: A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed.
              items:
                type: string
              type: array
            options:
              description: A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy.
              items:
                description: PodDNSConfigOption defines DNS resolver options of a pod.
                properties:
                  name:
                    description: Required.
                    type: string
                  value:
                    type: string
                type: object
              type: array
            searches:
              description: A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed.
              items:
                type: string
              type: array
          type: object
        dnsPolicy:
          description: Set DNS policy for the pod. Defaults to "ClusterFirst". Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'.
          type: string
        domain:
          description: Specification of the desired behavior of the VirtualMachineInstance on the host.
          properties:
            chassis:
              description: Chassis specifies the chassis info passed to the domain.
              properties:
                asset:
                  type: string
                manufacturer:
                  type: string
                serial:
                  type: string
                sku:
                  type: string
                version:
                  type: string
              type: object
            clock:
              description: Clock sets the clock and timers of the vmi.
              properties:
                timer:
                  description: Timer specifies whih timers are attached to the vmi.
                  properties:
                    hpet:
                      description: HPET (High Precision Event Timer) - multiple timers with periodic interrupts.
                      properties:
                        present:
                          description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                          type: boolean
                        tickPolicy:
                          description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup", "merge", "discard".
                          type: string
                      type: object
                    hyperv:
                      description: Hyperv (Hypervclock) - lets guests read the host’s wall clock time (paravirtualized). For windows guests.
                      properties:
                        present:
                          description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                          type: boolean
                      type: object
                    kvm:
                      description: "KVM \t(KVM clock) - lets guests read the host’s wall clock time (paravirtualized). For linux guests."
                      properties:
                        present:
                          description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                          type: boolean
                      type: object
                    pit:
                      description: PIT (Programmable Interval Timer) - a timer with periodic interrupts.
                      properties:
                        present:
                          description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                          type: boolean
                        tickPolicy:
                          description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup", "discard".
                          type: string
                      type: object
                    rtc:
                      description: RTC (Real Time Clock) - a continuously running timer with periodic interrupts.
                      properties:
                        present:
                          description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                          type: boolean
                        tickPolicy:
                          description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup".
                          type: string
                        track:
                          description: Track the guest or the wall clock.
                          type: string
                      type: object
                  type: object
                timezone:
                  description: Timezone sets the guest clock to the specified timezone. Zone name follows the TZ environment variable format (e.g. 'America/New_York').
                  type: string
                utc:
                  description: UTC sets the guest clock to UTC on each boot. If an offset is specified, guest changes to the clock will be kept during reboots and are not reset.
                  properties:
                    offsetSeconds:
                      description: OffsetSeconds specifies an offset in seconds, relative to UTC. If set, guest changes to the clock will be kept during reboots and not reset.
                      type: integer
                  type: object
              type: object
            cpu:
              description: CPU allow specified the detailed CPU topology inside the vmi.
              properties:
                cores:
                  description: Cores specifies the number of cores inside the vmi. Must be a value greater or equal 1.
                  format: int32
                  type: integer
                dedicatedCpuPlacement:
                  description: DedicatedCPUPlacement requests the scheduler to place the VirtualMachineInstance on a node with enough dedicated pCPUs and pin the vCPUs to it.
                  type: boolean
                features:
                  description: Features specifies the CPU features list inside the VMI.
                  items:
                    description: CPUFeature allows specifying a CPU feature.
                    properties:
                      name:
                        description: Name of the CPU feature
                        type: string
                      policy:
                        description: 'Policy is the CPU feature attribute which can have the following attributes: force    - The virtual CPU will claim the feature is supported regardless of it being supported by host CPU. require  - Guest creation will fail unless the feature is supported by the host CPU or the hypervisor is able to emulate it. optional - The feature will be supported by virtual CPU if and only if it is supported by host CPU. disable  - The feature will not be supported by virtual CPU. forbid   - Guest creation will fail if the feature is supported by host CPU. Defaults to require'
                        type: string
                    required:
                    - name
                    type: object
                  type: array
                isolateEmulatorThread:
                  description: IsolateEmulatorThread requests one more dedicated pCPU to be allocated for the VMI to place the emulator thread on it.
                  type: boolean
                model:
                  description: Model specifies the CPU model inside the VMI. List of available models https://github.com/libvirt/libvirt/tree/master/src/cpu_map. It is possible to specify special cases like "host-passthrough" to get the same CPU as the node and "host-model" to get CPU closest to the node one. Defaults to host-model.
                  type: string
                sockets:
                  description: Sockets specifies the number of sockets inside the vmi. Must be a value greater or equal 1.
                  format: int32
                  type: integer
                threads:
                  description: Threads specifies the number of threads inside the vmi. Must be a value greater or equal 1.
                  format: int32
                  type: integer
              type: object
            devices:
              description: Devices allows adding disks, network interfaces, and others
              properties:
                autoattachGraphicsDevice:
                  description: Whether to attach the default graphics device or not. VNC will not be available if set to false. Defaults to true.
                  type: boolean
                autoattachMemBalloon:
                  description: Whether to attach the Memory balloon device with default period. Period can be adjusted in virt-config. Defaults to true.
                  type: boolean
                autoattachPodInterface:
                  description: Whether to attach a pod network interface. Defaults to true.
                  type: boolean
                autoattachSerialConsole:
                  description: Whether to attach the default serial console or not. Serial console access will not be available if set to false. Defaults to true.
                  type: boolean
                blockMultiQueue:
                  description: Whether or not to enable virtio multi-queue for block devices
                  type: boolean
                disks:
                  description: Disks describes disks, cdroms, floppy and luns which are connected to the vmi.
                  items:
                    properties:
                      bootOrder:
                        description: BootOrder is an integer value > 0, used to determine ordering of boot devices. Lower values take precedence. Each disk or interface that has a boot order must have a unique value. Disks without a boot order are not tried if a disk with a boot order exists.
                        type: integer
                      cache:
                        description: Cache specifies which kvm disk cache mode should be used.
                        type: string
                      cdrom:
                        description: Attach a volume as a cdrom to the vmi.
                        properties:
                          bus:
                            description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                            type: string
                          readonly:
                            description: ReadOnly. Defaults to true.
                            type: boolean
                          tray:
                            description: Tray indicates if the tray of the device is open or closed. Allowed values are "open" and "closed". Defaults to closed.
                            type: string
                        type: object
                      dedicatedIOThread:
                        description: dedicatedIOThread indicates this disk should have an exclusive IO Thread. Enabling this implies useIOThreads = true. Defaults to false.
                        type: boolean
                      disk:
                        description: Attach a volume as a disk to the vmi.
                        properties:
                          bus:
                            description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                            type: string
                          pciAddress:
                            description: 'If specified, the virtual disk will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10'
                            type: string
                          readonly:
                            description: ReadOnly. Defaults to false.
                            type: boolean
                        type: object
                      floppy:
                        description: Attach a volume as a floppy to the vmi.
                        properties:
                          readonly:
                            description: ReadOnly. Defaults to false.
                            type: boolean
                          tray:
                            description: Tray indicates if the tray of the device is open or closed. Allowed values are "open" and "closed". Defaults to closed.
                            type: string
                        type: object
                      io:
                        description: 'IO specifies which QEMU disk IO mode should be used. Supported values are: native, default, threads.'
                        type: string
                      lun:
                        description: Attach a volume as a LUN to the vmi.
                        properties:
                          bus:
                            description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                            type: string
                          readonly:
                            description: ReadOnly. Defaults to false.
                            type: boolean
                        type: object
                      name:
                        description: Name is the device name
                        type: string
                      serial:
                        description: Serial provides the ability to specify a serial number for the disk device.
                        type: string
                      tag:
                        description: If specified, disk address and its tag will be provided to the guest via config drive metadata
                        type: string
                    required:
                    - name
                    type: object
                  type: array
                filesystems:
                  description: Filesystems describes filesystem which is connected to the vmi.
                  items:
                    properties:
                      name:
                        description: Name is the device name
                        type: string
                      virtiofs:
                        description: Virtiofs is supported
                        type: object
                    required:
                    - name
                    - virtiofs
                    type: object
                  type: array
                gpus:
                  description: Whether to attach a GPU device to the vmi.
                  items:
                    properties:
                      deviceName:
                        type: string
                      name:
                        description: Name of the GPU device as exposed by a device plugin
                        type: string
                    required:
                    - deviceName
                    - name
                    type: object
                  type: array
                inputs:
                  description: Inputs describe input devices
                  items:
                    properties:
                      bus:
                        description: 'Bus indicates the bus of input device to emulate. Supported values: virtio, usb.'
                        type: string
                      name:
                        description: Name is the device name
                        type: string
                      type:
                        description: 'Type indicated the type of input device. Supported values: tablet.'
                        type: string
                    required:
                    - name
                    - type
                    type: object
                  type: array
                interfaces:
                  description: Interfaces describe network interfaces which are added to the vmi.
                  items:
                    properties:
                      bootOrder:
                        description: BootOrder is an integer value > 0, used to determine ordering of boot devices. Lower values take precedence. Each interface or disk that has a boot order must have a unique value. Interfaces without a boot order are not tried.
                        type: integer
                      bridge:
                        type: object
                      dhcpOptions:
                        description: If specified the network interface will pass additional DHCP options to the VMI
                        properties:
                          bootFileName:
                            description: If specified will pass option 67 to interface's DHCP server
                            type: string
                          ntpServers:
                            description: If specified will pass the configured NTP server to the VM via DHCP option 042.
                            items:
                              type: string
                            type: array
                          privateOptions:
                            description: 'If specified will pass extra DHCP options for private use, range: 224-254'
                            items:
                              description: DHCPExtraOptions defines Extra DHCP options for a VM.
                              properties:
                                option:
                                  description: Option is an Integer value from 224-254 Required.
                                  type: integer
                                value:
                                  description: Value is a String value for the Option provided Required.
                                  type: string
                              required:
                              - option
                              - value
                              type: object
                            type: array
                          tftpServerName:
                            description: If specified will pass option 66 to interface's DHCP server
                            type: string
                        type: object
                      macAddress:
                        description: 'Interface MAC address. For example: de:ad:00:00:be:af or DE-AD-00-00-BE-AF.'
                        type: string
                      masquerade:
                        type: object
                      model:
                        description: 'Interface model. One of: e1000, e1000e, ne2k_pci, pcnet, rtl8139, virtio. Defaults to virtio. TODO:(ihar) switch to enums once opengen-api supports them. See: https://github.com/kubernetes/kube-openapi/issues/51'
                        type: string
                      name:
                        description: Logical name of the interface as well as a reference to the associated networks. Must match the Name of a Network.
                        type: string
                      pciAddress:
                        description: 'If specified, the virtual network interface will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10'
                        type: string
                      ports:
                        description: List of ports to be forwarded to the virtual machine.
                        items:
                          description: Port repesents a port to expose from the virtual machine. Default protocol TCP. The port field is mandatory
                          properties:
                            name:
                              description: If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.
                              type: string
                            port:
                              description: Number of port to expose for the virtual machine. This must be a valid port number, 0 < x < 65536.
                              format: int32
                              type: integer
                            protocol:
                              description: Protocol for port. Must be UDP or TCP. Defaults to "TCP".
                              type: string
                          required:
                          - port
                          type: object
                        type: array
                      slirp:
                        type: object
                      sriov:
                        type: object
                      tag:
                        description: If specified, the virtual network interface address and its tag will be provided to the guest via config drive
                        type: string
                    required:
                    - name
                    type: object
                  type: array
                networkInterfaceMultiqueue:
                  description: If specified, virtual network interfaces configured with a virtio bus will also enable the vhost multiqueue feature for network devices. The number of queues created depends on additional factors of the VirtualMachineInstance, like the number of guest CPUs.
                  type: boolean
                rng:
                  description: Whether to have random number generator from host
                  type: object
                watchdog:
                  description: Watchdog describes a watchdog device which can be added to the vmi.
                  properties:
                    i6300esb:
                      description: i6300esb watchdog device.
                      properties:
                        action:
                          description: The action to take. Valid values are poweroff, reset, shutdown. Defaults to reset.
                          type: string
                      type: object
                    name:
                      description: Name of the watchdog.
                      type: string
                  required:
                  - name
                  type: object
              type: object
            features:
              description: Features like acpi, apic, hyperv, smm.
              properties:
                acpi:
                  description: ACPI enables/disables ACPI inside the guest. Defaults to enabled.
                  properties:
                    enabled:
                      description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                      type: boolean
                  type: object
                apic:
                  description: Defaults to the machine type setting.
                  properties:
                    enabled:
                      description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                      type: boolean
                    endOfInterrupt:
                      description: EndOfInterrupt enables the end of interrupt notification in the guest. Defaults to false.
                      type: boolean
                  type: object
                hyperv:
                  description: Defaults to the machine type setting.
                  properties:
                    evmcs:
                      description: EVMCS Speeds up L2 vmexits, but disables other virtualization features. Requires vapic. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    frequencies:
                      description: Frequencies improves the TSC clock source handling for Hyper-V on KVM. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    ipi:
                      description: IPI improves performances in overcommited environments. Requires vpindex. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    reenlightenment:
                      description: Reenlightenment enables the notifications on TSC frequency changes. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    relaxed:
                      description: Relaxed instructs the guest OS to disable watchdog timeouts. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    reset:
                      description: Reset enables Hyperv reboot/reset for the vmi. Requires synic. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    runtime:
                      description: Runtime improves the time accounting to improve scheduling in the guest. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    spinlocks:
                      description: Spinlocks allows to configure the spinlock retry attempts.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                        spinlocks:
                          description: Retries indicates the number of retries. Must be a value greater or equal 4096. Defaults to 4096.
                          format: int32
                          type: integer
                      type: object
                    synic:
                      description: SyNIC enables the Synthetic Interrupt Controller. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    synictimer:
                      description: SyNICTimer enables Synthetic Interrupt Controller Timers, reducing CPU load. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    tlbflush:
                      description: TLBFlush improves performances in overcommited environments. Requires vpindex. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    vapic:
                      description: VAPIC improves the paravirtualized handling of interrupts. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    vendorid:
                      description: VendorID allows setting the hypervisor vendor id. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                        vendorid:
                          description: VendorID sets the hypervisor vendor id, visible to the vmi. String up to twelve characters.
                          type: string
                      type: object
                    vpindex:
                      description: VPIndex enables the Virtual Processor Index to help windows identifying virtual processors. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                  type: object
                kvm:
                  description: Configure how KVM presence is exposed to the guest.
                  properties:
                    hidden:
                      description: Hide the KVM hypervisor from standard MSR based discovery. Defaults to false
                      type: boolean
                  type: object
                smm:
                  description: SMM enables/disables System Management Mode. TSEG not yet implemented.
                  properties:
                    enabled:
                      description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                      type: boolean
                  type: object
              type: object
            firmware:
              description: Firmware.
              properties:
                bootloader:
                  description: Settings to control the bootloader that is used.
                  properties:
                    bios:
                      description: If set (default), BIOS will be used.
                      properties:
                        useSerial:
                          description: If set, the BIOS output will be transmitted over serial
                          type: boolean
                      type: object
                    efi:
                      description: If set, EFI will be used instead of BIOS.
                      properties:
                        secureBoot:
                          description: If set, SecureBoot will be enabled and the OVMF roms will be swapped for SecureBoot-enabled ones. Requires SMM to be enabled. Defaults to true
                          type: boolean
                      type: object
                  type: object
                serial:
                  description: The system-serial-number in SMBIOS
                  type: string
                uuid:
                  description: UUID reported by the vmi bios. Defaults to a random generated uid.
                  type: string
              type: object
            ioThreadsPolicy:
              description: 'Controls whether or not disks will share IOThreads. Omitting IOThreadsPolicy disables use of IOThreads. One of: shared, auto'
              type: string
            machine:
              description: Machine type.
              properties:
                type:
                  description: QEMU machine type is the actual chipset of the VirtualMachineInstance.
                  type: string
              required:
              - type
              type: object
            memory:
              description: Memory allow specifying the VMI memory features.
              properties:
                guest:
                  anyOf:
                  - type: integer
                  - type: string
                  description: Guest allows to specifying the amount of memory which is visible inside the Guest OS. The Guest must lie between Requests and Limits from the resources section. Defaults to the requested memory in the resources section if not specified.
                  pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                  x-kubernetes-int-or-string: true
                hugepages:
                  description: Hugepages allow to use hugepages for the VirtualMachineInstance instead of regular memory.
                  properties:
                    pageSize:
                      description: PageSize specifies the hugepage size, for x86_64 architecture valid values are 1Gi and 2Mi.
                      type: string
                  type: object
              type: object
            resources:
              description: Resources describes the Compute Resources required by this vmi.
              properties:
                limits:
                  additionalProperties:
                    anyOf:
                    - type: integer
                    - type: string
                    pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                    x-kubernetes-int-or-string: true
                  description: Limits describes the maximum amount of compute resources allowed. Valid resource keys are "memory" and "cpu".
                  type: object
                overcommitGuestOverhead:
                  description: Don't ask the scheduler to take the guest-management overhead into account. Instead put the overhead only into the container's memory limit. This can lead to crashes if all memory is in use on a node. Defaults to false.
                  type: boolean
                requests:
                  additionalProperties:
                    anyOf:
                    - type: integer
                    - type: string
                    pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                    x-kubernetes-int-or-string: true
                  description: Requests is a description of the initial vmi resources. Valid resource keys are "memory" and "cpu".
                  type: object
              type: object
          required:
          - devices
          type: object
        evictionStrategy:
          description: EvictionStrategy can be set to "LiveMigrate" if the VirtualMachineInstance should be migrated instead of shut-off in case of a node drain.
          type: string
        hostname:
          description: Specifies the hostname of the vmi If not specified, the hostname will be set to the name of the vmi, if dhcp or cloud-init is configured properly.
          type: string
        livenessProbe:
          description: 'Periodic probe of VirtualMachineInstance liveness. VirtualmachineInstances will be stopped if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
          properties:
            failureThreshold:
              description: Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.
              format: int32
              type: integer
            httpGet:
              description: HTTPGet specifies the http request to perform.
              properties:
                host:
                  description: Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.
                  type: string
                httpHeaders:
                  description: Custom headers to set in the request. HTTP allows repeated headers.
                  items:
                    description: HTTPHeader describes a custom header to be used in HTTP probes
                    properties:
                      name:
                        description: The header field name
                        type: string
                      value:
                        description: The header field value
                        type: string
                    required:
                    - name
                    - value
                    type: object
                  type: array
                path:
                  description: Path to access on the HTTP server.
                  type: string
                port:
                  anyOf:
                  - type: integer
                  - type: string
                  description: Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                  x-kubernetes-int-or-string: true
                scheme:
                  description: Scheme to use for connecting to the host. Defaults to HTTP.
                  type: string
              required:
              - port
              type: object
            initialDelaySeconds:
              description: 'Number of seconds after the VirtualMachineInstance has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
              format: int32
              type: integer
            periodSeconds:
              description: How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.
              format: int32
              type: integer
            successThreshold:
              description: Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness. Minimum value is 1.
              format: int32
              type: integer
            tcpSocket:
              description: 'TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported TODO: implement a realistic TCP lifecycle hook'
              properties:
                host:
                  description: 'Optional: Host name to connect to, defaults to the pod IP.'
                  type: string
                port:
                  anyOf:
                  - type: integer
                  - type: string
                  description: Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                  x-kubernetes-int-or-string: true
              required:
              - port
              type: object
            timeoutSeconds:
              description: 'Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
              format: int32
              type: integer
          type: object
        networks:
          description: List of networks that can be attached to a vm's virtual interface.
          items:
            description: Network represents a network type and a resource that should be connected to the vm.
            properties:
              multus:
                description: Represents the multus cni network.
                properties:
                  default:
                    description: Select the default network and add it to the multus-cni.io/default-network annotation.
                    type: boolean
                  networkName:
                    description: 'References to a NetworkAttachmentDefinition CRD object. Format: <networkName>, <namespace>/<networkName>. If namespace is not specified, VMI namespace is assumed.'
                    type: string
                required:
                - networkName
                type: object
              name:
                description: 'Network name. Must be a DNS_LABEL and unique within the vm. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names'
                type: string
              pod:
                description: Represents the stock pod network interface.
                properties:
                  vmNetworkCIDR:
                    description: CIDR for vm network. Default 10.0.2.0/24 if not specified.
                    type: string
                type: object
            required:
            - name
            type: object
          type: array
        nodeSelector:
          additionalProperties:
            type: string
          description: 'NodeSelector is a selector which must be true for the vmi to fit on a node. Selector which must match a node''s labels for the vmi to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/'
          type: object
        priorityClassName:
          description: If specified, indicates the pod's priority. If not specified, the pod priority will be default or zero if there is no default.
          type: string
        readinessProbe:
          description: 'Periodic probe of VirtualMachineInstance service readiness. VirtualmachineInstances will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
          properties:
            failureThreshold:
              description: Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.
              format: int32
              type: integer
            httpGet:
              description: HTTPGet specifies the http request to perform.
              properties:
                host:
                  description: Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.
                  type: string
                httpHeaders:
                  description: Custom headers to set in the request. HTTP allows repeated headers.
                  items:
                    description: HTTPHeader describes a custom header to be used in HTTP probes
                    properties:
                      name:
                        description: The header field name
                        type: string
                      value:
                        description: The header field value
                        type: string
                    required:
                    - name
                    - value
                    type: object
                  type: array
                path:
                  description: Path to access on the HTTP server.
                  type: string
                port:
                  anyOf:
                  - type: integer
                  - type: string
                  description: Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                  x-kubernetes-int-or-string: true
                scheme:
                  description: Scheme to use for connecting to the host. Defaults to HTTP.
                  type: string
              required:
              - port
              type: object
            initialDelaySeconds:
              description: 'Number of seconds after the VirtualMachineInstance has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
              format: int32
              type: integer
            periodSeconds:
              description: How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.
              format: int32
              type: integer
            successThreshold:
              description: Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness. Minimum value is 1.
              format: int32
              type: integer
            tcpSocket:
              description: 'TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported TODO: implement a realistic TCP lifecycle hook'
              properties:
                host:
                  description: 'Optional: Host name to connect to, defaults to the pod IP.'
                  type: string
                port:
                  anyOf:
                  - type: integer
                  - type: string
                  description: Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                  x-kubernetes-int-or-string: true
              required:
              - port
              type: object
            timeoutSeconds:
              description: 'Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
              format: int32
              type: integer
          type: object
        schedulerName:
          description: If specified, the VMI will be dispatched by specified scheduler. If not specified, the VMI will be dispatched by default scheduler.
          type: string
        subdomain:
          description: If specified, the fully qualified vmi hostname will be "<hostname>.<subdomain>.<pod namespace>.svc.<cluster domain>". If not specified, the vmi will not have a domainname at all. The DNS entry will resolve to the vmi, no matter if the vmi itself can pick up a hostname.
          type: string
        terminationGracePeriodSeconds:
          description: Grace period observed after signalling a VirtualMachineInstance to stop after which the VirtualMachineInstance is force terminated.
          format: int64
          type: integer
        tolerations:
          description: If toleration is specified, obey all the toleration rules.
          items:
            description: The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.
            properties:
              effect:
                description: Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.
                type: string
              key:
                description: Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.
                type: string
              operator:
                description: Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.
                type: string
              tolerationSeconds:
                description: TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.
                format: int64
                type: integer
              value:
                description: Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.
                type: string
            type: object
          type: array
        volumes:
          description: List of volumes that can be mounted by disks belonging to the vmi.
          items:
            description: Volume represents a named volume in a vmi.
            properties:
              cloudInitConfigDrive:
                description: 'CloudInitConfigDrive represents a cloud-init Config Drive user-data source. The Config Drive data will be added as a disk to the vmi. A proper cloud-init installation is required inside the guest. More info: https://cloudinit.readthedocs.io/en/latest/topics/datasources/configdrive.html'
                properties:
                  networkData:
                    description: NetworkData contains config drive inline cloud-init networkdata.
                    type: string
                  networkDataBase64:
                    description: NetworkDataBase64 contains config drive cloud-init networkdata as a base64 encoded string.
                    type: string
                  networkDataSecretRef:
                    description: NetworkDataSecretRef references a k8s secret that contains config drive networkdata.
                    properties:
                      name:
                        description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                        type: string
                    type: object
                  secretRef:
                    description: UserDataSecretRef references a k8s secret that contains config drive userdata.
                    properties:
                      name:
                        description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                        type: string
                    type: object
                  userData:
                    description: UserData contains config drive inline cloud-init userdata.
                    type: string
                  userDataBase64:
                    description: UserDataBase64 contains config drive cloud-init userdata as a base64 encoded string.
                    type: string
                type: object
              cloudInitNoCloud:
                description: 'CloudInitNoCloud represents a cloud-init NoCloud user-data source. The NoCloud data will be added as a disk to the vmi. A proper cloud-init installation is required inside the guest. More info: http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html'
                properties:
                  networkData:
                    description: NetworkData contains NoCloud inline cloud-init networkdata.
                    type: string
                  networkDataBase64:
                    description: NetworkDataBase64 contains NoCloud cloud-init networkdata as a base64 encoded string.
                    type: string
                  networkDataSecretRef:
                    description: NetworkDataSecretRef references a k8s secret that contains NoCloud networkdata.
                    properties:
                      name:
                        description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                        type: string
                    type: object
                  secretRef:
                    description: UserDataSecretRef references a k8s secret that contains NoCloud userdata.
                    properties:
                      name:
                        description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                        type: string
                    type: object
                  userData:
                    description: UserData contains NoCloud inline cloud-init userdata.
                    type: string
                  userDataBase64:
                    description: UserDataBase64 contains NoCloud cloud-init userdata as a base64 encoded string.
                    type: string
                type: object
              configMap:
                description: 'ConfigMapSource represents a reference to a ConfigMap in the same namespace. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/'
                properties:
                  name:
                    description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                    type: string
                  optional:
                    description: Specify whether the ConfigMap or it's keys must be defined
                    type: boolean
                  volumeLabel:
                    description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                    type: string
                type: object
              containerDisk:
                description: 'ContainerDisk references a docker image, embedding a qcow or raw disk. More info: https://kubevirt.gitbooks.io/user-guide/registry-disk.html'
                properties:
                  image:
                    description: Image is the name of the image with the embedded disk.
                    type: string
                  imagePullPolicy:
                    description: 'Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images'
                    type: string
                  imagePullSecret:
                    description: ImagePullSecret is the name of the Docker registry secret required to pull the image. The secret must already exist.
                    type: string
                  path:
                    description: Path defines the path to disk file in the container
                    type: string
                required:
                - image
                type: object
              dataVolume:
                description: DataVolume represents the dynamic creation a PVC for this volume as well as the process of populating that PVC with a disk image.
                properties:
                  name:
                    description: Name represents the name of the DataVolume in the same namespace
                    type: string
                required:
                - name
                type: object
              downwardAPI:
                description: DownwardAPI represents downward API about the pod that should populate this volume
                properties:
                  fields:
                    description: Fields is a list of downward API volume file
                    items:
                      description: DownwardAPIVolumeFile represents information to create the file containing the pod field
                      properties:
                        fieldRef:
                          description: 'Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.'
                          properties:
                            apiVersion:
                              description: Version of the schema the FieldPath is written in terms of, defaults to "v1".
                              type: string
                            fieldPath:
                              description: Path of the field to select in the specified API version.
                              type: string
                          required:
                          - fieldPath
                          type: object
                        mode:
                          description: 'Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.'
                          format: int32
                          type: integer
                        path:
                          description: 'Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the ''..'' path. Must be utf-8 encoded. The first item of the relative path must not start with ''..'''
                          type: string
                        resourceFieldRef:
                          description: 'Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.'
                          properties:
                            containerName:
                              description: 'Container name: required for volumes, optional for env vars'
                              type: string
                            divisor:
                              anyOf:
                              - type: integer
                              - type: string
                              description: Specifies the output format of the exposed resources, defaults to "1"
                              pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                              x-kubernetes-int-or-string: true
                            resource:
                              description: 'Required: resource to select'
                              type: string
                          required:
                          - resource
                          type: object
                      required:
                      - path
                      type: object
                    type: array
                  volumeLabel:
                    description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                    type: string
                type: object
              emptyDisk:
                description: 'EmptyDisk represents a temporary disk which shares the vmis lifecycle. More info: https://kubevirt.gitbooks.io/user-guide/disks-and-volumes.html'
                properties:
                  capacity:
                    anyOf:
                    - type: integer
                    - type: string
                    description: Capacity of the sparse disk.
                    pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                    x-kubernetes-int-or-string: true
                required:
                - capacity
                type: object
              ephemeral:
                description: Ephemeral is a special volume source that "wraps" specified source and provides copy-on-write image on top of it.
                properties:
                  persistentVolumeClaim:
                    description: 'PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. Directly attached to the vmi via qemu. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                    properties:
                      claimName:
                        description: 'ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                        type: string
                      readOnly:
                        description: Will force the ReadOnly setting in VolumeMounts. Default false.
                        type: boolean
                    required:
                    - claimName
                    type: object
                type: object
              hostDisk:
                description: HostDisk represents a disk created on the cluster level
                properties:
                  capacity:
                    anyOf:
                    - type: integer
                    - type: string
                    description: Capacity of the sparse disk
                    pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                    x-kubernetes-int-or-string: true
                  path:
                    description: The path to HostDisk image located on the cluster
                    type: string
                  shared:
                    description: Shared indicate whether the path is shared between nodes
                    type: boolean
                  type:
                    description: Contains information if disk.img exists or should be created allowed options are 'Disk' and 'DiskOrCreate'
                    type: string
                required:
                - path
                - type
                type: object
              name:
                description: 'Volume''s name. Must be a DNS_LABEL and unique within the vmi. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names'
                type: string
              persistentVolumeClaim:
                description: 'PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. Directly attached to the vmi via qemu. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                properties:
                  claimName:
                    description: 'ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                    type: string
                  readOnly:
                    description: Will force the ReadOnly setting in VolumeMounts. Default false.
                    type: boolean
                required:
                - claimName
                type: object
              secret:
                description: 'SecretVolumeSource represents a reference to a secret data in the same namespace. More info: https://kubernetes.io/docs/concepts/configuration/secret/'
                properties:
                  optional:
                    description: Specify whether the Secret or it's keys must be defined
                    type: boolean
                  secretName:
                    description: 'Name of the secret in the pod''s namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret'
                    type: string
                  volumeLabel:
                    description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                    type: string
                type: object
              serviceAccount:
                description: 'ServiceAccountVolumeSource represents a reference to a service account. There can only be one volume of this type! More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/'
                properties:
                  serviceAccountName:
                    description: 'Name of the service account in the pod''s namespace to use. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/'
                    type: string
                type: object
            required:
            - name
            type: object
          type: array
      required:
      - domain
      type: object
    status:
      description: Status is the high level overview of how the VirtualMachineInstance is doing. It contains information available to controllers and users.
      properties:
        activePods:
          additionalProperties:
            type: string
          description: ActivePods is a mapping of pod UID to node name. It is possible for multiple pods to be running for a single VMI during migration.
          type: object
        conditions:
          description: Conditions are specific points in VirtualMachineInstance's pod runtime.
          items:
            properties:
              lastProbeTime:
                format: date-time
                nullable: true
                type: string
              lastTransitionTime:
                format: date-time
                nullable: true
                type: string
              message:
                type: string
              reason:
                type: string
              status:
                type: string
              type:
                type: string
            required:
            - status
            - type
            type: object
          type: array
        evacuationNodeName:
          description: EvacuationNodeName is used to track the eviction process of a VMI. It stores the name of the node that we want to evacuate. It is meant to be used by KubeVirt core components only and can't be set or modified by users.
          type: string
        guestOSInfo:
          description: Guest OS Information
          properties:
            id:
              description: Guest OS Id
              type: string
            kernelRelease:
              description: Guest OS Kernel Release
              type: string
            kernelVersion:
              description: Kernel version of the Guest OS
              type: string
            machine:
              description: Machine type of the Guest OS
              type: string
            name:
              description: Name of the Guest OS
              type: string
            prettyName:
              description: Guest OS Pretty Name
              type: string
            version:
              description: Guest OS Version
              type: string
            versionId:
              description: Version ID of the Guest OS
              type: string
          type: object
        interfaces:
          description: Interfaces represent the details of available network interfaces.
          items:
            properties:
              interfaceName:
                description: The interface name inside the Virtual Machine
                type: string
              ipAddress:
                description: IP address of a Virtual Machine interface. It is always the first item of IPs
                type: string
              ipAddresses:
                description: List of all IP addresses of a Virtual Machine interface
                items:
                  type: string
                type: array
              mac:
                description: Hardware address of a Virtual Machine interface
                type: string
              name:
                description: 'Name of the interface, corresponds to name of the network assigned to the interface TODO: remove omitempty, when api breaking changes are allowed'
                type: string
            type: object
          type: array
        migrationMethod:
          description: 'Represents the method using which the vmi can be migrated: live migration or block migration'
          type: string
        migrationState:
          description: Represents the status of a live migration
          properties:
            abortRequested:
              description: Indicates that the migration has been requested to abort
              type: boolean
            abortStatus:
              description: Indicates the final status of the live migration abortion
              type: string
            completed:
              description: Indicates the migration completed
              type: boolean
            endTimestamp:
              description: The time the migration action ended
              format: date-time
              nullable: true
              type: string
            failed:
              description: Indicates that the migration failed
              type: boolean
            migrationUid:
              description: The VirtualMachineInstanceMigration object associated with this migration
              type: string
            mode:
              description: Lets us know if the vmi is currenly running pre or post copy migration
              type: string
            sourceNode:
              description: The source node that the VMI originated on
              type: string
            startTimestamp:
              description: The time the migration action began
              format: date-time
              nullable: true
              type: string
            targetDirectMigrationNodePorts:
              additionalProperties:
                type: integer
              description: The list of ports opened for live migration on the destination node
              type: object
            targetNode:
              description: The target node that the VMI is moving to
              type: string
            targetNodeAddress:
              description: The address of the target node to use for the migration
              type: string
            targetNodeDomainDetected:
              description: The Target Node has seen the Domain Start Event
              type: boolean
            targetPod:
              description: The target pod that the VMI is moving to
              type: string
          type: object
        nodeName:
          description: NodeName is the name where the VirtualMachineInstance is currently running.
          type: string
        phase:
          description: Phase is the status of the VirtualMachineInstance in kubernetes world. It is not the VirtualMachineInstance status, but partially correlates to it.
          type: string
        qosClass:
          description: 'The Quality of Service (QOS) classification assigned to the virtual machine instance based on resource requirements See PodQOSClass type for available QOS classes More info: https://git.k8s.io/community/contributors/design-proposals/node/resource-qos.md'
          type: string
        reason:
          description: A brief CamelCase message indicating details about why the VMI is in this state. e.g. 'NodeUnresponsive'
          type: string
      type: object
  required:
  - spec
  type: object
`,
	"virtualmachineinstancemigration": `openAPIV3Schema:
  description: VirtualMachineInstanceMigration represents the object tracking a VMI's migration to another host in the cluster
  properties:
    apiVersion:
      description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
      type: string
    kind:
      description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
      type: string
    metadata:
      type: object
    spec:
      properties:
        vmiName:
          description: The name of the VMI to perform the migration on. VMI must exist in the migration objects namespace
          type: string
      type: object
    status:
      description: VirtualMachineInstanceMigration reprents information pertaining to a VMI's migration.
      properties:
        conditions:
          items:
            properties:
              lastProbeTime:
                format: date-time
                nullable: true
                type: string
              lastTransitionTime:
                format: date-time
                nullable: true
                type: string
              message:
                type: string
              reason:
                type: string
              status:
                type: string
              type:
                type: string
            required:
            - status
            - type
            type: object
          type: array
        phase:
          description: VirtualMachineInstanceMigrationPhase is a label for the condition of a VirtualMachineInstanceMigration at the current time.
          type: string
      type: object
  required:
  - spec
  type: object
`,
	"virtualmachineinstancepreset": `openAPIV3Schema:
  properties:
    apiVersion:
      description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
      type: string
    kind:
      description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
      type: string
    metadata:
      type: object
    spec:
      description: VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
      properties:
        domain:
          description: Domain is the same object type as contained in VirtualMachineInstanceSpec
          properties:
            chassis:
              description: Chassis specifies the chassis info passed to the domain.
              properties:
                asset:
                  type: string
                manufacturer:
                  type: string
                serial:
                  type: string
                sku:
                  type: string
                version:
                  type: string
              type: object
            clock:
              description: Clock sets the clock and timers of the vmi.
              properties:
                timer:
                  description: Timer specifies whih timers are attached to the vmi.
                  properties:
                    hpet:
                      description: HPET (High Precision Event Timer) - multiple timers with periodic interrupts.
                      properties:
                        present:
                          description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                          type: boolean
                        tickPolicy:
                          description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup", "merge", "discard".
                          type: string
                      type: object
                    hyperv:
                      description: Hyperv (Hypervclock) - lets guests read the host’s wall clock time (paravirtualized). For windows guests.
                      properties:
                        present:
                          description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                          type: boolean
                      type: object
                    kvm:
                      description: "KVM \t(KVM clock) - lets guests read the host’s wall clock time (paravirtualized). For linux guests."
                      properties:
                        present:
                          description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                          type: boolean
                      type: object
                    pit:
                      description: PIT (Programmable Interval Timer) - a timer with periodic interrupts.
                      properties:
                        present:
                          description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                          type: boolean
                        tickPolicy:
                          description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup", "discard".
                          type: string
                      type: object
                    rtc:
                      description: RTC (Real Time Clock) - a continuously running timer with periodic interrupts.
                      properties:
                        present:
                          description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                          type: boolean
                        tickPolicy:
                          description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup".
                          type: string
                        track:
                          description: Track the guest or the wall clock.
                          type: string
                      type: object
                  type: object
                timezone:
                  description: Timezone sets the guest clock to the specified timezone. Zone name follows the TZ environment variable format (e.g. 'America/New_York').
                  type: string
                utc:
                  description: UTC sets the guest clock to UTC on each boot. If an offset is specified, guest changes to the clock will be kept during reboots and are not reset.
                  properties:
                    offsetSeconds:
                      description: OffsetSeconds specifies an offset in seconds, relative to UTC. If set, guest changes to the clock will be kept during reboots and not reset.
                      type: integer
                  type: object
              type: object
            cpu:
              description: CPU allow specified the detailed CPU topology inside the vmi.
              properties:
                cores:
                  description: Cores specifies the number of cores inside the vmi. Must be a value greater or equal 1.
                  format: int32
                  type: integer
                dedicatedCpuPlacement:
                  description: DedicatedCPUPlacement requests the scheduler to place the VirtualMachineInstance on a node with enough dedicated pCPUs and pin the vCPUs to it.
                  type: boolean
                features:
                  description: Features specifies the CPU features list inside the VMI.
                  items:
                    description: CPUFeature allows specifying a CPU feature.
                    properties:
                      name:
                        description: Name of the CPU feature
                        type: string
                      policy:
                        description: 'Policy is the CPU feature attribute which can have the following attributes: force    - The virtual CPU will claim the feature is supported regardless of it being supported by host CPU. require  - Guest creation will fail unless the feature is supported by the host CPU or the hypervisor is able to emulate it. optional - The feature will be supported by virtual CPU if and only if it is supported by host CPU. disable  - The feature will not be supported by virtual CPU. forbid   - Guest creation will fail if the feature is supported by host CPU. Defaults to require'
                        type: string
                    required:
                    - name
                    type: object
                  type: array
                isolateEmulatorThread:
                  description: IsolateEmulatorThread requests one more dedicated pCPU to be allocated for the VMI to place the emulator thread on it.
                  type: boolean
                model:
                  description: Model specifies the CPU model inside the VMI. List of available models https://github.com/libvirt/libvirt/tree/master/src/cpu_map. It is possible to specify special cases like "host-passthrough" to get the same CPU as the node and "host-model" to get CPU closest to the node one. Defaults to host-model.
                  type: string
                sockets:
                  description: Sockets specifies the number of sockets inside the vmi. Must be a value greater or equal 1.
                  format: int32
                  type: integer
                threads:
                  description: Threads specifies the number of threads inside the vmi. Must be a value greater or equal 1.
                  format: int32
                  type: integer
              type: object
            devices:
              description: Devices allows adding disks, network interfaces, and others
              properties:
                autoattachGraphicsDevice:
                  description: Whether to attach the default graphics device or not. VNC will not be available if set to false. Defaults to true.
                  type: boolean
                autoattachMemBalloon:
                  description: Whether to attach the Memory balloon device with default period. Period can be adjusted in virt-config. Defaults to true.
                  type: boolean
                autoattachPodInterface:
                  description: Whether to attach a pod network interface. Defaults to true.
                  type: boolean
                autoattachSerialConsole:
                  description: Whether to attach the default serial console or not. Serial console access will not be available if set to false. Defaults to true.
                  type: boolean
                blockMultiQueue:
                  description: Whether or not to enable virtio multi-queue for block devices
                  type: boolean
                disks:
                  description: Disks describes disks, cdroms, floppy and luns which are connected to the vmi.
                  items:
                    properties:
                      bootOrder:
                        description: BootOrder is an integer value > 0, used to determine ordering of boot devices. Lower values take precedence. Each disk or interface that has a boot order must have a unique value. Disks without a boot order are not tried if a disk with a boot order exists.
                        type: integer
                      cache:
                        description: Cache specifies which kvm disk cache mode should be used.
                        type: string
                      cdrom:
                        description: Attach a volume as a cdrom to the vmi.
                        properties:
                          bus:
                            description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                            type: string
                          readonly:
                            description: ReadOnly. Defaults to true.
                            type: boolean
                          tray:
                            description: Tray indicates if the tray of the device is open or closed. Allowed values are "open" and "closed". Defaults to closed.
                            type: string
                        type: object
                      dedicatedIOThread:
                        description: dedicatedIOThread indicates this disk should have an exclusive IO Thread. Enabling this implies useIOThreads = true. Defaults to false.
                        type: boolean
                      disk:
                        description: Attach a volume as a disk to the vmi.
                        properties:
                          bus:
                            description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                            type: string
                          pciAddress:
                            description: 'If specified, the virtual disk will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10'
                            type: string
                          readonly:
                            description: ReadOnly. Defaults to false.
                            type: boolean
                        type: object
                      floppy:
                        description: Attach a volume as a floppy to the vmi.
                        properties:
                          readonly:
                            description: ReadOnly. Defaults to false.
                            type: boolean
                          tray:
                            description: Tray indicates if the tray of the device is open or closed. Allowed values are "open" and "closed". Defaults to closed.
                            type: string
                        type: object
                      io:
                        description: 'IO specifies which QEMU disk IO mode should be used. Supported values are: native, default, threads.'
                        type: string
                      lun:
                        description: Attach a volume as a LUN to the vmi.
                        properties:
                          bus:
                            description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                            type: string
                          readonly:
                            description: ReadOnly. Defaults to false.
                            type: boolean
                        type: object
                      name:
                        description: Name is the device name
                        type: string
                      serial:
                        description: Serial provides the ability to specify a serial number for the disk device.
                        type: string
                      tag:
                        description: If specified, disk address and its tag will be provided to the guest via config drive metadata
                        type: string
                    required:
                    - name
                    type: object
                  type: array
                filesystems:
                  description: Filesystems describes filesystem which is connected to the vmi.
                  items:
                    properties:
                      name:
                        description: Name is the device name
                        type: string
                      virtiofs:
                        description: Virtiofs is supported
                        type: object
                    required:
                    - name
                    - virtiofs
                    type: object
                  type: array
                gpus:
                  description: Whether to attach a GPU device to the vmi.
                  items:
                    properties:
                      deviceName:
                        type: string
                      name:
                        description: Name of the GPU device as exposed by a device plugin
                        type: string
                    required:
                    - deviceName
                    - name
                    type: object
                  type: array
                inputs:
                  description: Inputs describe input devices
                  items:
                    properties:
                      bus:
                        description: 'Bus indicates the bus of input device to emulate. Supported values: virtio, usb.'
                        type: string
                      name:
                        description: Name is the device name
                        type: string
                      type:
                        description: 'Type indicated the type of input device. Supported values: tablet.'
                        type: string
                    required:
                    - name
                    - type
                    type: object
                  type: array
                interfaces:
                  description: Interfaces describe network interfaces which are added to the vmi.
                  items:
                    properties:
                      bootOrder:
                        description: BootOrder is an integer value > 0, used to determine ordering of boot devices. Lower values take precedence. Each interface or disk that has a boot order must have a unique value. Interfaces without a boot order are not tried.
                        type: integer
                      bridge:
                        type: object
                      dhcpOptions:
                        description: If specified the network interface will pass additional DHCP options to the VMI
                        properties:
                          bootFileName:
                            description: If specified will pass option 67 to interface's DHCP server
                            type: string
                          ntpServers:
                            description: If specified will pass the configured NTP server to the VM via DHCP option 042.
                            items:
                              type: string
                            type: array
                          privateOptions:
                            description: 'If specified will pass extra DHCP options for private use, range: 224-254'
                            items:
                              description: DHCPExtraOptions defines Extra DHCP options for a VM.
                              properties:
                                option:
                                  description: Option is an Integer value from 224-254 Required.
                                  type: integer
                                value:
                                  description: Value is a String value for the Option provided Required.
                                  type: string
                              required:
                              - option
                              - value
                              type: object
                            type: array
                          tftpServerName:
                            description: If specified will pass option 66 to interface's DHCP server
                            type: string
                        type: object
                      macAddress:
                        description: 'Interface MAC address. For example: de:ad:00:00:be:af or DE-AD-00-00-BE-AF.'
                        type: string
                      masquerade:
                        type: object
                      model:
                        description: 'Interface model. One of: e1000, e1000e, ne2k_pci, pcnet, rtl8139, virtio. Defaults to virtio. TODO:(ihar) switch to enums once opengen-api supports them. See: https://github.com/kubernetes/kube-openapi/issues/51'
                        type: string
                      name:
                        description: Logical name of the interface as well as a reference to the associated networks. Must match the Name of a Network.
                        type: string
                      pciAddress:
                        description: 'If specified, the virtual network interface will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10'
                        type: string
                      ports:
                        description: List of ports to be forwarded to the virtual machine.
                        items:
                          description: Port repesents a port to expose from the virtual machine. Default protocol TCP. The port field is mandatory
                          properties:
                            name:
                              description: If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.
                              type: string
                            port:
                              description: Number of port to expose for the virtual machine. This must be a valid port number, 0 < x < 65536.
                              format: int32
                              type: integer
                            protocol:
                              description: Protocol for port. Must be UDP or TCP. Defaults to "TCP".
                              type: string
                          required:
                          - port
                          type: object
                        type: array
                      slirp:
                        type: object
                      sriov:
                        type: object
                      tag:
                        description: If specified, the virtual network interface address and its tag will be provided to the guest via config drive
                        type: string
                    required:
                    - name
                    type: object
                  type: array
                networkInterfaceMultiqueue:
                  description: If specified, virtual network interfaces configured with a virtio bus will also enable the vhost multiqueue feature for network devices. The number of queues created depends on additional factors of the VirtualMachineInstance, like the number of guest CPUs.
                  type: boolean
                rng:
                  description: Whether to have random number generator from host
                  type: object
                watchdog:
                  description: Watchdog describes a watchdog device which can be added to the vmi.
                  properties:
                    i6300esb:
                      description: i6300esb watchdog device.
                      properties:
                        action:
                          description: The action to take. Valid values are poweroff, reset, shutdown. Defaults to reset.
                          type: string
                      type: object
                    name:
                      description: Name of the watchdog.
                      type: string
                  required:
                  - name
                  type: object
              type: object
            features:
              description: Features like acpi, apic, hyperv, smm.
              properties:
                acpi:
                  description: ACPI enables/disables ACPI inside the guest. Defaults to enabled.
                  properties:
                    enabled:
                      description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                      type: boolean
                  type: object
                apic:
                  description: Defaults to the machine type setting.
                  properties:
                    enabled:
                      description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                      type: boolean
                    endOfInterrupt:
                      description: EndOfInterrupt enables the end of interrupt notification in the guest. Defaults to false.
                      type: boolean
                  type: object
                hyperv:
                  description: Defaults to the machine type setting.
                  properties:
                    evmcs:
                      description: EVMCS Speeds up L2 vmexits, but disables other virtualization features. Requires vapic. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    frequencies:
                      description: Frequencies improves the TSC clock source handling for Hyper-V on KVM. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    ipi:
                      description: IPI improves performances in overcommited environments. Requires vpindex. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    reenlightenment:
                      description: Reenlightenment enables the notifications on TSC frequency changes. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    relaxed:
                      description: Relaxed instructs the guest OS to disable watchdog timeouts. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    reset:
                      description: Reset enables Hyperv reboot/reset for the vmi. Requires synic. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    runtime:
                      description: Runtime improves the time accounting to improve scheduling in the guest. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    spinlocks:
                      description: Spinlocks allows to configure the spinlock retry attempts.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                        spinlocks:
                          description: Retries indicates the number of retries. Must be a value greater or equal 4096. Defaults to 4096.
                          format: int32
                          type: integer
                      type: object
                    synic:
                      description: SyNIC enables the Synthetic Interrupt Controller. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    synictimer:
                      description: SyNICTimer enables Synthetic Interrupt Controller Timers, reducing CPU load. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    tlbflush:
                      description: TLBFlush improves performances in overcommited environments. Requires vpindex. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    vapic:
                      description: VAPIC improves the paravirtualized handling of interrupts. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                    vendorid:
                      description: VendorID allows setting the hypervisor vendor id. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                        vendorid:
                          description: VendorID sets the hypervisor vendor id, visible to the vmi. String up to twelve characters.
                          type: string
                      type: object
                    vpindex:
                      description: VPIndex enables the Virtual Processor Index to help windows identifying virtual processors. Defaults to the machine type setting.
                      properties:
                        enabled:
                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                          type: boolean
                      type: object
                  type: object
                kvm:
                  description: Configure how KVM presence is exposed to the guest.
                  properties:
                    hidden:
                      description: Hide the KVM hypervisor from standard MSR based discovery. Defaults to false
                      type: boolean
                  type: object
                smm:
                  description: SMM enables/disables System Management Mode. TSEG not yet implemented.
                  properties:
                    enabled:
                      description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                      type: boolean
                  type: object
              type: object
            firmware:
              description: Firmware.
              properties:
                bootloader:
                  description: Settings to control the bootloader that is used.
                  properties:
                    bios:
                      description: If set (default), BIOS will be used.
                      properties:
                        useSerial:
                          description: If set, the BIOS output will be transmitted over serial
                          type: boolean
                      type: object
                    efi:
                      description: If set, EFI will be used instead of BIOS.
                      properties:
                        secureBoot:
                          description: If set, SecureBoot will be enabled and the OVMF roms will be swapped for SecureBoot-enabled ones. Requires SMM to be enabled. Defaults to true
                          type: boolean
                      type: object
                  type: object
                serial:
                  description: The system-serial-number in SMBIOS
                  type: string
                uuid:
                  description: UUID reported by the vmi bios. Defaults to a random generated uid.
                  type: string
              type: object
            ioThreadsPolicy:
              description: 'Controls whether or not disks will share IOThreads. Omitting IOThreadsPolicy disables use of IOThreads. One of: shared, auto'
              type: string
            machine:
              description: Machine type.
              properties:
                type:
                  description: QEMU machine type is the actual chipset of the VirtualMachineInstance.
                  type: string
              required:
              - type
              type: object
            memory:
              description: Memory allow specifying the VMI memory features.
              properties:
                guest:
                  anyOf:
                  - type: integer
                  - type: string
                  description: Guest allows to specifying the amount of memory which is visible inside the Guest OS. The Guest must lie between Requests and Limits from the resources section. Defaults to the requested memory in the resources section if not specified.
                  pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                  x-kubernetes-int-or-string: true
                hugepages:
                  description: Hugepages allow to use hugepages for the VirtualMachineInstance instead of regular memory.
                  properties:
                    pageSize:
                      description: PageSize specifies the hugepage size, for x86_64 architecture valid values are 1Gi and 2Mi.
                      type: string
                  type: object
              type: object
            resources:
              description: Resources describes the Compute Resources required by this vmi.
              properties:
                limits:
                  additionalProperties:
                    anyOf:
                    - type: integer
                    - type: string
                    pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                    x-kubernetes-int-or-string: true
                  description: Limits describes the maximum amount of compute resources allowed. Valid resource keys are "memory" and "cpu".
                  type: object
                overcommitGuestOverhead:
                  description: Don't ask the scheduler to take the guest-management overhead into account. Instead put the overhead only into the container's memory limit. This can lead to crashes if all memory is in use on a node. Defaults to false.
                  type: boolean
                requests:
                  additionalProperties:
                    anyOf:
                    - type: integer
                    - type: string
                    pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                    x-kubernetes-int-or-string: true
                  description: Requests is a description of the initial vmi resources. Valid resource keys are "memory" and "cpu".
                  type: object
              type: object
          required:
          - devices
          type: object
        selector:
          description: Selector is a label query over a set of VMIs. Required.
          properties:
            matchExpressions:
              description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
              items:
                description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                properties:
                  key:
                    description: key is the label key that the selector applies to.
                    type: string
                  operator:
                    description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                    type: string
                  values:
                    description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                    items:
                      type: string
                    type: array
                required:
                - key
                - operator
                type: object
              type: array
            matchLabels:
              additionalProperties:
                type: string
              description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
              type: object
          type: object
      required:
      - selector
      type: object
  type: object
`,
	"virtualmachineinstancereplicaset": `openAPIV3Schema:
  description: VirtualMachineInstance is *the* VirtualMachineInstance Definition. It represents a virtual machine in the runtime environment of kubernetes.
  properties:
    apiVersion:
      description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
      type: string
    kind:
      description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
      type: string
    metadata:
      type: object
    spec:
      description: VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
      properties:
        paused:
          description: Indicates that the replica set is paused.
          type: boolean
        replicas:
          description: Number of desired pods. This is a pointer to distinguish between explicit zero and not specified. Defaults to 1.
          format: int32
          type: integer
        selector:
          description: Label selector for pods. Existing ReplicaSets whose pods are selected by this will be the ones affected by this deployment.
          properties:
            matchExpressions:
              description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
              items:
                description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                properties:
                  key:
                    description: key is the label key that the selector applies to.
                    type: string
                  operator:
                    description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                    type: string
                  values:
                    description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                    items:
                      type: string
                    type: array
                required:
                - key
                - operator
                type: object
              type: array
            matchLabels:
              additionalProperties:
                type: string
              description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
              type: object
          type: object
        template:
          description: Template describes the pods that will be created.
          properties:
            metadata:
              nullable: true
              type: object
              x-kubernetes-preserve-unknown-fields: true
            spec:
              description: VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
              properties:
                affinity:
                  description: If affinity is specifies, obey all the affinity rules
                  properties:
                    nodeAffinity:
                      description: Describes node affinity scheduling rules for the pod.
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.
                          items:
                            description: An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).
                            properties:
                              preference:
                                description: A node selector term, associated with the corresponding weight.
                                properties:
                                  matchExpressions:
                                    description: A list of node selector requirements by node's labels.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchFields:
                                    description: A list of node selector requirements by node's fields.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                type: object
                              weight:
                                description: Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - preference
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.
                          properties:
                            nodeSelectorTerms:
                              description: Required. A list of node selector terms. The terms are ORed.
                              items:
                                description: A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.
                                properties:
                                  matchExpressions:
                                    description: A list of node selector requirements by node's labels.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchFields:
                                    description: A list of node selector requirements by node's fields.
                                    items:
                                      description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: The label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                          type: string
                                        values:
                                          description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                type: object
                              type: array
                          required:
                          - nodeSelectorTerms
                          type: object
                      type: object
                    podAffinity:
                      description: Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                          items:
                            description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                            properties:
                              podAffinityTerm:
                                description: Required. A pod affinity term, associated with the corresponding weight.
                                properties:
                                  labelSelector:
                                    description: A label query over a set of resources, in this case pods.
                                    properties:
                                      matchExpressions:
                                        description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                        items:
                                          description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                          properties:
                                            key:
                                              description: key is the label key that the selector applies to.
                                              type: string
                                            operator:
                                              description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                              type: string
                                            values:
                                              description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                              items:
                                                type: string
                                              type: array
                                          required:
                                          - key
                                          - operator
                                          type: object
                                        type: array
                                      matchLabels:
                                        additionalProperties:
                                          type: string
                                        description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                        type: object
                                    type: object
                                  namespaces:
                                    description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                    items:
                                      type: string
                                    type: array
                                  topologyKey:
                                    description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                    type: string
                                required:
                                - topologyKey
                                type: object
                              weight:
                                description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - podAffinityTerm
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                          items:
                            description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                            properties:
                              labelSelector:
                                description: A label query over a set of resources, in this case pods.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                    items:
                                      description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: key is the label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                    type: object
                                type: object
                              namespaces:
                                description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                items:
                                  type: string
                                type: array
                              topologyKey:
                                description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                type: string
                            required:
                            - topologyKey
                            type: object
                          type: array
                      type: object
                    podAntiAffinity:
                      description: Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).
                      properties:
                        preferredDuringSchedulingIgnoredDuringExecution:
                          description: The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                          items:
                            description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                            properties:
                              podAffinityTerm:
                                description: Required. A pod affinity term, associated with the corresponding weight.
                                properties:
                                  labelSelector:
                                    description: A label query over a set of resources, in this case pods.
                                    properties:
                                      matchExpressions:
                                        description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                        items:
                                          description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                          properties:
                                            key:
                                              description: key is the label key that the selector applies to.
                                              type: string
                                            operator:
                                              description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                              type: string
                                            values:
                                              description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                              items:
                                                type: string
                                              type: array
                                          required:
                                          - key
                                          - operator
                                          type: object
                                        type: array
                                      matchLabels:
                                        additionalProperties:
                                          type: string
                                        description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                        type: object
                                    type: object
                                  namespaces:
                                    description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                    items:
                                      type: string
                                    type: array
                                  topologyKey:
                                    description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                    type: string
                                required:
                                - topologyKey
                                type: object
                              weight:
                                description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                                format: int32
                                type: integer
                            required:
                            - podAffinityTerm
                            - weight
                            type: object
                          type: array
                        requiredDuringSchedulingIgnoredDuringExecution:
                          description: If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                          items:
                            description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                            properties:
                              labelSelector:
                                description: A label query over a set of resources, in this case pods.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                    items:
                                      description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                      properties:
                                        key:
                                          description: key is the label key that the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                    type: object
                                type: object
                              namespaces:
                                description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                items:
                                  type: string
                                type: array
                              topologyKey:
                                description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                type: string
                            required:
                            - topologyKey
                            type: object
                          type: array
                      type: object
                  type: object
                dnsConfig:
                  description: Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS configuration based on DNSPolicy.
                  properties:
                    nameservers:
                      description: A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed.
                      items:
                        type: string
                      type: array
                    options:
                      description: A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy.
                      items:
                        description: PodDNSConfigOption defines DNS resolver options of a pod.
                        properties:
                          name:
                            description: Required.
                            type: string
                          value:
                            type: string
                        type: object
                      type: array
                    searches:
                      description: A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed.
                      items:
                        type: string
                      type: array
                  type: object
                dnsPolicy:
                  description: Set DNS policy for the pod. Defaults to "ClusterFirst". Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'.
                  type: string
                domain:
                  description: Specification of the desired behavior of the VirtualMachineInstance on the host.
                  properties:
                    chassis:
                      description: Chassis specifies the chassis info passed to the domain.
                      properties:
                        asset:
                          type: string
                        manufacturer:
                          type: string
                        serial:
                          type: string
                        sku:
                          type: string
                        version:
                          type: string
                      type: object
                    clock:
                      description: Clock sets the clock and timers of the vmi.
                      properties:
                        timer:
                          description: Timer specifies whih timers are attached to the vmi.
                          properties:
                            hpet:
                              description: HPET (High Precision Event Timer) - multiple timers with periodic interrupts.
                              properties:
                                present:
                                  description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                  type: boolean
                                tickPolicy:
                                  description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup", "merge", "discard".
                                  type: string
                              type: object
                            hyperv:
                              description: Hyperv (Hypervclock) - lets guests read the host’s wall clock time (paravirtualized). For windows guests.
                              properties:
                                present:
                                  description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                  type: boolean
                              type: object
                            kvm:
                              description: "KVM \t(KVM clock) - lets guests read the host’s wall clock time (paravirtualized). For linux guests."
                              properties:
                                present:
                                  description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                  type: boolean
                              type: object
                            pit:
                              description: PIT (Programmable Interval Timer) - a timer with periodic interrupts.
                              properties:
                                present:
                                  description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                  type: boolean
                                tickPolicy:
                                  description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup", "discard".
                                  type: string
                              type: object
                            rtc:
                              description: RTC (Real Time Clock) - a continuously running timer with periodic interrupts.
                              properties:
                                present:
                                  description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                  type: boolean
                                tickPolicy:
                                  description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup".
                                  type: string
                                track:
                                  description: Track the guest or the wall clock.
                                  type: string
                              type: object
                          type: object
                        timezone:
                          description: Timezone sets the guest clock to the specified timezone. Zone name follows the TZ environment variable format (e.g. 'America/New_York').
                          type: string
                        utc:
                          description: UTC sets the guest clock to UTC on each boot. If an offset is specified, guest changes to the clock will be kept during reboots and are not reset.
                          properties:
                            offsetSeconds:
                              description: OffsetSeconds specifies an offset in seconds, relative to UTC. If set, guest changes to the clock will be kept during reboots and not reset.
                              type: integer
                          type: object
                      type: object
                    cpu:
                      description: CPU allow specified the detailed CPU topology inside the vmi.
                      properties:
                        cores:
                          description: Cores specifies the number of cores inside the vmi. Must be a value greater or equal 1.
                          format: int32
                          type: integer
                        dedicatedCpuPlacement:
                          description: DedicatedCPUPlacement requests the scheduler to place the VirtualMachineInstance on a node with enough dedicated pCPUs and pin the vCPUs to it.
                          type: boolean
                        features:
                          description: Features specifies the CPU features list inside the VMI.
                          items:
                            description: CPUFeature allows specifying a CPU feature.
                            properties:
                              name:
                                description: Name of the CPU feature
                                type: string
                              policy:
                                description: 'Policy is the CPU feature attribute which can have the following attributes: force    - The virtual CPU will claim the feature is supported regardless of it being supported by host CPU. require  - Guest creation will fail unless the feature is supported by the host CPU or the hypervisor is able to emulate it. optional - The feature will be supported by virtual CPU if and only if it is supported by host CPU. disable  - The feature will not be supported by virtual CPU. forbid   - Guest creation will fail if the feature is supported by host CPU. Defaults to require'
                                type: string
                            required:
                            - name
                            type: object
                          type: array
                        isolateEmulatorThread:
                          description: IsolateEmulatorThread requests one more dedicated pCPU to be allocated for the VMI to place the emulator thread on it.
                          type: boolean
                        model:
                          description: Model specifies the CPU model inside the VMI. List of available models https://github.com/libvirt/libvirt/tree/master/src/cpu_map. It is possible to specify special cases like "host-passthrough" to get the same CPU as the node and "host-model" to get CPU closest to the node one. Defaults to host-model.
                          type: string
                        sockets:
                          description: Sockets specifies the number of sockets inside the vmi. Must be a value greater or equal 1.
                          format: int32
                          type: integer
                        threads:
                          description: Threads specifies the number of threads inside the vmi. Must be a value greater or equal 1.
                          format: int32
                          type: integer
                      type: object
                    devices:
                      description: Devices allows adding disks, network interfaces, and others
                      properties:
                        autoattachGraphicsDevice:
                          description: Whether to attach the default graphics device or not. VNC will not be available if set to false. Defaults to true.
                          type: boolean
                        autoattachMemBalloon:
                          description: Whether to attach the Memory balloon device with default period. Period can be adjusted in virt-config. Defaults to true.
                          type: boolean
                        autoattachPodInterface:
                          description: Whether to attach a pod network interface. Defaults to true.
                          type: boolean
                        autoattachSerialConsole:
                          description: Whether to attach the default serial console or not. Serial console access will not be available if set to false. Defaults to true.
                          type: boolean
                        blockMultiQueue:
                          description: Whether or not to enable virtio multi-queue for block devices
                          type: boolean
                        disks:
                          description: Disks describes disks, cdroms, floppy and luns which are connected to the vmi.
                          items:
                            properties:
                              bootOrder:
                                description: BootOrder is an integer value > 0, used to determine ordering of boot devices. Lower values take precedence. Each disk or interface that has a boot order must have a unique value. Disks without a boot order are not tried if a disk with a boot order exists.
                                type: integer
                              cache:
                                description: Cache specifies which kvm disk cache mode should be used.
                                type: string
                              cdrom:
                                description: Attach a volume as a cdrom to the vmi.
                                properties:
                                  bus:
                                    description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                                    type: string
                                  readonly:
                                    description: ReadOnly. Defaults to true.
                                    type: boolean
                                  tray:
                                    description: Tray indicates if the tray of the device is open or closed. Allowed values are "open" and "closed". Defaults to closed.
                                    type: string
                                type: object
                              dedicatedIOThread:
                                description: dedicatedIOThread indicates this disk should have an exclusive IO Thread. Enabling this implies useIOThreads = true. Defaults to false.
                                type: boolean
                              disk:
                                description: Attach a volume as a disk to the vmi.
                                properties:
                                  bus:
                                    description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                                    type: string
                                  pciAddress:
                                    description: 'If specified, the virtual disk will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10'
                                    type: string
                                  readonly:
                                    description: ReadOnly. Defaults to false.
                                    type: boolean
                                type: object
                              floppy:
                                description: Attach a volume as a floppy to the vmi.
                                properties:
                                  readonly:
                                    description: ReadOnly. Defaults to false.
                                    type: boolean
                                  tray:
                                    description: Tray indicates if the tray of the device is open or closed. Allowed values are "open" and "closed". Defaults to closed.
                                    type: string
                                type: object
                              io:
                                description: 'IO specifies which QEMU disk IO mode should be used. Supported values are: native, default, threads.'
                                type: string
                              lun:
                                description: Attach a volume as a LUN to the vmi.
                                properties:
                                  bus:
                                    description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                                    type: string
                                  readonly:
                                    description: ReadOnly. Defaults to false.
                                    type: boolean
                                type: object
                              name:
                                description: Name is the device name
                                type: string
                              serial:
                                description: Serial provides the ability to specify a serial number for the disk device.
                                type: string
                              tag:
                                description: If specified, disk address and its tag will be provided to the guest via config drive metadata
                                type: string
                            required:
                            - name
                            type: object
                          type: array
                        filesystems:
                          description: Filesystems describes filesystem which is connected to the vmi.
                          items:
                            properties:
                              name:
                                description: Name is the device name
                                type: string
                              virtiofs:
                                description: Virtiofs is supported
                                type: object
                            required:
                            - name
                            - virtiofs
                            type: object
                          type: array
                        gpus:
                          description: Whether to attach a GPU device to the vmi.
                          items:
                            properties:
                              deviceName:
                                type: string
                              name:
                                description: Name of the GPU device as exposed by a device plugin
                                type: string
                            required:
                            - deviceName
                            - name
                            type: object
                          type: array
                        inputs:
                          description: Inputs describe input devices
                          items:
                            properties:
                              bus:
                                description: 'Bus indicates the bus of input device to emulate. Supported values: virtio, usb.'
                                type: string
                              name:
                                description: Name is the device name
                                type: string
                              type:
                                description: 'Type indicated the type of input device. Supported values: tablet.'
                                type: string
                            required:
                            - name
                            - type
                            type: object
                          type: array
                        interfaces:
                          description: Interfaces describe network interfaces which are added to the vmi.
                          items:
                            properties:
                              bootOrder:
                                description: BootOrder is an integer value > 0, used to determine ordering of boot devices. Lower values take precedence. Each interface or disk that has a boot order must have a unique value. Interfaces without a boot order are not tried.
                                type: integer
                              bridge:
                                type: object
                              dhcpOptions:
                                description: If specified the network interface will pass additional DHCP options to the VMI
                                properties:
                                  bootFileName:
                                    description: If specified will pass option 67 to interface's DHCP server
                                    type: string
                                  ntpServers:
                                    description: If specified will pass the configured NTP server to the VM via DHCP option 042.
                                    items:
                                      type: string
                                    type: array
                                  privateOptions:
                                    description: 'If specified will pass extra DHCP options for private use, range: 224-254'
                                    items:
                                      description: DHCPExtraOptions defines Extra DHCP options for a VM.
                                      properties:
                                        option:
                                          description: Option is an Integer value from 224-254 Required.
                                          type: integer
                                        value:
                                          description: Value is a String value for the Option provided Required.
                                          type: string
                                      required:
                                      - option
                                      - value
                                      type: object
                                    type: array
                                  tftpServerName:
                                    description: If specified will pass option 66 to interface's DHCP server
                                    type: string
                                type: object
                              macAddress:
                                description: 'Interface MAC address. For example: de:ad:00:00:be:af or DE-AD-00-00-BE-AF.'
                                type: string
                              masquerade:
                                type: object
                              model:
                                description: 'Interface model. One of: e1000, e1000e, ne2k_pci, pcnet, rtl8139, virtio. Defaults to virtio. TODO:(ihar) switch to enums once opengen-api supports them. See: https://github.com/kubernetes/kube-openapi/issues/51'
                                type: string
                              name:
                                description: Logical name of the interface as well as a reference to the associated networks. Must match the Name of a Network.
                                type: string
                              pciAddress:
                                description: 'If specified, the virtual network interface will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10'
                                type: string
                              ports:
                                description: List of ports to be forwarded to the virtual machine.
                                items:
                                  description: Port repesents a port to expose from the virtual machine. Default protocol TCP. The port field is mandatory
                                  properties:
                                    name:
                                      description: If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.
                                      type: string
                                    port:
                                      description: Number of port to expose for the virtual machine. This must be a valid port number, 0 < x < 65536.
                                      format: int32
                                      type: integer
                                    protocol:
                                      description: Protocol for port. Must be UDP or TCP. Defaults to "TCP".
                                      type: string
                                  required:
                                  - port
                                  type: object
                                type: array
                              slirp:
                                type: object
                              sriov:
                                type: object
                              tag:
                                description: If specified, the virtual network interface address and its tag will be provided to the guest via config drive
                                type: string
                            required:
                            - name
                            type: object
                          type: array
                        networkInterfaceMultiqueue:
                          description: If specified, virtual network interfaces configured with a virtio bus will also enable the vhost multiqueue feature for network devices. The number of queues created depends on additional factors of the VirtualMachineInstance, like the number of guest CPUs.
                          type: boolean
                        rng:
                          description: Whether to have random number generator from host
                          type: object
                        watchdog:
                          description: Watchdog describes a watchdog device which can be added to the vmi.
                          properties:
                            i6300esb:
                              description: i6300esb watchdog device.
                              properties:
                                action:
                                  description: The action to take. Valid values are poweroff, reset, shutdown. Defaults to reset.
                                  type: string
                              type: object
                            name:
                              description: Name of the watchdog.
                              type: string
                          required:
                          - name
                          type: object
                      type: object
                    features:
                      description: Features like acpi, apic, hyperv, smm.
                      properties:
                        acpi:
                          description: ACPI enables/disables ACPI inside the guest. Defaults to enabled.
                          properties:
                            enabled:
                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                              type: boolean
                          type: object
                        apic:
                          description: Defaults to the machine type setting.
                          properties:
                            enabled:
                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                              type: boolean
                            endOfInterrupt:
                              description: EndOfInterrupt enables the end of interrupt notification in the guest. Defaults to false.
                              type: boolean
                          type: object
                        hyperv:
                          description: Defaults to the machine type setting.
                          properties:
                            evmcs:
                              description: EVMCS Speeds up L2 vmexits, but disables other virtualization features. Requires vapic. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            frequencies:
                              description: Frequencies improves the TSC clock source handling for Hyper-V on KVM. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            ipi:
                              description: IPI improves performances in overcommited environments. Requires vpindex. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            reenlightenment:
                              description: Reenlightenment enables the notifications on TSC frequency changes. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            relaxed:
                              description: Relaxed instructs the guest OS to disable watchdog timeouts. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            reset:
                              description: Reset enables Hyperv reboot/reset for the vmi. Requires synic. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            runtime:
                              description: Runtime improves the time accounting to improve scheduling in the guest. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            spinlocks:
                              description: Spinlocks allows to configure the spinlock retry attempts.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                                spinlocks:
                                  description: Retries indicates the number of retries. Must be a value greater or equal 4096. Defaults to 4096.
                                  format: int32
                                  type: integer
                              type: object
                            synic:
                              description: SyNIC enables the Synthetic Interrupt Controller. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            synictimer:
                              description: SyNICTimer enables Synthetic Interrupt Controller Timers, reducing CPU load. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            tlbflush:
                              description: TLBFlush improves performances in overcommited environments. Requires vpindex. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            vapic:
                              description: VAPIC improves the paravirtualized handling of interrupts. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                            vendorid:
                              description: VendorID allows setting the hypervisor vendor id. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                                vendorid:
                                  description: VendorID sets the hypervisor vendor id, visible to the vmi. String up to twelve characters.
                                  type: string
                              type: object
                            vpindex:
                              description: VPIndex enables the Virtual Processor Index to help windows identifying virtual processors. Defaults to the machine type setting.
                              properties:
                                enabled:
                                  description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                  type: boolean
                              type: object
                          type: object
                        kvm:
                          description: Configure how KVM presence is exposed to the guest.
                          properties:
                            hidden:
                              description: Hide the KVM hypervisor from standard MSR based discovery. Defaults to false
                              type: boolean
                          type: object
                        smm:
                          description: SMM enables/disables System Management Mode. TSEG not yet implemented.
                          properties:
                            enabled:
                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                              type: boolean
                          type: object
                      type: object
                    firmware:
                      description: Firmware.
                      properties:
                        bootloader:
                          description: Settings to control the bootloader that is used.
                          properties:
                            bios:
                              description: If set (default), BIOS will be used.
                              properties:
                                useSerial:
                                  description: If set, the BIOS output will be transmitted over serial
                                  type: boolean
                              type: object
                            efi:
                              description: If set, EFI will be used instead of BIOS.
                              properties:
                                secureBoot:
                                  description: If set, SecureBoot will be enabled and the OVMF roms will be swapped for SecureBoot-enabled ones. Requires SMM to be enabled. Defaults to true
                                  type: boolean
                              type: object
                          type: object
                        serial:
                          description: The system-serial-number in SMBIOS
                          type: string
                        uuid:
                          description: UUID reported by the vmi bios. Defaults to a random generated uid.
                          type: string
                      type: object
                    ioThreadsPolicy:
                      description: 'Controls whether or not disks will share IOThreads. Omitting IOThreadsPolicy disables use of IOThreads. One of: shared, auto'
                      type: string
                    machine:
                      description: Machine type.
                      properties:
                        type:
                          description: QEMU machine type is the actual chipset of the VirtualMachineInstance.
                          type: string
                      required:
                      - type
                      type: object
                    memory:
                      description: Memory allow specifying the VMI memory features.
                      properties:
                        guest:
                          anyOf:
                          - type: integer
                          - type: string
                          description: Guest allows to specifying the amount of memory which is visible inside the Guest OS. The Guest must lie between Requests and Limits from the resources section. Defaults to the requested memory in the resources section if not specified.
                          pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                          x-kubernetes-int-or-string: true
                        hugepages:
                          description: Hugepages allow to use hugepages for the VirtualMachineInstance instead of regular memory.
                          properties:
                            pageSize:
                              description: PageSize specifies the hugepage size, for x86_64 architecture valid values are 1Gi and 2Mi.
                              type: string
                          type: object
                      type: object
                    resources:
                      description: Resources describes the Compute Resources required by this vmi.
                      properties:
                        limits:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Limits describes the maximum amount of compute resources allowed. Valid resource keys are "memory" and "cpu".
                          type: object
                        overcommitGuestOverhead:
                          description: Don't ask the scheduler to take the guest-management overhead into account. Instead put the overhead only into the container's memory limit. This can lead to crashes if all memory is in use on a node. Defaults to false.
                          type: boolean
                        requests:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Requests is a description of the initial vmi resources. Valid resource keys are "memory" and "cpu".
                          type: object
                      type: object
                  required:
                  - devices
                  type: object
                evictionStrategy:
                  description: EvictionStrategy can be set to "LiveMigrate" if the VirtualMachineInstance should be migrated instead of shut-off in case of a node drain.
                  type: string
                hostname:
                  description: Specifies the hostname of the vmi If not specified, the hostname will be set to the name of the vmi, if dhcp or cloud-init is configured properly.
                  type: string
                livenessProbe:
                  description: 'Periodic probe of VirtualMachineInstance liveness. VirtualmachineInstances will be stopped if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                  properties:
                    failureThreshold:
                      description: Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.
                      format: int32
                      type: integer
                    httpGet:
                      description: HTTPGet specifies the http request to perform.
                      properties:
                        host:
                          description: Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.
                          type: string
                        httpHeaders:
                          description: Custom headers to set in the request. HTTP allows repeated headers.
                          items:
                            description: HTTPHeader describes a custom header to be used in HTTP probes
                            properties:
                              name:
                                description: The header field name
                                type: string
                              value:
                                description: The header field value
                                type: string
                            required:
                            - name
                            - value
                            type: object
                          type: array
                        path:
                          description: Path to access on the HTTP server.
                          type: string
                        port:
                          anyOf:
                          - type: integer
                          - type: string
                          description: Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                          x-kubernetes-int-or-string: true
                        scheme:
                          description: Scheme to use for connecting to the host. Defaults to HTTP.
                          type: string
                      required:
                      - port
                      type: object
                    initialDelaySeconds:
                      description: 'Number of seconds after the VirtualMachineInstance has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                      format: int32
                      type: integer
                    periodSeconds:
                      description: How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.
                      format: int32
                      type: integer
                    successThreshold:
                      description: Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness. Minimum value is 1.
                      format: int32
                      type: integer
                    tcpSocket:
                      description: 'TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported TODO: implement a realistic TCP lifecycle hook'
                      properties:
                        host:
                          description: 'Optional: Host name to connect to, defaults to the pod IP.'
                          type: string
                        port:
                          anyOf:
                          - type: integer
                          - type: string
                          description: Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                          x-kubernetes-int-or-string: true
                      required:
                      - port
                      type: object
                    timeoutSeconds:
                      description: 'Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                      format: int32
                      type: integer
                  type: object
                networks:
                  description: List of networks that can be attached to a vm's virtual interface.
                  items:
                    description: Network represents a network type and a resource that should be connected to the vm.
                    properties:
                      multus:
                        description: Represents the multus cni network.
                        properties:
                          default:
                            description: Select the default network and add it to the multus-cni.io/default-network annotation.
                            type: boolean
                          networkName:
                            description: 'References to a NetworkAttachmentDefinition CRD object. Format: <networkName>, <namespace>/<networkName>. If namespace is not specified, VMI namespace is assumed.'
                            type: string
                        required:
                        - networkName
                        type: object
                      name:
                        description: 'Network name. Must be a DNS_LABEL and unique within the vm. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names'
                        type: string
                      pod:
                        description: Represents the stock pod network interface.
                        properties:
                          vmNetworkCIDR:
                            description: CIDR for vm network. Default 10.0.2.0/24 if not specified.
                            type: string
                        type: object
                    required:
                    - name
                    type: object
                  type: array
                nodeSelector:
                  additionalProperties:
                    type: string
                  description: 'NodeSelector is a selector which must be true for the vmi to fit on a node. Selector which must match a node''s labels for the vmi to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/'
                  type: object
                priorityClassName:
                  description: If specified, indicates the pod's priority. If not specified, the pod priority will be default or zero if there is no default.
                  type: string
                readinessProbe:
                  description: 'Periodic probe of VirtualMachineInstance service readiness. VirtualmachineInstances will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                  properties:
                    failureThreshold:
                      description: Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.
                      format: int32
                      type: integer
                    httpGet:
                      description: HTTPGet specifies the http request to perform.
                      properties:
                        host:
                          description: Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.
                          type: string
                        httpHeaders:
                          description: Custom headers to set in the request. HTTP allows repeated headers.
                          items:
                            description: HTTPHeader describes a custom header to be used in HTTP probes
                            properties:
                              name:
                                description: The header field name
                                type: string
                              value:
                                description: The header field value
                                type: string
                            required:
                            - name
                            - value
                            type: object
                          type: array
                        path:
                          description: Path to access on the HTTP server.
                          type: string
                        port:
                          anyOf:
                          - type: integer
                          - type: string
                          description: Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                          x-kubernetes-int-or-string: true
                        scheme:
                          description: Scheme to use for connecting to the host. Defaults to HTTP.
                          type: string
                      required:
                      - port
                      type: object
                    initialDelaySeconds:
                      description: 'Number of seconds after the VirtualMachineInstance has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                      format: int32
                      type: integer
                    periodSeconds:
                      description: How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.
                      format: int32
                      type: integer
                    successThreshold:
                      description: Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness. Minimum value is 1.
                      format: int32
                      type: integer
                    tcpSocket:
                      description: 'TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported TODO: implement a realistic TCP lifecycle hook'
                      properties:
                        host:
                          description: 'Optional: Host name to connect to, defaults to the pod IP.'
                          type: string
                        port:
                          anyOf:
                          - type: integer
                          - type: string
                          description: Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                          x-kubernetes-int-or-string: true
                      required:
                      - port
                      type: object
                    timeoutSeconds:
                      description: 'Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                      format: int32
                      type: integer
                  type: object
                schedulerName:
                  description: If specified, the VMI will be dispatched by specified scheduler. If not specified, the VMI will be dispatched by default scheduler.
                  type: string
                subdomain:
                  description: If specified, the fully qualified vmi hostname will be "<hostname>.<subdomain>.<pod namespace>.svc.<cluster domain>". If not specified, the vmi will not have a domainname at all. The DNS entry will resolve to the vmi, no matter if the vmi itself can pick up a hostname.
                  type: string
                terminationGracePeriodSeconds:
                  description: Grace period observed after signalling a VirtualMachineInstance to stop after which the VirtualMachineInstance is force terminated.
                  format: int64
                  type: integer
                tolerations:
                  description: If toleration is specified, obey all the toleration rules.
                  items:
                    description: The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.
                    properties:
                      effect:
                        description: Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.
                        type: string
                      key:
                        description: Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.
                        type: string
                      operator:
                        description: Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.
                        type: string
                      tolerationSeconds:
                        description: TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.
                        format: int64
                        type: integer
                      value:
                        description: Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.
                        type: string
                    type: object
                  type: array
                volumes:
                  description: List of volumes that can be mounted by disks belonging to the vmi.
                  items:
                    description: Volume represents a named volume in a vmi.
                    properties:
                      cloudInitConfigDrive:
                        description: 'CloudInitConfigDrive represents a cloud-init Config Drive user-data source. The Config Drive data will be added as a disk to the vmi. A proper cloud-init installation is required inside the guest. More info: https://cloudinit.readthedocs.io/en/latest/topics/datasources/configdrive.html'
                        properties:
                          networkData:
                            description: NetworkData contains config drive inline cloud-init networkdata.
                            type: string
                          networkDataBase64:
                            description: NetworkDataBase64 contains config drive cloud-init networkdata as a base64 encoded string.
                            type: string
                          networkDataSecretRef:
                            description: NetworkDataSecretRef references a k8s secret that contains config drive networkdata.
                            properties:
                              name:
                                description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                type: string
                            type: object
                          secretRef:
                            description: UserDataSecretRef references a k8s secret that contains config drive userdata.
                            properties:
                              name:
                                description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                type: string
                            type: object
                          userData:
                            description: UserData contains config drive inline cloud-init userdata.
                            type: string
                          userDataBase64:
                            description: UserDataBase64 contains config drive cloud-init userdata as a base64 encoded string.
                            type: string
                        type: object
                      cloudInitNoCloud:
                        description: 'CloudInitNoCloud represents a cloud-init NoCloud user-data source. The NoCloud data will be added as a disk to the vmi. A proper cloud-init installation is required inside the guest. More info: http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html'
                        properties:
                          networkData:
                            description: NetworkData contains NoCloud inline cloud-init networkdata.
                            type: string
                          networkDataBase64:
                            description: NetworkDataBase64 contains NoCloud cloud-init networkdata as a base64 encoded string.
                            type: string
                          networkDataSecretRef:
                            description: NetworkDataSecretRef references a k8s secret that contains NoCloud networkdata.
                            properties:
                              name:
                                description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                type: string
                            type: object
                          secretRef:
                            description: UserDataSecretRef references a k8s secret that contains NoCloud userdata.
                            properties:
                              name:
                                description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                type: string
                            type: object
                          userData:
                            description: UserData contains NoCloud inline cloud-init userdata.
                            type: string
                          userDataBase64:
                            description: UserDataBase64 contains NoCloud cloud-init userdata as a base64 encoded string.
                            type: string
                        type: object
                      configMap:
                        description: 'ConfigMapSource represents a reference to a ConfigMap in the same namespace. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/'
                        properties:
                          name:
                            description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                            type: string
                          optional:
                            description: Specify whether the ConfigMap or it's keys must be defined
                            type: boolean
                          volumeLabel:
                            description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                            type: string
                        type: object
                      containerDisk:
                        description: 'ContainerDisk references a docker image, embedding a qcow or raw disk. More info: https://kubevirt.gitbooks.io/user-guide/registry-disk.html'
                        properties:
                          image:
                            description: Image is the name of the image with the embedded disk.
                            type: string
                          imagePullPolicy:
                            description: 'Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images'
                            type: string
                          imagePullSecret:
                            description: ImagePullSecret is the name of the Docker registry secret required to pull the image. The secret must already exist.
                            type: string
                          path:
                            description: Path defines the path to disk file in the container
                            type: string
                        required:
                        - image
                        type: object
                      dataVolume:
                        description: DataVolume represents the dynamic creation a PVC for this volume as well as the process of populating that PVC with a disk image.
                        properties:
                          name:
                            description: Name represents the name of the DataVolume in the same namespace
                            type: string
                        required:
                        - name
                        type: object
                      downwardAPI:
                        description: DownwardAPI represents downward API about the pod that should populate this volume
                        properties:
                          fields:
                            description: Fields is a list of downward API volume file
                            items:
                              description: DownwardAPIVolumeFile represents information to create the file containing the pod field
                              properties:
                                fieldRef:
                                  description: 'Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.'
                                  properties:
                                    apiVersion:
                                      description: Version of the schema the FieldPath is written in terms of, defaults to "v1".
                                      type: string
                                    fieldPath:
                                      description: Path of the field to select in the specified API version.
                                      type: string
                                  required:
                                  - fieldPath
                                  type: object
                                mode:
                                  description: 'Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.'
                                  format: int32
                                  type: integer
                                path:
                                  description: 'Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the ''..'' path. Must be utf-8 encoded. The first item of the relative path must not start with ''..'''
                                  type: string
                                resourceFieldRef:
                                  description: 'Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.'
                                  properties:
                                    containerName:
                                      description: 'Container name: required for volumes, optional for env vars'
                                      type: string
                                    divisor:
                                      anyOf:
                                      - type: integer
                                      - type: string
                                      description: Specifies the output format of the exposed resources, defaults to "1"
                                      pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                                      x-kubernetes-int-or-string: true
                                    resource:
                                      description: 'Required: resource to select'
                                      type: string
                                  required:
                                  - resource
                                  type: object
                              required:
                              - path
                              type: object
                            type: array
                          volumeLabel:
                            description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                            type: string
                        type: object
                      emptyDisk:
                        description: 'EmptyDisk represents a temporary disk which shares the vmis lifecycle. More info: https://kubevirt.gitbooks.io/user-guide/disks-and-volumes.html'
                        properties:
                          capacity:
                            anyOf:
                            - type: integer
                            - type: string
                            description: Capacity of the sparse disk.
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                        required:
                        - capacity
                        type: object
                      ephemeral:
                        description: Ephemeral is a special volume source that "wraps" specified source and provides copy-on-write image on top of it.
                        properties:
                          persistentVolumeClaim:
                            description: 'PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. Directly attached to the vmi via qemu. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                            properties:
                              claimName:
                                description: 'ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                                type: string
                              readOnly:
                                description: Will force the ReadOnly setting in VolumeMounts. Default false.
                                type: boolean
                            required:
                            - claimName
                            type: object
                        type: object
                      hostDisk:
                        description: HostDisk represents a disk created on the cluster level
                        properties:
                          capacity:
                            anyOf:
                            - type: integer
                            - type: string
                            description: Capacity of the sparse disk
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          path:
                            description: The path to HostDisk image located on the cluster
                            type: string
                          shared:
                            description: Shared indicate whether the path is shared between nodes
                            type: boolean
                          type:
                            description: Contains information if disk.img exists or should be created allowed options are 'Disk' and 'DiskOrCreate'
                            type: string
                        required:
                        - path
                        - type
                        type: object
                      name:
                        description: 'Volume''s name. Must be a DNS_LABEL and unique within the vmi. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names'
                        type: string
                      persistentVolumeClaim:
                        description: 'PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. Directly attached to the vmi via qemu. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                        properties:
                          claimName:
                            description: 'ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                            type: string
                          readOnly:
                            description: Will force the ReadOnly setting in VolumeMounts. Default false.
                            type: boolean
                        required:
                        - claimName
                        type: object
                      secret:
                        description: 'SecretVolumeSource represents a reference to a secret data in the same namespace. More info: https://kubernetes.io/docs/concepts/configuration/secret/'
                        properties:
                          optional:
                            description: Specify whether the Secret or it's keys must be defined
                            type: boolean
                          secretName:
                            description: 'Name of the secret in the pod''s namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret'
                            type: string
                          volumeLabel:
                            description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                            type: string
                        type: object
                      serviceAccount:
                        description: 'ServiceAccountVolumeSource represents a reference to a service account. There can only be one volume of this type! More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/'
                        properties:
                          serviceAccountName:
                            description: 'Name of the service account in the pod''s namespace to use. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/'
                            type: string
                        type: object
                    required:
                    - name
                    type: object
                  type: array
              required:
              - domain
              type: object
          type: object
      required:
      - selector
      - template
      type: object
    status:
      description: Status is the high level overview of how the VirtualMachineInstance is doing. It contains information available to controllers and users.
      nullable: true
      properties:
        conditions:
          items:
            properties:
              lastProbeTime:
                format: date-time
                nullable: true
                type: string
              lastTransitionTime:
                format: date-time
                nullable: true
                type: string
              message:
                type: string
              reason:
                type: string
              status:
                type: string
              type:
                type: string
            required:
            - status
            - type
            type: object
          type: array
        labelSelector:
          description: Canonical form of the label selector for HPA which consumes it through the scale subresource.
          type: string
        readyReplicas:
          description: The number of ready replicas for this replica set.
          format: int32
          type: integer
        replicas:
          description: Total number of non-terminated pods targeted by this deployment (their labels match the selector).
          format: int32
          type: integer
      type: object
  required:
  - spec
  type: object
`,
	"virtualmachinerestore": `openAPIV3Schema:
  description: VirtualMachineRestore defines the operation of restoring a VM
  properties:
    apiVersion:
      description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
      type: string
    kind:
      description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
      type: string
    metadata:
      type: object
    spec:
      description: VirtualMachineRestoreSpec is the spec for a VirtualMachineRestoreresource
      properties:
        target:
          description: initially only VirtualMachine type supported
          properties:
            apiGroup:
              description: APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.
              type: string
            kind:
              description: Kind is the type of resource being referenced
              type: string
            name:
              description: Name is the name of resource being referenced
              type: string
          required:
          - kind
          - name
          type: object
        virtualMachineSnapshotName:
          type: string
      required:
      - target
      - virtualMachineSnapshotName
      type: object
    status:
      description: VirtualMachineRestoreStatus is the spec for a VirtualMachineRestoreresource
      properties:
        complete:
          type: boolean
        conditions:
          items:
            description: Condition defines conditions
            properties:
              lastProbeTime:
                format: date-time
                nullable: true
                type: string
              lastTransitionTime:
                format: date-time
                nullable: true
                type: string
              message:
                type: string
              reason:
                type: string
              status:
                type: string
              type:
                description: ConditionType is the const type for Conditions
                type: string
            required:
            - status
            - type
            type: object
          type: array
        deletedDataVolumes:
          items:
            type: string
          type: array
        restoreTime:
          format: date-time
          type: string
        restores:
          items:
            description: VolumeRestore contains the data neeed to restore a PVC
            properties:
              dataVolumeName:
                type: string
              persistentVolumeClaim:
                type: string
              volumeName:
                type: string
              volumeSnapshotName:
                type: string
            required:
            - persistentVolumeClaim
            - volumeName
            - volumeSnapshotName
            type: object
          type: array
      type: object
  required:
  - spec
  type: object
`,
	"virtualmachinesnapshot": `openAPIV3Schema:
  description: VirtualMachineSnapshot defines the operation of snapshotting a VM
  properties:
    apiVersion:
      description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
      type: string
    kind:
      description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
      type: string
    metadata:
      type: object
    spec:
      description: VirtualMachineSnapshotSpec is the spec for a VirtualMachineSnapshot resource
      properties:
        deletionPolicy:
          description: DeletionPolicy defines that to do with VirtualMachineSnapshot when VirtualMachineSnapshot is deleted
          type: string
        source:
          description: TypedLocalObjectReference contains enough information to let you locate the typed referenced object inside the same namespace.
          properties:
            apiGroup:
              description: APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.
              type: string
            kind:
              description: Kind is the type of resource being referenced
              type: string
            name:
              description: Name is the name of resource being referenced
              type: string
          required:
          - kind
          - name
          type: object
      required:
      - source
      type: object
    status:
      description: VirtualMachineSnapshotStatus is the status for a VirtualMachineSnapshot resource
      properties:
        conditions:
          items:
            description: Condition defines conditions
            properties:
              lastProbeTime:
                format: date-time
                nullable: true
                type: string
              lastTransitionTime:
                format: date-time
                nullable: true
                type: string
              message:
                type: string
              reason:
                type: string
              status:
                type: string
              type:
                description: ConditionType is the const type for Conditions
                type: string
            required:
            - status
            - type
            type: object
          type: array
        creationTime:
          format: date-time
          nullable: true
          type: string
        error:
          description: Error is the last error encountered during the snapshot/restore
          properties:
            message:
              type: string
            time:
              format: date-time
              type: string
          type: object
        readyToUse:
          type: boolean
        sourceUID:
          description: UID is a type that holds unique ID values, including UUIDs.  Because we don't ONLY use UUIDs, this is an alias to string.  Being a type captures intent and helps make sure that UIDs and names do not get conflated.
          type: string
        virtualMachineSnapshotContentName:
          type: string
      type: object
  required:
  - spec
  type: object
`,
	"virtualmachinesnapshotcontent": `openAPIV3Schema:
  description: VirtualMachineSnapshotContent contains the snapshot data
  properties:
    apiVersion:
      description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
      type: string
    kind:
      description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
      type: string
    metadata:
      type: object
    spec:
      description: VirtualMachineSnapshotContentSpec is the spec for a VirtualMachineSnapshotContent resource
      properties:
        source:
          description: SourceSpec contains the appropriate spec for the resource being snapshotted
          properties:
            virtualMachine:
              description: VirtualMachine handles the VirtualMachines that are not running or are in a stopped state The VirtualMachine contains the template to create the VirtualMachineInstance. It also mirrors the running state of the created VirtualMachineInstance in its status.
              properties:
                apiVersion:
                  description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
                  type: string
                kind:
                  description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
                  type: string
                metadata:
                  type: object
                spec:
                  description: Spec contains the specification of VirtualMachineInstance created
                  properties:
                    dataVolumeTemplates:
                      description: dataVolumeTemplates is a list of dataVolumes that the VirtualMachineInstance template can reference. DataVolumes in this list are dynamically created for the VirtualMachine and are tied to the VirtualMachine's life-cycle.
                      items:
                        nullable: true
                        properties:
                          apiVersion:
                            description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
                            type: string
                          kind:
                            description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
                            type: string
                          metadata:
                            nullable: true
                            type: object
                            x-kubernetes-preserve-unknown-fields: true
                          spec:
                            description: DataVolumeSpec contains the DataVolume specification.
                            properties:
                              contentType:
                                description: 'DataVolumeContentType options: "kubevirt", "archive"'
                                enum:
                                - kubevirt
                                - archive
                                type: string
                              pvc:
                                description: PVC is the PVC specification
                                properties:
                                  accessModes:
                                    description: 'AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1'
                                    items:
                                      type: string
                                    type: array
                                  dataSource:
                                    description: This field requires the VolumeSnapshotDataSource alpha feature gate to be enabled and currently VolumeSnapshot is the only supported data source. If the provisioner can support VolumeSnapshot data source, it will create a new volume and data will be restored to the volume at the same time. If the provisioner does not support VolumeSnapshot data source, volume will not be created and the failure will be reported as an event. In the future, we plan to support more data source types and the behavior of the provisioner may change.
                                    properties:
                                      apiGroup:
                                        description: APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.
                                        type: string
                                      kind:
                                        description: Kind is the type of resource being referenced
                                        type: string
                                      name:
                                        description: Name is the name of resource being referenced
                                        type: string
                                    required:
                                    - kind
                                    - name
                                    type: object
                                  resources:
                                    description: 'Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources'
                                    properties:
                                      limits:
                                        additionalProperties:
                                          anyOf:
                                          - type: integer
                                          - type: string
                                          pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                                          x-kubernetes-int-or-string: true
                                        description: 'Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/'
                                        type: object
                                      requests:
                                        additionalProperties:
                                          anyOf:
                                          - type: integer
                                          - type: string
                                          pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                                          x-kubernetes-int-or-string: true
                                        description: 'Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/'
                                        type: object
                                    type: object
                                  selector:
                                    description: A label query over volumes to consider for binding.
                                    properties:
                                      matchExpressions:
                                        description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                        items:
                                          description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                          properties:
                                            key:
                                              description: key is the label key that the selector applies to.
                                              type: string
                                            operator:
                                              description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                              type: string
                                            values:
                                              description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                              items:
                                                type: string
                                              type: array
                                          required:
                                          - key
                                          - operator
                                          type: object
                                        type: array
                                      matchLabels:
                                        additionalProperties:
                                          type: string
                                        description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                        type: object
                                    type: object
                                  storageClassName:
                                    description: 'Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1'
                                    type: string
                                  volumeMode:
                                    description: volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec. This is a beta feature.
                                    type: string
                                  volumeName:
                                    description: VolumeName is the binding reference to the PersistentVolume backing this claim.
                                    type: string
                                type: object
                              source:
                                description: Source is the src of the data for the requested DataVolume
                                properties:
                                  blank:
                                    description: DataVolumeBlankImage provides the parameters to create a new raw blank image for the PVC
                                    type: object
                                  http:
                                    description: DataVolumeSourceHTTP can be either an http or https endpoint, with an optional basic auth user name and password, and an optional configmap containing additional CAs
                                    properties:
                                      certConfigMap:
                                        description: CertConfigMap is a configmap reference, containing a Certificate Authority(CA) public key, and a base64 encoded pem certificate
                                        type: string
                                      secretRef:
                                        description: SecretRef A Secret reference, the secret should contain accessKeyId (user name) base64 encoded, and secretKey (password) also base64 encoded
                                        type: string
                                      url:
                                        description: URL is the URL of the http(s) endpoint
                                        type: string
                                    required:
                                    - url
                                    type: object
                                  imageio:
                                    description: DataVolumeSourceImageIO provides the parameters to create a Data Volume from an imageio source
                                    properties:
                                      certConfigMap:
                                        description: CertConfigMap provides a reference to the CA cert
                                        type: string
                                      diskId:
                                        description: DiskID provides id of a disk to be imported
                                        type: string
                                      secretRef:
                                        description: SecretRef provides the secret reference needed to access the ovirt-engine
                                        type: string
                                      url:
                                        description: URL is the URL of the ovirt-engine
                                        type: string
                                    required:
                                    - diskId
                                    - url
                                    type: object
                                  pvc:
                                    description: DataVolumeSourcePVC provides the parameters to create a Data Volume from an existing PVC
                                    properties:
                                      name:
                                        description: The name of the source PVC
                                        type: string
                                      namespace:
                                        description: The namespace of the source PVC
                                        type: string
                                    required:
                                    - name
                                    - namespace
                                    type: object
                                  registry:
                                    description: DataVolumeSourceRegistry provides the parameters to create a Data Volume from an registry source
                                    properties:
                                      certConfigMap:
                                        description: CertConfigMap provides a reference to the Registry certs
                                        type: string
                                      secretRef:
                                        description: SecretRef provides the secret reference needed to access the Registry source
                                        type: string
                                      url:
                                        description: URL is the url of the Docker registry source
                                        type: string
                                    required:
                                    - url
                                    type: object
                                  s3:
                                    description: DataVolumeSourceS3 provides the parameters to create a Data Volume from an S3 source
                                    properties:
                                      secretRef:
                                        description: SecretRef provides the secret reference needed to access the S3 source
                                        type: string
                                      url:
                                        description: URL is the url of the S3 source
                                        type: string
                                    required:
                                    - url
                                    type: object
                                  upload:
                                    description: DataVolumeSourceUpload provides the parameters to create a Data Volume by uploading the source
                                    type: object
                                  vddk:
                                    description: DataVolumeSourceVDDK provides the parameters to create a Data Volume from a Vmware source
                                    properties:
                                      backingFile:
                                        description: BackingFile is the path to the virtual hard disk to migrate from vCenter/ESXi
                                        type: string
                                      secretRef:
                                        description: SecretRef provides a reference to a secret containing the username and password needed to access the vCenter or ESXi host
                                        type: string
                                      thumbprint:
                                        description: Thumbprint is the certificate thumbprint of the vCenter or ESXi host
                                        type: string
                                      url:
                                        description: URL is the URL of the vCenter or ESXi host with the VM to migrate
                                        type: string
                                      uuid:
                                        description: UUID is the UUID of the virtual machine that the backing file is attached to in vCenter/ESXi
                                        type: string
                                    type: object
                                type: object
                            required:
                            - pvc
                            - source
                            type: object
                          status:
                            description: DataVolumeTemplateDummyStatus is here simply for backwards compatibility with a previous API.
                            nullable: true
                            type: object
                        required:
                        - spec
                        type: object
                      type: array
                    runStrategy:
                      description: Running state indicates the requested running state of the VirtualMachineInstance mutually exclusive with Running
                      type: string
                    running:
                      description: Running controls whether the associatied VirtualMachineInstance is created or not Mutually exclusive with RunStrategy
                      type: boolean
                    template:
                      description: Template is the direct specification of VirtualMachineInstance
                      properties:
                        metadata:
                          nullable: true
                          type: object
                          x-kubernetes-preserve-unknown-fields: true
                        spec:
                          description: VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
                          properties:
                            affinity:
                              description: If affinity is specifies, obey all the affinity rules
                              properties:
                                nodeAffinity:
                                  description: Describes node affinity scheduling rules for the pod.
                                  properties:
                                    preferredDuringSchedulingIgnoredDuringExecution:
                                      description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.
                                      items:
                                        description: An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).
                                        properties:
                                          preference:
                                            description: A node selector term, associated with the corresponding weight.
                                            properties:
                                              matchExpressions:
                                                description: A list of node selector requirements by node's labels.
                                                items:
                                                  description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                                  properties:
                                                    key:
                                                      description: The label key that the selector applies to.
                                                      type: string
                                                    operator:
                                                      description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                                      type: string
                                                    values:
                                                      description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                                      items:
                                                        type: string
                                                      type: array
                                                  required:
                                                  - key
                                                  - operator
                                                  type: object
                                                type: array
                                              matchFields:
                                                description: A list of node selector requirements by node's fields.
                                                items:
                                                  description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                                  properties:
                                                    key:
                                                      description: The label key that the selector applies to.
                                                      type: string
                                                    operator:
                                                      description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                                      type: string
                                                    values:
                                                      description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                                      items:
                                                        type: string
                                                      type: array
                                                  required:
                                                  - key
                                                  - operator
                                                  type: object
                                                type: array
                                            type: object
                                          weight:
                                            description: Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.
                                            format: int32
                                            type: integer
                                        required:
                                        - preference
                                        - weight
                                        type: object
                                      type: array
                                    requiredDuringSchedulingIgnoredDuringExecution:
                                      description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.
                                      properties:
                                        nodeSelectorTerms:
                                          description: Required. A list of node selector terms. The terms are ORed.
                                          items:
                                            description: A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.
                                            properties:
                                              matchExpressions:
                                                description: A list of node selector requirements by node's labels.
                                                items:
                                                  description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                                  properties:
                                                    key:
                                                      description: The label key that the selector applies to.
                                                      type: string
                                                    operator:
                                                      description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                                      type: string
                                                    values:
                                                      description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                                      items:
                                                        type: string
                                                      type: array
                                                  required:
                                                  - key
                                                  - operator
                                                  type: object
                                                type: array
                                              matchFields:
                                                description: A list of node selector requirements by node's fields.
                                                items:
                                                  description: A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                                  properties:
                                                    key:
                                                      description: The label key that the selector applies to.
                                                      type: string
                                                    operator:
                                                      description: Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
                                                      type: string
                                                    values:
                                                      description: An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.
                                                      items:
                                                        type: string
                                                      type: array
                                                  required:
                                                  - key
                                                  - operator
                                                  type: object
                                                type: array
                                            type: object
                                          type: array
                                      required:
                                      - nodeSelectorTerms
                                      type: object
                                  type: object
                                podAffinity:
                                  description: Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).
                                  properties:
                                    preferredDuringSchedulingIgnoredDuringExecution:
                                      description: The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                                      items:
                                        description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                                        properties:
                                          podAffinityTerm:
                                            description: Required. A pod affinity term, associated with the corresponding weight.
                                            properties:
                                              labelSelector:
                                                description: A label query over a set of resources, in this case pods.
                                                properties:
                                                  matchExpressions:
                                                    description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                                    items:
                                                      description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                                      properties:
                                                        key:
                                                          description: key is the label key that the selector applies to.
                                                          type: string
                                                        operator:
                                                          description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                                          type: string
                                                        values:
                                                          description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                                          items:
                                                            type: string
                                                          type: array
                                                      required:
                                                      - key
                                                      - operator
                                                      type: object
                                                    type: array
                                                  matchLabels:
                                                    additionalProperties:
                                                      type: string
                                                    description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                                    type: object
                                                type: object
                                              namespaces:
                                                description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                                items:
                                                  type: string
                                                type: array
                                              topologyKey:
                                                description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                                type: string
                                            required:
                                            - topologyKey
                                            type: object
                                          weight:
                                            description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                                            format: int32
                                            type: integer
                                        required:
                                        - podAffinityTerm
                                        - weight
                                        type: object
                                      type: array
                                    requiredDuringSchedulingIgnoredDuringExecution:
                                      description: If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                                      items:
                                        description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                                        properties:
                                          labelSelector:
                                            description: A label query over a set of resources, in this case pods.
                                            properties:
                                              matchExpressions:
                                                description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                                items:
                                                  description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                                  properties:
                                                    key:
                                                      description: key is the label key that the selector applies to.
                                                      type: string
                                                    operator:
                                                      description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                                      type: string
                                                    values:
                                                      description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                                      items:
                                                        type: string
                                                      type: array
                                                  required:
                                                  - key
                                                  - operator
                                                  type: object
                                                type: array
                                              matchLabels:
                                                additionalProperties:
                                                  type: string
                                                description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                                type: object
                                            type: object
                                          namespaces:
                                            description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                            items:
                                              type: string
                                            type: array
                                          topologyKey:
                                            description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                            type: string
                                        required:
                                        - topologyKey
                                        type: object
                                      type: array
                                  type: object
                                podAntiAffinity:
                                  description: Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).
                                  properties:
                                    preferredDuringSchedulingIgnoredDuringExecution:
                                      description: The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.
                                      items:
                                        description: The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
                                        properties:
                                          podAffinityTerm:
                                            description: Required. A pod affinity term, associated with the corresponding weight.
                                            properties:
                                              labelSelector:
                                                description: A label query over a set of resources, in this case pods.
                                                properties:
                                                  matchExpressions:
                                                    description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                                    items:
                                                      description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                                      properties:
                                                        key:
                                                          description: key is the label key that the selector applies to.
                                                          type: string
                                                        operator:
                                                          description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                                          type: string
                                                        values:
                                                          description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                                          items:
                                                            type: string
                                                          type: array
                                                      required:
                                                      - key
                                                      - operator
                                                      type: object
                                                    type: array
                                                  matchLabels:
                                                    additionalProperties:
                                                      type: string
                                                    description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                                    type: object
                                                type: object
                                              namespaces:
                                                description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                                items:
                                                  type: string
                                                type: array
                                              topologyKey:
                                                description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                                type: string
                                            required:
                                            - topologyKey
                                            type: object
                                          weight:
                                            description: weight associated with matching the corresponding podAffinityTerm, in the range 1-100.
                                            format: int32
                                            type: integer
                                        required:
                                        - podAffinityTerm
                                        - weight
                                        type: object
                                      type: array
                                    requiredDuringSchedulingIgnoredDuringExecution:
                                      description: If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.
                                      items:
                                        description: Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running
                                        properties:
                                          labelSelector:
                                            description: A label query over a set of resources, in this case pods.
                                            properties:
                                              matchExpressions:
                                                description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                                                items:
                                                  description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                                                  properties:
                                                    key:
                                                      description: key is the label key that the selector applies to.
                                                      type: string
                                                    operator:
                                                      description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                                      type: string
                                                    values:
                                                      description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                                      items:
                                                        type: string
                                                      type: array
                                                  required:
                                                  - key
                                                  - operator
                                                  type: object
                                                type: array
                                              matchLabels:
                                                additionalProperties:
                                                  type: string
                                                description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                                                type: object
                                            type: object
                                          namespaces:
                                            description: namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"
                                            items:
                                              type: string
                                            type: array
                                          topologyKey:
                                            description: This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.
                                            type: string
                                        required:
                                        - topologyKey
                                        type: object
                                      type: array
                                  type: object
                              type: object
                            dnsConfig:
                              description: Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS configuration based on DNSPolicy.
                              properties:
                                nameservers:
                                  description: A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed.
                                  items:
                                    type: string
                                  type: array
                                options:
                                  description: A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy.
                                  items:
                                    description: PodDNSConfigOption defines DNS resolver options of a pod.
                                    properties:
                                      name:
                                        description: Required.
                                        type: string
                                      value:
                                        type: string
                                    type: object
                                  type: array
                                searches:
                                  description: A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed.
                                  items:
                                    type: string
                                  type: array
                              type: object
                            dnsPolicy:
                              description: Set DNS policy for the pod. Defaults to "ClusterFirst". Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'.
                              type: string
                            domain:
                              description: Specification of the desired behavior of the VirtualMachineInstance on the host.
                              properties:
                                chassis:
                                  description: Chassis specifies the chassis info passed to the domain.
                                  properties:
                                    asset:
                                      type: string
                                    manufacturer:
                                      type: string
                                    serial:
                                      type: string
                                    sku:
                                      type: string
                                    version:
                                      type: string
                                  type: object
                                clock:
                                  description: Clock sets the clock and timers of the vmi.
                                  properties:
                                    timer:
                                      description: Timer specifies whih timers are attached to the vmi.
                                      properties:
                                        hpet:
                                          description: HPET (High Precision Event Timer) - multiple timers with periodic interrupts.
                                          properties:
                                            present:
                                              description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                              type: boolean
                                            tickPolicy:
                                              description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup", "merge", "discard".
                                              type: string
                                          type: object
                                        hyperv:
                                          description: Hyperv (Hypervclock) - lets guests read the host’s wall clock time (paravirtualized). For windows guests.
                                          properties:
                                            present:
                                              description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                              type: boolean
                                          type: object
                                        kvm:
                                          description: "KVM \t(KVM clock) - lets guests read the host’s wall clock time (paravirtualized). For linux guests."
                                          properties:
                                            present:
                                              description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                              type: boolean
                                          type: object
                                        pit:
                                          description: PIT (Programmable Interval Timer) - a timer with periodic interrupts.
                                          properties:
                                            present:
                                              description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                              type: boolean
                                            tickPolicy:
                                              description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup", "discard".
                                              type: string
                                          type: object
                                        rtc:
                                          description: RTC (Real Time Clock) - a continuously running timer with periodic interrupts.
                                          properties:
                                            present:
                                              description: Enabled set to false makes sure that the machine type or a preset can't add the timer. Defaults to true.
                                              type: boolean
                                            tickPolicy:
                                              description: TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest. One of "delay", "catchup".
                                              type: string
                                            track:
                                              description: Track the guest or the wall clock.
                                              type: string
                                          type: object
                                      type: object
                                    timezone:
                                      description: Timezone sets the guest clock to the specified timezone. Zone name follows the TZ environment variable format (e.g. 'America/New_York').
                                      type: string
                                    utc:
                                      description: UTC sets the guest clock to UTC on each boot. If an offset is specified, guest changes to the clock will be kept during reboots and are not reset.
                                      properties:
                                        offsetSeconds:
                                          description: OffsetSeconds specifies an offset in seconds, relative to UTC. If set, guest changes to the clock will be kept during reboots and not reset.
                                          type: integer
                                      type: object
                                  type: object
                                cpu:
                                  description: CPU allow specified the detailed CPU topology inside the vmi.
                                  properties:
                                    cores:
                                      description: Cores specifies the number of cores inside the vmi. Must be a value greater or equal 1.
                                      format: int32
                                      type: integer
                                    dedicatedCpuPlacement:
                                      description: DedicatedCPUPlacement requests the scheduler to place the VirtualMachineInstance on a node with enough dedicated pCPUs and pin the vCPUs to it.
                                      type: boolean
                                    features:
                                      description: Features specifies the CPU features list inside the VMI.
                                      items:
                                        description: CPUFeature allows specifying a CPU feature.
                                        properties:
                                          name:
                                            description: Name of the CPU feature
                                            type: string
                                          policy:
                                            description: 'Policy is the CPU feature attribute which can have the following attributes: force    - The virtual CPU will claim the feature is supported regardless of it being supported by host CPU. require  - Guest creation will fail unless the feature is supported by the host CPU or the hypervisor is able to emulate it. optional - The feature will be supported by virtual CPU if and only if it is supported by host CPU. disable  - The feature will not be supported by virtual CPU. forbid   - Guest creation will fail if the feature is supported by host CPU. Defaults to require'
                                            type: string
                                        required:
                                        - name
                                        type: object
                                      type: array
                                    isolateEmulatorThread:
                                      description: IsolateEmulatorThread requests one more dedicated pCPU to be allocated for the VMI to place the emulator thread on it.
                                      type: boolean
                                    model:
                                      description: Model specifies the CPU model inside the VMI. List of available models https://github.com/libvirt/libvirt/tree/master/src/cpu_map. It is possible to specify special cases like "host-passthrough" to get the same CPU as the node and "host-model" to get CPU closest to the node one. Defaults to host-model.
                                      type: string
                                    sockets:
                                      description: Sockets specifies the number of sockets inside the vmi. Must be a value greater or equal 1.
                                      format: int32
                                      type: integer
                                    threads:
                                      description: Threads specifies the number of threads inside the vmi. Must be a value greater or equal 1.
                                      format: int32
                                      type: integer
                                  type: object
                                devices:
                                  description: Devices allows adding disks, network interfaces, and others
                                  properties:
                                    autoattachGraphicsDevice:
                                      description: Whether to attach the default graphics device or not. VNC will not be available if set to false. Defaults to true.
                                      type: boolean
                                    autoattachMemBalloon:
                                      description: Whether to attach the Memory balloon device with default period. Period can be adjusted in virt-config. Defaults to true.
                                      type: boolean
                                    autoattachPodInterface:
                                      description: Whether to attach a pod network interface. Defaults to true.
                                      type: boolean
                                    autoattachSerialConsole:
                                      description: Whether to attach the default serial console or not. Serial console access will not be available if set to false. Defaults to true.
                                      type: boolean
                                    blockMultiQueue:
                                      description: Whether or not to enable virtio multi-queue for block devices
                                      type: boolean
                                    disks:
                                      description: Disks describes disks, cdroms, floppy and luns which are connected to the vmi.
                                      items:
                                        properties:
                                          bootOrder:
                                            description: BootOrder is an integer value > 0, used to determine ordering of boot devices. Lower values take precedence. Each disk or interface that has a boot order must have a unique value. Disks without a boot order are not tried if a disk with a boot order exists.
                                            type: integer
                                          cache:
                                            description: Cache specifies which kvm disk cache mode should be used.
                                            type: string
                                          cdrom:
                                            description: Attach a volume as a cdrom to the vmi.
                                            properties:
                                              bus:
                                                description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                                                type: string
                                              readonly:
                                                description: ReadOnly. Defaults to true.
                                                type: boolean
                                              tray:
                                                description: Tray indicates if the tray of the device is open or closed. Allowed values are "open" and "closed". Defaults to closed.
                                                type: string
                                            type: object
                                          dedicatedIOThread:
                                            description: dedicatedIOThread indicates this disk should have an exclusive IO Thread. Enabling this implies useIOThreads = true. Defaults to false.
                                            type: boolean
                                          disk:
                                            description: Attach a volume as a disk to the vmi.
                                            properties:
                                              bus:
                                                description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                                                type: string
                                              pciAddress:
                                                description: 'If specified, the virtual disk will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10'
                                                type: string
                                              readonly:
                                                description: ReadOnly. Defaults to false.
                                                type: boolean
                                            type: object
                                          floppy:
                                            description: Attach a volume as a floppy to the vmi.
                                            properties:
                                              readonly:
                                                description: ReadOnly. Defaults to false.
                                                type: boolean
                                              tray:
                                                description: Tray indicates if the tray of the device is open or closed. Allowed values are "open" and "closed". Defaults to closed.
                                                type: string
                                            type: object
                                          io:
                                            description: 'IO specifies which QEMU disk IO mode should be used. Supported values are: native, default, threads.'
                                            type: string
                                          lun:
                                            description: Attach a volume as a LUN to the vmi.
                                            properties:
                                              bus:
                                                description: 'Bus indicates the type of disk device to emulate. supported values: virtio, sata, scsi.'
                                                type: string
                                              readonly:
                                                description: ReadOnly. Defaults to false.
                                                type: boolean
                                            type: object
                                          name:
                                            description: Name is the device name
                                            type: string
                                          serial:
                                            description: Serial provides the ability to specify a serial number for the disk device.
                                            type: string
                                          tag:
                                            description: If specified, disk address and its tag will be provided to the guest via config drive metadata
                                            type: string
                                        required:
                                        - name
                                        type: object
                                      type: array
                                    filesystems:
                                      description: Filesystems describes filesystem which is connected to the vmi.
                                      items:
                                        properties:
                                          name:
                                            description: Name is the device name
                                            type: string
                                          virtiofs:
                                            description: Virtiofs is supported
                                            type: object
                                        required:
                                        - name
                                        - virtiofs
                                        type: object
                                      type: array
                                    gpus:
                                      description: Whether to attach a GPU device to the vmi.
                                      items:
                                        properties:
                                          deviceName:
                                            type: string
                                          name:
                                            description: Name of the GPU device as exposed by a device plugin
                                            type: string
                                        required:
                                        - deviceName
                                        - name
                                        type: object
                                      type: array
                                    inputs:
                                      description: Inputs describe input devices
                                      items:
                                        properties:
                                          bus:
                                            description: 'Bus indicates the bus of input device to emulate. Supported values: virtio, usb.'
                                            type: string
                                          name:
                                            description: Name is the device name
                                            type: string
                                          type:
                                            description: 'Type indicated the type of input device. Supported values: tablet.'
                                            type: string
                                        required:
                                        - name
                                        - type
                                        type: object
                                      type: array
                                    interfaces:
                                      description: Interfaces describe network interfaces which are added to the vmi.
                                      items:
                                        properties:
                                          bootOrder:
                                            description: BootOrder is an integer value > 0, used to determine ordering of boot devices. Lower values take precedence. Each interface or disk that has a boot order must have a unique value. Interfaces without a boot order are not tried.
                                            type: integer
                                          bridge:
                                            type: object
                                          dhcpOptions:
                                            description: If specified the network interface will pass additional DHCP options to the VMI
                                            properties:
                                              bootFileName:
                                                description: If specified will pass option 67 to interface's DHCP server
                                                type: string
                                              ntpServers:
                                                description: If specified will pass the configured NTP server to the VM via DHCP option 042.
                                                items:
                                                  type: string
                                                type: array
                                              privateOptions:
                                                description: 'If specified will pass extra DHCP options for private use, range: 224-254'
                                                items:
                                                  description: DHCPExtraOptions defines Extra DHCP options for a VM.
                                                  properties:
                                                    option:
                                                      description: Option is an Integer value from 224-254 Required.
                                                      type: integer
                                                    value:
                                                      description: Value is a String value for the Option provided Required.
                                                      type: string
                                                  required:
                                                  - option
                                                  - value
                                                  type: object
                                                type: array
                                              tftpServerName:
                                                description: If specified will pass option 66 to interface's DHCP server
                                                type: string
                                            type: object
                                          macAddress:
                                            description: 'Interface MAC address. For example: de:ad:00:00:be:af or DE-AD-00-00-BE-AF.'
                                            type: string
                                          masquerade:
                                            type: object
                                          model:
                                            description: 'Interface model. One of: e1000, e1000e, ne2k_pci, pcnet, rtl8139, virtio. Defaults to virtio. TODO:(ihar) switch to enums once opengen-api supports them. See: https://github.com/kubernetes/kube-openapi/issues/51'
                                            type: string
                                          name:
                                            description: Logical name of the interface as well as a reference to the associated networks. Must match the Name of a Network.
                                            type: string
                                          pciAddress:
                                            description: 'If specified, the virtual network interface will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10'
                                            type: string
                                          ports:
                                            description: List of ports to be forwarded to the virtual machine.
                                            items:
                                              description: Port repesents a port to expose from the virtual machine. Default protocol TCP. The port field is mandatory
                                              properties:
                                                name:
                                                  description: If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.
                                                  type: string
                                                port:
                                                  description: Number of port to expose for the virtual machine. This must be a valid port number, 0 < x < 65536.
                                                  format: int32
                                                  type: integer
                                                protocol:
                                                  description: Protocol for port. Must be UDP or TCP. Defaults to "TCP".
                                                  type: string
                                              required:
                                              - port
                                              type: object
                                            type: array
                                          slirp:
                                            type: object
                                          sriov:
                                            type: object
                                          tag:
                                            description: If specified, the virtual network interface address and its tag will be provided to the guest via config drive
                                            type: string
                                        required:
                                        - name
                                        type: object
                                      type: array
                                    networkInterfaceMultiqueue:
                                      description: If specified, virtual network interfaces configured with a virtio bus will also enable the vhost multiqueue feature for network devices. The number of queues created depends on additional factors of the VirtualMachineInstance, like the number of guest CPUs.
                                      type: boolean
                                    rng:
                                      description: Whether to have random number generator from host
                                      type: object
                                    watchdog:
                                      description: Watchdog describes a watchdog device which can be added to the vmi.
                                      properties:
                                        i6300esb:
                                          description: i6300esb watchdog device.
                                          properties:
                                            action:
                                              description: The action to take. Valid values are poweroff, reset, shutdown. Defaults to reset.
                                              type: string
                                          type: object
                                        name:
                                          description: Name of the watchdog.
                                          type: string
                                      required:
                                      - name
                                      type: object
                                  type: object
                                features:
                                  description: Features like acpi, apic, hyperv, smm.
                                  properties:
                                    acpi:
                                      description: ACPI enables/disables ACPI inside the guest. Defaults to enabled.
                                      properties:
                                        enabled:
                                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                          type: boolean
                                      type: object
                                    apic:
                                      description: Defaults to the machine type setting.
                                      properties:
                                        enabled:
                                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                          type: boolean
                                        endOfInterrupt:
                                          description: EndOfInterrupt enables the end of interrupt notification in the guest. Defaults to false.
                                          type: boolean
                                      type: object
                                    hyperv:
                                      description: Defaults to the machine type setting.
                                      properties:
                                        evmcs:
                                          description: EVMCS Speeds up L2 vmexits, but disables other virtualization features. Requires vapic. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                        frequencies:
                                          description: Frequencies improves the TSC clock source handling for Hyper-V on KVM. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                        ipi:
                                          description: IPI improves performances in overcommited environments. Requires vpindex. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                        reenlightenment:
                                          description: Reenlightenment enables the notifications on TSC frequency changes. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                        relaxed:
                                          description: Relaxed instructs the guest OS to disable watchdog timeouts. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                        reset:
                                          description: Reset enables Hyperv reboot/reset for the vmi. Requires synic. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                        runtime:
                                          description: Runtime improves the time accounting to improve scheduling in the guest. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                        spinlocks:
                                          description: Spinlocks allows to configure the spinlock retry attempts.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                            spinlocks:
                                              description: Retries indicates the number of retries. Must be a value greater or equal 4096. Defaults to 4096.
                                              format: int32
                                              type: integer
                                          type: object
                                        synic:
                                          description: SyNIC enables the Synthetic Interrupt Controller. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                        synictimer:
                                          description: SyNICTimer enables Synthetic Interrupt Controller Timers, reducing CPU load. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                        tlbflush:
                                          description: TLBFlush improves performances in overcommited environments. Requires vpindex. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                        vapic:
                                          description: VAPIC improves the paravirtualized handling of interrupts. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                        vendorid:
                                          description: VendorID allows setting the hypervisor vendor id. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                            vendorid:
                                              description: VendorID sets the hypervisor vendor id, visible to the vmi. String up to twelve characters.
                                              type: string
                                          type: object
                                        vpindex:
                                          description: VPIndex enables the Virtual Processor Index to help windows identifying virtual processors. Defaults to the machine type setting.
                                          properties:
                                            enabled:
                                              description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                              type: boolean
                                          type: object
                                      type: object
                                    kvm:
                                      description: Configure how KVM presence is exposed to the guest.
                                      properties:
                                        hidden:
                                          description: Hide the KVM hypervisor from standard MSR based discovery. Defaults to false
                                          type: boolean
                                      type: object
                                    smm:
                                      description: SMM enables/disables System Management Mode. TSEG not yet implemented.
                                      properties:
                                        enabled:
                                          description: Enabled determines if the feature should be enabled or disabled on the guest. Defaults to true.
                                          type: boolean
                                      type: object
                                  type: object
                                firmware:
                                  description: Firmware.
                                  properties:
                                    bootloader:
                                      description: Settings to control the bootloader that is used.
                                      properties:
                                        bios:
                                          description: If set (default), BIOS will be used.
                                          properties:
                                            useSerial:
                                              description: If set, the BIOS output will be transmitted over serial
                                              type: boolean
                                          type: object
                                        efi:
                                          description: If set, EFI will be used instead of BIOS.
                                          properties:
                                            secureBoot:
                                              description: If set, SecureBoot will be enabled and the OVMF roms will be swapped for SecureBoot-enabled ones. Requires SMM to be enabled. Defaults to true
                                              type: boolean
                                          type: object
                                      type: object
                                    serial:
                                      description: The system-serial-number in SMBIOS
                                      type: string
                                    uuid:
                                      description: UUID reported by the vmi bios. Defaults to a random generated uid.
                                      type: string
                                  type: object
                                ioThreadsPolicy:
                                  description: 'Controls whether or not disks will share IOThreads. Omitting IOThreadsPolicy disables use of IOThreads. One of: shared, auto'
                                  type: string
                                machine:
                                  description: Machine type.
                                  properties:
                                    type:
                                      description: QEMU machine type is the actual chipset of the VirtualMachineInstance.
                                      type: string
                                  required:
                                  - type
                                  type: object
                                memory:
                                  description: Memory allow specifying the VMI memory features.
                                  properties:
                                    guest:
                                      anyOf:
                                      - type: integer
                                      - type: string
                                      description: Guest allows to specifying the amount of memory which is visible inside the Guest OS. The Guest must lie between Requests and Limits from the resources section. Defaults to the requested memory in the resources section if not specified.
                                      pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                                      x-kubernetes-int-or-string: true
                                    hugepages:
                                      description: Hugepages allow to use hugepages for the VirtualMachineInstance instead of regular memory.
                                      properties:
                                        pageSize:
                                          description: PageSize specifies the hugepage size, for x86_64 architecture valid values are 1Gi and 2Mi.
                                          type: string
                                      type: object
                                  type: object
                                resources:
                                  description: Resources describes the Compute Resources required by this vmi.
                                  properties:
                                    limits:
                                      additionalProperties:
                                        anyOf:
                                        - type: integer
                                        - type: string
                                        pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                                        x-kubernetes-int-or-string: true
                                      description: Limits describes the maximum amount of compute resources allowed. Valid resource keys are "memory" and "cpu".
                                      type: object
                                    overcommitGuestOverhead:
                                      description: Don't ask the scheduler to take the guest-management overhead into account. Instead put the overhead only into the container's memory limit. This can lead to crashes if all memory is in use on a node. Defaults to false.
                                      type: boolean
                                    requests:
                                      additionalProperties:
                                        anyOf:
                                        - type: integer
                                        - type: string
                                        pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                                        x-kubernetes-int-or-string: true
                                      description: Requests is a description of the initial vmi resources. Valid resource keys are "memory" and "cpu".
                                      type: object
                                  type: object
                              required:
                              - devices
                              type: object
                            evictionStrategy:
                              description: EvictionStrategy can be set to "LiveMigrate" if the VirtualMachineInstance should be migrated instead of shut-off in case of a node drain.
                              type: string
                            hostname:
                              description: Specifies the hostname of the vmi If not specified, the hostname will be set to the name of the vmi, if dhcp or cloud-init is configured properly.
                              type: string
                            livenessProbe:
                              description: 'Periodic probe of VirtualMachineInstance liveness. VirtualmachineInstances will be stopped if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                              properties:
                                failureThreshold:
                                  description: Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.
                                  format: int32
                                  type: integer
                                httpGet:
                                  description: HTTPGet specifies the http request to perform.
                                  properties:
                                    host:
                                      description: Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.
                                      type: string
                                    httpHeaders:
                                      description: Custom headers to set in the request. HTTP allows repeated headers.
                                      items:
                                        description: HTTPHeader describes a custom header to be used in HTTP probes
                                        properties:
                                          name:
                                            description: The header field name
                                            type: string
                                          value:
                                            description: The header field value
                                            type: string
                                        required:
                                        - name
                                        - value
                                        type: object
                                      type: array
                                    path:
                                      description: Path to access on the HTTP server.
                                      type: string
                                    port:
                                      anyOf:
                                      - type: integer
                                      - type: string
                                      description: Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                                      x-kubernetes-int-or-string: true
                                    scheme:
                                      description: Scheme to use for connecting to the host. Defaults to HTTP.
                                      type: string
                                  required:
                                  - port
                                  type: object
                                initialDelaySeconds:
                                  description: 'Number of seconds after the VirtualMachineInstance has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                                  format: int32
                                  type: integer
                                periodSeconds:
                                  description: How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.
                                  format: int32
                                  type: integer
                                successThreshold:
                                  description: Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness. Minimum value is 1.
                                  format: int32
                                  type: integer
                                tcpSocket:
                                  description: 'TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported TODO: implement a realistic TCP lifecycle hook'
                                  properties:
                                    host:
                                      description: 'Optional: Host name to connect to, defaults to the pod IP.'
                                      type: string
                                    port:
                                      anyOf:
                                      - type: integer
                                      - type: string
                                      description: Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                                      x-kubernetes-int-or-string: true
                                  required:
                                  - port
                                  type: object
                                timeoutSeconds:
                                  description: 'Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                                  format: int32
                                  type: integer
                              type: object
                            networks:
                              description: List of networks that can be attached to a vm's virtual interface.
                              items:
                                description: Network represents a network type and a resource that should be connected to the vm.
                                properties:
                                  multus:
                                    description: Represents the multus cni network.
                                    properties:
                                      default:
                                        description: Select the default network and add it to the multus-cni.io/default-network annotation.
                                        type: boolean
                                      networkName:
                                        description: 'References to a NetworkAttachmentDefinition CRD object. Format: <networkName>, <namespace>/<networkName>. If namespace is not specified, VMI namespace is assumed.'
                                        type: string
                                    required:
                                    - networkName
                                    type: object
                                  name:
                                    description: 'Network name. Must be a DNS_LABEL and unique within the vm. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names'
                                    type: string
                                  pod:
                                    description: Represents the stock pod network interface.
                                    properties:
                                      vmNetworkCIDR:
                                        description: CIDR for vm network. Default 10.0.2.0/24 if not specified.
                                        type: string
                                    type: object
                                required:
                                - name
                                type: object
                              type: array
                            nodeSelector:
                              additionalProperties:
                                type: string
                              description: 'NodeSelector is a selector which must be true for the vmi to fit on a node. Selector which must match a node''s labels for the vmi to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/'
                              type: object
                            priorityClassName:
                              description: If specified, indicates the pod's priority. If not specified, the pod priority will be default or zero if there is no default.
                              type: string
                            readinessProbe:
                              description: 'Periodic probe of VirtualMachineInstance service readiness. VirtualmachineInstances will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                              properties:
                                failureThreshold:
                                  description: Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.
                                  format: int32
                                  type: integer
                                httpGet:
                                  description: HTTPGet specifies the http request to perform.
                                  properties:
                                    host:
                                      description: Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.
                                      type: string
                                    httpHeaders:
                                      description: Custom headers to set in the request. HTTP allows repeated headers.
                                      items:
                                        description: HTTPHeader describes a custom header to be used in HTTP probes
                                        properties:
                                          name:
                                            description: The header field name
                                            type: string
                                          value:
                                            description: The header field value
                                            type: string
                                        required:
                                        - name
                                        - value
                                        type: object
                                      type: array
                                    path:
                                      description: Path to access on the HTTP server.
                                      type: string
                                    port:
                                      anyOf:
                                      - type: integer
                                      - type: string
                                      description: Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                                      x-kubernetes-int-or-string: true
                                    scheme:
                                      description: Scheme to use for connecting to the host. Defaults to HTTP.
                                      type: string
                                  required:
                                  - port
                                  type: object
                                initialDelaySeconds:
                                  description: 'Number of seconds after the VirtualMachineInstance has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                                  format: int32
                                  type: integer
                                periodSeconds:
                                  description: How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.
                                  format: int32
                                  type: integer
                                successThreshold:
                                  description: Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness. Minimum value is 1.
                                  format: int32
                                  type: integer
                                tcpSocket:
                                  description: 'TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported TODO: implement a realistic TCP lifecycle hook'
                                  properties:
                                    host:
                                      description: 'Optional: Host name to connect to, defaults to the pod IP.'
                                      type: string
                                    port:
                                      anyOf:
                                      - type: integer
                                      - type: string
                                      description: Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
                                      x-kubernetes-int-or-string: true
                                  required:
                                  - port
                                  type: object
                                timeoutSeconds:
                                  description: 'Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes'
                                  format: int32
                                  type: integer
                              type: object
                            schedulerName:
                              description: If specified, the VMI will be dispatched by specified scheduler. If not specified, the VMI will be dispatched by default scheduler.
                              type: string
                            subdomain:
                              description: If specified, the fully qualified vmi hostname will be "<hostname>.<subdomain>.<pod namespace>.svc.<cluster domain>". If not specified, the vmi will not have a domainname at all. The DNS entry will resolve to the vmi, no matter if the vmi itself can pick up a hostname.
                              type: string
                            terminationGracePeriodSeconds:
                              description: Grace period observed after signalling a VirtualMachineInstance to stop after which the VirtualMachineInstance is force terminated.
                              format: int64
                              type: integer
                            tolerations:
                              description: If toleration is specified, obey all the toleration rules.
                              items:
                                description: The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.
                                properties:
                                  effect:
                                    description: Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.
                                    type: string
                                  key:
                                    description: Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.
                                    type: string
                                  operator:
                                    description: Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.
                                    type: string
                                  tolerationSeconds:
                                    description: TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.
                                    format: int64
                                    type: integer
                                  value:
                                    description: Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.
                                    type: string
                                type: object
                              type: array
                            volumes:
                              description: List of volumes that can be mounted by disks belonging to the vmi.
                              items:
                                description: Volume represents a named volume in a vmi.
                                properties:
                                  cloudInitConfigDrive:
                                    description: 'CloudInitConfigDrive represents a cloud-init Config Drive user-data source. The Config Drive data will be added as a disk to the vmi. A proper cloud-init installation is required inside the guest. More info: https://cloudinit.readthedocs.io/en/latest/topics/datasources/configdrive.html'
                                    properties:
                                      networkData:
                                        description: NetworkData contains config drive inline cloud-init networkdata.
                                        type: string
                                      networkDataBase64:
                                        description: NetworkDataBase64 contains config drive cloud-init networkdata as a base64 encoded string.
                                        type: string
                                      networkDataSecretRef:
                                        description: NetworkDataSecretRef references a k8s secret that contains config drive networkdata.
                                        properties:
                                          name:
                                            description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                            type: string
                                        type: object
                                      secretRef:
                                        description: UserDataSecretRef references a k8s secret that contains config drive userdata.
                                        properties:
                                          name:
                                            description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                            type: string
                                        type: object
                                      userData:
                                        description: UserData contains config drive inline cloud-init userdata.
                                        type: string
                                      userDataBase64:
                                        description: UserDataBase64 contains config drive cloud-init userdata as a base64 encoded string.
                                        type: string
                                    type: object
                                  cloudInitNoCloud:
                                    description: 'CloudInitNoCloud represents a cloud-init NoCloud user-data source. The NoCloud data will be added as a disk to the vmi. A proper cloud-init installation is required inside the guest. More info: http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html'
                                    properties:
                                      networkData:
                                        description: NetworkData contains NoCloud inline cloud-init networkdata.
                                        type: string
                                      networkDataBase64:
                                        description: NetworkDataBase64 contains NoCloud cloud-init networkdata as a base64 encoded string.
                                        type: string
                                      networkDataSecretRef:
                                        description: NetworkDataSecretRef references a k8s secret that contains NoCloud networkdata.
                                        properties:
                                          name:
                                            description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                            type: string
                                        type: object
                                      secretRef:
                                        description: UserDataSecretRef references a k8s secret that contains NoCloud userdata.
                                        properties:
                                          name:
                                            description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                            type: string
                                        type: object
                                      userData:
                                        description: UserData contains NoCloud inline cloud-init userdata.
                                        type: string
                                      userDataBase64:
                                        description: UserDataBase64 contains NoCloud cloud-init userdata as a base64 encoded string.
                                        type: string
                                    type: object
                                  configMap:
                                    description: 'ConfigMapSource represents a reference to a ConfigMap in the same namespace. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/'
                                    properties:
                                      name:
                                        description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?'
                                        type: string
                                      optional:
                                        description: Specify whether the ConfigMap or it's keys must be defined
                                        type: boolean
                                      volumeLabel:
                                        description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                                        type: string
                                    type: object
                                  containerDisk:
                                    description: 'ContainerDisk references a docker image, embedding a qcow or raw disk. More info: https://kubevirt.gitbooks.io/user-guide/registry-disk.html'
                                    properties:
                                      image:
                                        description: Image is the name of the image with the embedded disk.
                                        type: string
                                      imagePullPolicy:
                                        description: 'Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images'
                                        type: string
                                      imagePullSecret:
                                        description: ImagePullSecret is the name of the Docker registry secret required to pull the image. The secret must already exist.
                                        type: string
                                      path:
                                        description: Path defines the path to disk file in the container
                                        type: string
                                    required:
                                    - image
                                    type: object
                                  dataVolume:
                                    description: DataVolume represents the dynamic creation a PVC for this volume as well as the process of populating that PVC with a disk image.
                                    properties:
                                      name:
                                        description: Name represents the name of the DataVolume in the same namespace
                                        type: string
                                    required:
                                    - name
                                    type: object
                                  downwardAPI:
                                    description: DownwardAPI represents downward API about the pod that should populate this volume
                                    properties:
                                      fields:
                                        description: Fields is a list of downward API volume file
                                        items:
                                          description: DownwardAPIVolumeFile represents information to create the file containing the pod field
                                          properties:
                                            fieldRef:
                                              description: 'Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.'
                                              properties:
                                                apiVersion:
                                                  description: Version of the schema the FieldPath is written in terms of, defaults to "v1".
                                                  type: string
                                                fieldPath:
                                                  description: Path of the field to select in the specified API version.
                                                  type: string
                                              required:
                                              - fieldPath
                                              type: object
                                            mode:
                                              description: 'Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.'
                                              format: int32
                                              type: integer
                                            path:
                                              description: 'Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the ''..'' path. Must be utf-8 encoded. The first item of the relative path must not start with ''..'''
                                              type: string
                                            resourceFieldRef:
                                              description: 'Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.'
                                              properties:
                                                containerName:
                                                  description: 'Container name: required for volumes, optional for env vars'
                                                  type: string
                                                divisor:
                                                  anyOf:
                                                  - type: integer
                                                  - type: string
                                                  description: Specifies the output format of the exposed resources, defaults to "1"
                                                  pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                                                  x-kubernetes-int-or-string: true
                                                resource:
                                                  description: 'Required: resource to select'
                                                  type: string
                                              required:
                                              - resource
                                              type: object
                                          required:
                                          - path
                                          type: object
                                        type: array
                                      volumeLabel:
                                        description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                                        type: string
                                    type: object
                                  emptyDisk:
                                    description: 'EmptyDisk represents a temporary disk which shares the vmis lifecycle. More info: https://kubevirt.gitbooks.io/user-guide/disks-and-volumes.html'
                                    properties:
                                      capacity:
                                        anyOf:
                                        - type: integer
                                        - type: string
                                        description: Capacity of the sparse disk.
                                        pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                                        x-kubernetes-int-or-string: true
                                    required:
                                    - capacity
                                    type: object
                                  ephemeral:
                                    description: Ephemeral is a special volume source that "wraps" specified source and provides copy-on-write image on top of it.
                                    properties:
                                      persistentVolumeClaim:
                                        description: 'PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. Directly attached to the vmi via qemu. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                                        properties:
                                          claimName:
                                            description: 'ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                                            type: string
                                          readOnly:
                                            description: Will force the ReadOnly setting in VolumeMounts. Default false.
                                            type: boolean
                                        required:
                                        - claimName
                                        type: object
                                    type: object
                                  hostDisk:
                                    description: HostDisk represents a disk created on the cluster level
                                    properties:
                                      capacity:
                                        anyOf:
                                        - type: integer
                                        - type: string
                                        description: Capacity of the sparse disk
                                        pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                                        x-kubernetes-int-or-string: true
                                      path:
                                        description: The path to HostDisk image located on the cluster
                                        type: string
                                      shared:
                                        description: Shared indicate whether the path is shared between nodes
                                        type: boolean
                                      type:
                                        description: Contains information if disk.img exists or should be created allowed options are 'Disk' and 'DiskOrCreate'
                                        type: string
                                    required:
                                    - path
                                    - type
                                    type: object
                                  name:
                                    description: 'Volume''s name. Must be a DNS_LABEL and unique within the vmi. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names'
                                    type: string
                                  persistentVolumeClaim:
                                    description: 'PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. Directly attached to the vmi via qemu. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                                    properties:
                                      claimName:
                                        description: 'ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                                        type: string
                                      readOnly:
                                        description: Will force the ReadOnly setting in VolumeMounts. Default false.
                                        type: boolean
                                    required:
                                    - claimName
                                    type: object
                                  secret:
                                    description: 'SecretVolumeSource represents a reference to a secret data in the same namespace. More info: https://kubernetes.io/docs/concepts/configuration/secret/'
                                    properties:
                                      optional:
                                        description: Specify whether the Secret or it's keys must be defined
                                        type: boolean
                                      secretName:
                                        description: 'Name of the secret in the pod''s namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret'
                                        type: string
                                      volumeLabel:
                                        description: The volume label of the resulting disk inside the VMI. Different bootstrapping mechanisms require different values. Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
                                        type: string
                                    type: object
                                  serviceAccount:
                                    description: 'ServiceAccountVolumeSource represents a reference to a service account. There can only be one volume of this type! More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/'
                                    properties:
                                      serviceAccountName:
                                        description: 'Name of the service account in the pod''s namespace to use. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/'
                                        type: string
                                    type: object
                                required:
                                - name
                                type: object
                              type: array
                          required:
                          - domain
                          type: object
                      type: object
                  required:
                  - template
                  type: object
                status:
                  description: Status holds the current state of the controller and brief information about its associated VirtualMachineInstance
                  properties:
                    conditions:
                      description: Hold the state information of the VirtualMachine and its VirtualMachineInstance
                      items:
                        description: VirtualMachineCondition represents the state of VirtualMachine
                        properties:
                          lastProbeTime:
                            format: date-time
                            nullable: true
                            type: string
                          lastTransitionTime:
                            format: date-time
                            nullable: true
                            type: string
                          message:
                            type: string
                          reason:
                            type: string
                          status:
                            type: string
                          type:
                            type: string
                        required:
                        - status
                        - type
                        type: object
                      type: array
                    created:
                      description: Created indicates if the virtual machine is created in the cluster
                      type: boolean
                    ready:
                      description: Ready indicates if the virtual machine is running and ready
                      type: boolean
                    snapshotInProgress:
                      description: SnapshotInProgress is the name of the VirtualMachineSnapshot currently executing
                      type: string
                    stateChangeRequests:
                      description: StateChangeRequests indicates a list of actions that should be taken on a VMI e.g. stop a specific VMI then start a new one.
                      items:
                        properties:
                          action:
                            description: Indicates the type of action that is requested. e.g. Start or Stop
                            type: string
                          data:
                            additionalProperties:
                              type: string
                            description: Provides additional data in order to perform the Action
                            type: object
                          uid:
                            description: Indicates the UUID of an existing Virtual Machine Instance that this change request applies to -- if applicable
                            type: string
                        required:
                        - action
                        type: object
                      type: array
                    volumeSnapshotStatuses:
                      description: VolumeSnapshotStatuses indicates a list of statuses whether snapshotting is supported by each volume.
                      items:
                        properties:
                          enabled:
                            description: True if the volume supports snapshotting
                            type: boolean
                          name:
                            description: Volume name
                            type: string
                          reason:
                            description: Empty if snapshotting is enabled, contains reason otherwise
                            type: string
                        required:
                        - enabled
                        - name
                        type: object
                      type: array
                  type: object
              required:
              - spec
              type: object
          type: object
        virtualMachineSnapshotName:
          type: string
        volumeBackups:
          items:
            description: VolumeBackup contains the data neeed to restore a PVC
            properties:
              persistentVolumeClaim:
                description: PersistentVolumeClaim is a user's request for and claim to a persistent volume
                properties:
                  apiVersion:
                    description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
                    type: string
                  kind:
                    description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
                    type: string
                  metadata:
                    description: 'Standard object''s metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata'
                    type: object
                  spec:
                    description: 'Spec defines the desired characteristics of a volume requested by a pod author. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                    properties:
                      accessModes:
                        description: 'AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1'
                        items:
                          type: string
                        type: array
                      dataSource:
                        description: This field requires the VolumeSnapshotDataSource alpha feature gate to be enabled and currently VolumeSnapshot is the only supported data source. If the provisioner can support VolumeSnapshot data source, it will create a new volume and data will be restored to the volume at the same time. If the provisioner does not support VolumeSnapshot data source, volume will not be created and the failure will be reported as an event. In the future, we plan to support more data source types and the behavior of the provisioner may change.
                        properties:
                          apiGroup:
                            description: APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.
                            type: string
                          kind:
                            description: Kind is the type of resource being referenced
                            type: string
                          name:
                            description: Name is the name of resource being referenced
                            type: string
                        required:
                        - kind
                        - name
                        type: object
                      resources:
                        description: 'Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources'
                        properties:
                          limits:
                            additionalProperties:
                              anyOf:
                              - type: integer
                              - type: string
                              pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                              x-kubernetes-int-or-string: true
                            description: 'Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/'
                            type: object
                          requests:
                            additionalProperties:
                              anyOf:
                              - type: integer
                              - type: string
                              pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                              x-kubernetes-int-or-string: true
                            description: 'Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/'
                            type: object
                        type: object
                      selector:
                        description: A label query over volumes to consider for binding.
                        properties:
                          matchExpressions:
                            description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
                            items:
                              description: A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.
                              properties:
                                key:
                                  description: key is the label key that the selector applies to.
                                  type: string
                                operator:
                                  description: operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.
                                  type: string
                                values:
                                  description: values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
                                  items:
                                    type: string
                                  type: array
                              required:
                              - key
                              - operator
                              type: object
                            type: array
                          matchLabels:
                            additionalProperties:
                              type: string
                            description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.
                            type: object
                        type: object
                      storageClassName:
                        description: 'Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1'
                        type: string
                      volumeMode:
                        description: volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec. This is a beta feature.
                        type: string
                      volumeName:
                        description: VolumeName is the binding reference to the PersistentVolume backing this claim.
                        type: string
                    type: object
                  status:
                    description: 'Status represents the current information/status of a persistent volume claim. Read-only. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims'
                    properties:
                      accessModes:
                        description: 'AccessModes contains the actual access modes the volume backing the PVC has. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1'
                        items:
                          type: string
                        type: array
                      capacity:
                        additionalProperties:
                          anyOf:
                          - type: integer
                          - type: string
                          pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                          x-kubernetes-int-or-string: true
                        description: Represents the actual resources of the underlying volume.
                        type: object
                      conditions:
                        description: Current Condition of persistent volume claim. If underlying persistent volume is being resized then the Condition will be set to 'ResizeStarted'.
                        items:
                          description: PersistentVolumeClaimCondition contails details about state of pvc
                          properties:
                            lastProbeTime:
                              description: Last time we probed the condition.
                              format: date-time
                              type: string
                            lastTransitionTime:
                              description: Last time the condition transitioned from one status to another.
                              format: date-time
                              type: string
                            message:
                              description: Human-readable message indicating details about last transition.
                              type: string
                            reason:
                              description: Unique, this should be a short, machine understandable string that gives the reason for condition's last transition. If it reports "ResizeStarted" that means the underlying persistent volume is being resized.
                              type: string
                            status:
                              type: string
                            type:
                              description: PersistentVolumeClaimConditionType is a valid value of PersistentVolumeClaimCondition.Type
                              type: string
                          required:
                          - status
                          - type
                          type: object
                        type: array
                      phase:
                        description: Phase represents the current phase of PersistentVolumeClaim.
                        type: string
                    type: object
                type: object
              volumeName:
                type: string
              volumeSnapshotName:
                type: string
            required:
            - persistentVolumeClaim
            - volumeName
            type: object
          type: array
      required:
      - source
      type: object
    status:
      description: VirtualMachineSnapshotContentStatus is the status for a VirtualMachineSnapshotStatus resource
      properties:
        creationTime:
          format: date-time
          nullable: true
          type: string
        error:
          description: Error is the last error encountered during the snapshot/restore
          properties:
            message:
              type: string
            time:
              format: date-time
              type: string
          type: object
        readyToUse:
          type: boolean
        volumeSnapshotStatus:
          items:
            description: VolumeSnapshotStatus is the status of a VolumeSnapshot
            properties:
              creationTime:
                format: date-time
                nullable: true
                type: string
              error:
                description: Error is the last error encountered during the snapshot/restore
                properties:
                  message:
                    type: string
                  time:
                    format: date-time
                    type: string
                type: object
              readyToUse:
                type: boolean
              volumeSnapshotName:
                type: string
            required:
            - volumeSnapshotName
            type: object
          type: array
      type: object
  required:
  - spec
  type: object
`,
}
