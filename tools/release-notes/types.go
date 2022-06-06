package main

import (
	"io"

	"github.com/kubevirt/hyperconverged-cluster-operator/tools/release-notes/git"
)

var projectNames = []*projectName{
	{"KUBEVIRT", "kubevirt"},
	{"CDI", "containerized-data-importer"},
	{"NETWORK_ADDONS", "cluster-network-addons-operator"},
	{"SSP", "ssp-operator"},
	{"TTO", "tekton-tasks-operator"},
	{"HPPO", "hostpath-provisioner-operator"},
	{"HPP", "hostpath-provisioner"},
	{"VM_IMPORT", "vm-import-operator"},
}

type projectName struct {
	short string
	name  string
}

type releaseData struct {
	hco      *git.Project
	projects []*git.Project

	writer io.Writer
}
