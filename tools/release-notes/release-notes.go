package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/golang/glog"
	"github.com/joho/godotenv"

	"github.com/kubevirt/hyperconverged-cluster-operator/tools/release-notes/git"
)

func (r *releaseData) writeHeader() {
	tagUrl := fmt.Sprintf("https://github.com/kubevirt/hyperconverged-cluster-operator/releases/tag/%s", r.hco.CurrentTag)

	numChanges, err := r.hco.GetNumChanges()
	if err != nil {
		glog.Fatalf("ERROR failed to get num changes: %s\n", err)
	}

	typeOfChanges, err := r.hco.GetTypeOfChanges()
	if err != nil {
		glog.Fatalf("ERROR failed to get type of changes: %s\n", err)
	}

	fmt.Fprintf(r.writer, "This release follows %s and consists of %d changes, leading to %s.\n\n", r.hco.PreviousTag, numChanges, typeOfChanges)
	fmt.Fprintf(r.writer, "The source code and selected binaries are available for download at: %s.\n\n", tagUrl)
	fmt.Fprintln(r.writer, "The primary release artifact of hyperconverged-cluster-operator is the git tree. The release tag is")
	fmt.Fprintf(r.writer, "signed and can be verified using `git tag -v %s`.\n\n", r.hco.CurrentTag)
	fmt.Fprintln(r.writer, "Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.")
	fmt.Fprintln(r.writer)
}

func (r *releaseData) writeHcoChanges() {
	releaseNotes, err := r.hco.GetReleaseNotes()
	if err != nil {
		glog.Fatalf("ERROR failed to get release notes of %s: %s\n", r.hco.Name, err)
	}

	fmt.Fprintf(r.writer, "### %s - %s\n", r.hco.Name, r.hco.CurrentTag)
	if len(releaseNotes) > 0 {
		for _, note := range releaseNotes {
			fmt.Fprintf(r.writer, "- %s\n", note)
		}
	} else {
		fmt.Fprintln(r.writer, "No notable changes")
	}
	fmt.Fprintln(r.writer)
}

func (r *releaseData) writeOtherChangesIfVersionUpdated(g *git.Project) {
	releaseNotes, err := g.GetReleaseNotes()
	if err != nil {
		glog.Fatalf("ERROR failed to get release notes of %s: %s\n", g.Name, err)
	}

	fmt.Fprintf(r.writer, "### %s: %s -> %s\n", g.Name, g.PreviousTag, g.CurrentTag)
	if len(releaseNotes) > 0 {
		for _, note := range releaseNotes {
			fmt.Fprintf(r.writer, "- %s\n", note)
		}
	} else {
		fmt.Fprintln(r.writer, "No notable changes")
	}
}

func (r *releaseData) writeOtherChanges() {
	for _, p := range r.projects {
		if len(p.PreviousTag) == 0 || p.PreviousTag == p.CurrentTag {
			fmt.Fprintf(r.writer, "### %s: %s\n", p.Name, p.CurrentTag)
			fmt.Fprintln(r.writer, "Not updated")
		} else {
			r.writeOtherChangesIfVersionUpdated(p)
		}

		fmt.Fprintln(r.writer)
	}
}

func (r *releaseData) getConfig(tag string) map[string]string {
	err := r.hco.SwitchToTag(tag)
	if err != nil {
		glog.Fatalf("ERROR failed to switch to tag %s in %s: %s\n", tag, r.hco.Name, err)
	}

	config, err := godotenv.Read(r.hco.Directory + "/hack/config")
	if err != nil {
		glog.Fatalf("ERROR failed to read /hack/config file : %s\n", err)
	}

	return config
}

func (r *releaseData) findProjectsCurrentAndPreviousReleases() {
	newConfig := r.getConfig(r.hco.CurrentTag)
	oldConfig := r.getConfig(r.hco.PreviousTag)

	for _, p := range r.projects {
		p.CurrentTag = newConfig[p.Short+"_VERSION"]
		p.PreviousTag = oldConfig[p.Short+"_VERSION"]
	}
}

func (r *releaseData) writeNotableChanges() {
	fmt.Fprintln(r.writer, "Notable changes")
	fmt.Fprintln(r.writer, "---------------")
	fmt.Fprintln(r.writer)

	r.writeHcoChanges()
	r.findProjectsCurrentAndPreviousReleases()
	r.writeOtherChanges()
}

func isNotBot(contributor string) bool {
	bots := []string{
		"kubevirt-bot",
		"hco-bot",
	}

	for _, bot := range bots {
		if strings.Contains(contributor, bot) {
			return false
		}
	}

	return true
}

func (r *releaseData) writeContributors() {
	contributorList, err := r.hco.GetContributors()
	if err != nil {
		glog.Fatalf("ERROR failed to get contributor list: %s\n", err)
	}

	var sb strings.Builder
	numContributors := 0
	for _, contributor := range contributorList {
		if isNotBot(contributor) && len(contributor) != 0 {
			numContributors++
			sb.WriteString(fmt.Sprintf(" - %s\n", strings.TrimSpace(contributor)))
		}
	}

	fmt.Fprintln(r.writer)
	fmt.Fprintln(r.writer, "Contributors")
	fmt.Fprintln(r.writer, "------------")
	fmt.Fprintf(r.writer, "%d people contributed to this HCO release:\n\n", numContributors)
	fmt.Fprintf(r.writer, sb.String())
	fmt.Fprintln(r.writer)
}

const additionalResources = `Additional Resources
--------------------
- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]


Contributing: https://github.com/kubevirt/hyperconverged-cluster-operator/blob/main/CONTRIBUTING.md

License: https://github.com/kubevirt/hyperconverged-cluster-operator/blob/main/LICENSE

---
`

func (r *releaseData) generateReleaseNotes() {
	releaseNotesFile := fmt.Sprintf("%s-release-notes.md", r.hco.CurrentTag)

	var err error
	r.writer, err = os.OpenFile(releaseNotesFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		glog.Fatalf("ERROR failed to create release notes file: %s\n", err)
	}

	r.writeHeader()
	r.writeNotableChanges()
	r.writeContributors()

	fmt.Fprint(r.writer, additionalResources)
}

func createProjects(baseDir string, token string) []*git.Project {
	var projects []*git.Project
	for _, n := range projectNames {
		projects = append(projects, git.InitProject("kubevirt", n.name, n.short, baseDir+n.name, "", token))
	}

	return projects
}

func getToken(githubTokenPath string) string {
	tokenBytes, err := ioutil.ReadFile(githubTokenPath)
	if err != nil {
		log.Fatalf("ERROR accessing github token: %s ", err)
	}
	return strings.TrimSpace(string(tokenBytes))
}

func parseArguments() *releaseData {
	release := flag.String("release", "", "Release tag. Must be a valid semver. The branch is automatically detected from the major and minor release")
	cacheDir := flag.String("cache-dir", "/tmp/release-tool", "The base directory used to cache git repos in")
	githubTokenFile := flag.String("github-token-file", "", "file containing the github token.")

	flag.Parse()

	if *githubTokenFile == "" {
		log.Fatal("--github-token-file is a required argument")
	} else if *release == "" {
		log.Fatal("--release is a required argument")
	}

	baseDir := path.Join(*cacheDir, "kubevirt")
	hco := "hyperconverged-cluster-operator"

	gitToken := getToken(*githubTokenFile)

	return &releaseData{
		hco:      git.InitProject("kubevirt", hco, "HCO", baseDir+hco, *release, gitToken),
		projects: createProjects(baseDir, gitToken),
	}
}

func (r *releaseData) checkoutProjects() {
	err := r.hco.CheckoutUpstream()
	if err != nil {
		log.Fatalf("ERROR checking out upstream: %s\n", err)
	}
	err = r.hco.CheckCurrentTagExists()
	if err != nil {
		log.Fatalf("ERROR checking out upstream: %s\n", err)
	}

	for _, p := range r.projects {
		err = p.CheckoutUpstream()
		if err != nil {
			log.Fatalf("ERROR checking out upstream: %s\n", err)
		}
	}
}

func main() {
	r := parseArguments()
	r.checkoutProjects()

	err := r.hco.VerifySemverTag()
	if err != nil {
		log.Fatalf("ERROR requested tag invalid: %s\n", err)
	}

	r.generateReleaseNotes()
}
