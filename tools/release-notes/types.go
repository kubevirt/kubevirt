package main

import (
	"os"

	"github.com/google/go-github/v32/github"
)

var githubClient *github.Client

var projectNames = []*projectName{
	{"KUBEVIRT", "kubevirt"},
	{"CDI", "containerized-data-importer"},
	{"NETWORK_ADDONS", "cluster-network-addons-operator"},
	{"SSP", "ssp-operator"},
	{"NMO", "node-maintenance-operator"},
	{"HPPO", "hostpath-provisioner-operator"},
	{"HPP", "hostpath-provisioner"},
	{"VM_IMPORT", "vm-import-operator"},
}

type projectName struct {
	short string
	name  string
}

type project struct {
	short       string
	name        string
	currentTag  string
	previousTag string
	tagBranch   string

	repoDir string
	repoUrl string

	// github cached results
	allReleases []*github.RepositoryRelease
	allBranches []*github.Branch
}

type releaseData struct {
	org      string
	hco      project
	projects []*project

	outFile *os.File
}
