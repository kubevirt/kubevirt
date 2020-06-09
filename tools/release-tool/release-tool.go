package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os/exec"

	"github.com/Masterminds/semver"
	"github.com/google/go-github/v32/github"
)

type releaseData struct {
	remote    string
	remoteUrl string
	repo      string
	org       string
	branch    string
	tag       string

	dryRun bool
}

func gitCommand(arg ...string) error {
	log.Printf("executing 'git %v", arg)
	cmd := exec.Command("git", arg...)
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ERROR: git command output: %s : %v", string(bytes), err)
		return err
	}
	return nil
}

func (r *releaseData) addUpstreamRemote() error {
	// add upstream remote branch
	err := gitCommand("remote", "add", r.remote, r.remoteUrl)
	if err != nil {
		return err
	}

	err = gitCommand("fetch", r.remote)
	if err != nil {
		return err
	}
	return nil
}

func (r *releaseData) removeUpstreamRemote() error {
	// remove upstream remote branch
	return gitCommand("remote", "remove", r.remote)
}

func (r *releaseData) cutNewBranch() error {
	client := github.NewClient(nil)

	branches, _, err := client.Repositories.ListBranches(context.Background(), r.org, r.repo, &github.BranchListOptions{
		ListOptions: github.ListOptions{
			PerPage: 10000,
		},
	})
	if err != nil {
		return err
	}

	for _, b := range branches {
		if b.Name != nil && *b.Name == r.branch {
			return fmt.Errorf("Release branch [%s] already exists", r.branch)
		}
	}

	// checkout remote branch
	err = r.addUpstreamRemote()
	if err != nil {
		return err
	}

	// TODO check for blockers on master
	// TODO cut branch

	return nil
}

func (r *releaseData) cutNewTag() error {
	client := github.NewClient(nil)

	// must be a valid semver version
	tagSemver, err := semver.NewVersion(r.tag)
	if err != nil {
		return err
	}

	expectedBranch := fmt.Sprintf("release-%d.%d", tagSemver.Major(), tagSemver.Minor())

	releases, _, err := client.Repositories.ListReleases(context.Background(), r.org, r.repo, &github.ListOptions{PerPage: 10000})

	for _, release := range releases {

		if *release.TagName == r.tag {
			return fmt.Errorf("Release tag [%s] already exists", r.tag)
		}
	}

	branches, _, err := client.Repositories.ListBranches(context.Background(), r.org, r.repo, &github.BranchListOptions{
		ListOptions: github.ListOptions{
			PerPage: 10000,
		},
	})
	if err != nil {
		return err
	}

	var releaseBranch *github.Branch
	for _, branch := range branches {
		if branch.Name != nil && *branch.Name == expectedBranch {
			releaseBranch = branch
			break
		}
	}

	if releaseBranch == nil {
		return fmt.Errorf("release branch [%s] not found for new release [%s]", expectedBranch, r.tag)
	}

	// checkout remote branch
	err = r.addUpstreamRemote()
	defer r.removeUpstreamRemote()
	if err != nil {
		return err
	}

	// TODO release notes
	// find the previous official release, and current offical release
	// and use the hack/release-annouce.sh script
	// TODO check for blockers on release branch
	// TODO create tag using release branch

	return nil
}

func (r *releaseData) releaseNotes() string {

	notes := ""

	return notes
}

func (r *releaseData) printData() {

	if r.dryRun {
		log.Print("DRY-RUN")
	}

	log.Print("Input Data")

	log.Printf("\tremoteUrl: %s", r.remoteUrl)
	log.Printf("\tnewTag: %s", r.tag)
	log.Printf("\tnewBranch: %s", r.branch)
	log.Printf("\torg: %s", r.org)
	log.Printf("\trepo: %s", r.repo)

}

func main() {
	newBranch := flag.String("new-branch", "", "New branch to cut from master.")
	releaseTag := flag.String("release-tag", "", "New release tag. Must be a valid semver. The branch is automatically detected from the major and minor release")
	org := flag.String("project-org", "kubevirt", "The project org")
	repo := flag.String("repo", "kubevirt", "The project repo")
	dryRun := flag.Bool("dry-run", true, "Should this be a dry run")

	flag.Parse()

	remoteUrl := fmt.Sprintf("git@github.com:%s/%s.git", *org, *repo)
	remote := "release-tool-upstream"
	if *dryRun {
		remoteUrl = fmt.Sprintf("https://github.com/%s/%s.git", *org, *repo)
		remote = "dry-run-release-tool-upstream"
	}

	r := releaseData{
		remote:    remote,
		remoteUrl: remoteUrl,
		repo:      *repo,
		org:       *org,
		branch:    *newBranch,
		tag:       *releaseTag,

		dryRun: *dryRun,
	}

	r.printData()

	if *newBranch != "" {
		err := r.cutNewBranch()
		if err != nil {
			log.Fatal(fmt.Printf("ERROR: %v", err))
		}
	}

	if *releaseTag != "" {
		err := r.cutNewTag()
		if err != nil {
			log.Fatal(fmt.Printf("ERROR: %v", err))
		}
	}
}
