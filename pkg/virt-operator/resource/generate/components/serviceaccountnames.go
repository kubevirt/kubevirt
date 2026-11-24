package components

import "fmt"

const (
	ApiServiceAccountName                       = "kubevirt-apiserver"
	ControllerServiceAccountName                = "kubevirt-controller"
	ExportProxyServiceAccountName               = "kubevirt-exportproxy"
	SynchronizationControllerServiceAccountName = "kubevirt-synchronization-controller"
	HandlerServiceAccountName                   = "kubevirt-handler"
	OperatorServiceAccountName                  = "kubevirt-operator"
)

// ServiceAccountUsername is the userInfo.username the apiserver records for a
// namespaced ServiceAccount.
func ServiceAccountUsername(namespace, name string) string {
	return fmt.Sprintf("system:serviceaccount:%s:%s", namespace, name)
}
