package manifests

import _ "embed"

//go:embed serviceaccount.yaml
var ServiceAccount []byte

//go:embed rbac.yaml
var RBAC []byte

//go:embed deviceclass.yaml
var DeviceClass []byte

//go:embed daemonset.yaml
var DaemonSet []byte
