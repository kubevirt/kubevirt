package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func (r *releaseData) writeHeader(span string) error {
	tagUrl := fmt.Sprintf("https://github.com/kubevirt/%s/releases/tag/%s", "hyperconverged-cluster-operator", r.hco.currentTag)

	numChanges, err := r.hco.gitGetNumChanges(span)
	if err != nil {
		return err
	}

	typeOfChanges, err := r.hco.gitGetTypeOfChanges(span)
	if err != nil {
		return err
	}

	r.outFile.WriteString(fmt.Sprintf("This release follows %s and consists of %d changes, leading to %s.\n", r.hco.previousTag, numChanges, typeOfChanges))
	r.outFile.WriteString("\n")
	r.outFile.WriteString(fmt.Sprintf("The source code and selected binaries are available for download at: %s.\n", tagUrl))
	r.outFile.WriteString("\n")
	r.outFile.WriteString("The primary release artifact of hyperconverged-cluster-operator is the git tree. The release tag is\n")
	r.outFile.WriteString(fmt.Sprintf("signed and can be verified using `git tag -v %s`.\n", r.hco.currentTag))
	r.outFile.WriteString("\n")
	r.outFile.WriteString("Pre-built containers are published on Quay and can be viewed at: <https://quay.io/kubevirt/>.\n")
	r.outFile.WriteString("\n")

	return nil
}

func (r *releaseData) writeHcoChanges(span string) error {
	releaseNotes, err := r.hco.gitGetReleaseNotes(span)
	if err != nil {
		return err
	}

	r.outFile.WriteString(fmt.Sprintf("### %s - %s\n", r.hco.name, r.hco.currentTag))
	if len(releaseNotes) > 0 {
		for _, note := range releaseNotes {
			r.outFile.WriteString(fmt.Sprintf("- %s\n", note))
		}
	} else {
		r.outFile.WriteString("No notable changes\n")
	}
	r.outFile.WriteString("\n")

	return nil
}

func (p *project) writeOtherChangesIfVersionUpdated(f *os.File) error {
	span := fmt.Sprintf("%s..%s", p.previousTag, p.currentTag)
	releaseNotes, err := p.gitGetReleaseNotes(span)
	if err != nil {
		return err
	}

	f.WriteString(fmt.Sprintf("### %s: %s -> %s\n", p.name, p.previousTag, p.currentTag))
	if len(releaseNotes) > 0 {
		for _, note := range releaseNotes {
			f.WriteString(fmt.Sprintf("- %s\n", note))
		}
	} else {
		f.WriteString("No notable changes\n")
	}

	return nil
}

func (r *releaseData) writeOtherChanges() error {
	for _, p := range r.projects {
		if len(p.previousTag) == 0 || p.previousTag == p.currentTag {
			r.outFile.WriteString(fmt.Sprintf("### %s: %s\n", p.name, p.currentTag))
			r.outFile.WriteString("Not updated\n")
		} else {
			p.writeOtherChangesIfVersionUpdated(r.outFile)
		}

		r.outFile.WriteString("\n")
	}

	return nil
}

func (r *releaseData) getConfig(branch string) (map[string]string, error) {
	err := r.hco.gitSwitchToBranch(branch)
	if err != nil {
		return nil, err
	}

	config, err := godotenv.Read(r.hco.repoDir + "/hack/config")
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (r *releaseData) findProjectsCurrentAndPreviousReleases() error {
	newConfig, err := r.getConfig(r.hco.currentTag)
	if err != nil {
		return err
	}
	oldConfig, err := r.getConfig(r.hco.previousTag)
	if err != nil {
		return err
	}

	for _, p := range r.projects {
		p.currentTag = newConfig[p.short+"_VERSION"]
		p.previousTag = oldConfig[p.short+"_VERSION"]
	}

	return nil
}

func (r *releaseData) writeNotableChanges(span string) error {
	r.outFile.WriteString("Notable changes\n---------------\n")
	r.outFile.WriteString("\n")

	err := r.writeHcoChanges(span)
	if err != nil {
		return err
	}

	err = r.findProjectsCurrentAndPreviousReleases()
	if err != nil {
		return err
	}

	err = r.writeOtherChanges()
	if err != nil {
		return err
	}

	return nil
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

func (r *releaseData) writeContributors(span string) error {
	contributorList, err := r.hco.gitGetContributors(span)
	if err != nil {
		return err
	}

	var sb strings.Builder
	numContributors := 0
	for _, contributor := range contributorList {
		if isNotBot(contributor) && len(contributor) != 0 {
			numContributors++
			sb.WriteString(fmt.Sprintf(" - %s\n", strings.TrimSpace(contributor)))
		}
	}

	r.outFile.WriteString("\n")
	r.outFile.WriteString("Contributors\n------------\n")
	r.outFile.WriteString(fmt.Sprintf("%d people contributed to this HCO release:\n\n", numContributors))
	r.outFile.WriteString(sb.String())
	r.outFile.WriteString("\n")

	return nil
}

func (r *releaseData) writeAdditionalResources() {
	additionalResources := fmt.Sprintf(`Additional Resources
--------------------
- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]


[contributing]: https://github.com/kubevirt/%s/blob/main/CONTRIBUTING.md
[license]: https://github.com/kubevirt/%s/blob/main/LICENSE
---
`, "hyperconverged-cluster-operator", "hyperconverged-cluster-operator")

	r.outFile.WriteString(additionalResources)
}

func (r *releaseData) generateReleaseNotes() error {
	releaseNotesFile := fmt.Sprintf("%s-release-notes.md", r.hco.currentTag)

	var err error
	r.outFile, err = os.Create(releaseNotesFile)
	if err != nil {
		return err
	}
	defer r.outFile.Close()

	span := fmt.Sprintf("%s..%s", r.hco.previousTag, r.hco.currentTag)

	err = r.writeHeader(span)
	if err != nil {
		return err
	}

	err = r.writeNotableChanges(span)
	if err != nil {
		return err
	}

	err = r.writeContributors(span)
	if err != nil {
		return err
	}

	r.writeAdditionalResources()

	return nil
}

func createProjects(baseDir string, token string) []*project {
	var projects []*project
	for _, n := range projectNames {
		projects = append(projects, &project{
			short:   n.short,
			name:    n.name,
			repoDir: baseDir + n.name,
			repoUrl: fmt.Sprintf("https://%s@github.com/kubevirt/%s.git", token, n.name),
		})
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

	baseDir := fmt.Sprintf("%s/%s/", *cacheDir, "kubevirt")
	hco := "hyperconverged-cluster-operator"

	gitToken := getToken(*githubTokenFile)
	gitHubInitClient(gitToken)

	return &releaseData{
		org: "kubevirt",
		hco: project{
			name:       hco,
			currentTag: *release,
			repoDir:    baseDir + hco,
			repoUrl:    fmt.Sprintf("https://%s@github.com/kubevirt/%s.git", gitToken, hco),
		},
		projects: createProjects(baseDir, gitToken),
	}
}

func (r *releaseData) checkoutProjects() {
	err := r.hco.gitCheckoutUpstream()
	if err != nil {
		log.Fatalf("ERROR checking out upstream: %s\n", err)
	}
	err = r.hco.gitCheckCurrentTagExists()
	if err != nil {
		log.Fatalf("ERROR checking out upstream: %s\n", err)
	}

	for _, p := range r.projects {
		err = p.gitCheckoutUpstream()
		if err != nil {
			log.Fatalf("ERROR checking out upstream: %s\n", err)
		}
	}
}

func main() {
	r := parseArguments()
	r.checkoutProjects()

	err := r.semverVerifyTag()
	if err != nil {
		log.Fatalf("ERROR generating release notes: %s\n", err)
	}

	err = r.generateReleaseNotes()
	if err != nil {
		log.Fatalf("ERROR generating release notes: %s\n", err)
	}
}
