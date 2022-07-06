package git

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Project struct {
	Owner       string
	Name        string
	Short       string
	Directory   string
	Remote      string
	CurrentTag  string
	PreviousTag string

	repository *git.Repository
	worktree   *git.Worktree
	commits    []*object.Commit

	github *gh
}

func InitProject(owner string, name string, short string, directory string, tag string, token string) *Project {
	p := new(Project)

	p.Owner = owner
	p.Name = name
	p.Short = short
	p.Directory = directory
	p.Remote = fmt.Sprintf("https://github.com/%s/%s.git", owner, name)
	p.CurrentTag = tag

	p.github = initGitHub(owner, name, token)

	return p
}

func (g *Project) CheckoutUpstream() error {
	var err error
	g.repository, err = git.PlainOpen(g.Directory)

	if err == nil {
		err = g.updateFromUpstream()
		if err != nil {
			return err
		}
	} else {
		err = g.cloneUpstream()
		if err != nil {
			return err
		}
	}

	return g.updateConfig()
}

func (g *Project) updateFromUpstream() error {
	var err error
	g.worktree, err = g.repository.Worktree()
	if err != nil {
		return err
	}

	main := plumbing.NewBranchReferenceName("main")

	err = g.worktree.Checkout(&git.CheckoutOptions{Branch: main})
	if err != nil {
		master := plumbing.NewBranchReferenceName("master")
		err = g.worktree.Checkout(&git.CheckoutOptions{Branch: master})
		if err != nil {
			return err
		}
	}

	err = g.worktree.Pull(&git.PullOptions{Progress: os.Stdout})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}

func (g *Project) cloneUpstream() error {
	err := g.createFreshGitDirectory()
	if err != nil {
		return err
	}

	options := &git.CloneOptions{
		URL:      g.Remote,
		Progress: os.Stdout,
	}

	g.repository, err = git.PlainClone(g.Directory, false, options)
	if err != nil {
		return err
	}

	g.worktree, err = g.repository.Worktree()
	if err != nil {
		return err
	}

	return nil
}

func (g *Project) createFreshGitDirectory() error {
	// start fresh because checkout doesn't exist or is corrupted
	_ = os.RemoveAll(g.Directory)
	return os.MkdirAll(g.Directory, 0755)
}

func (g *Project) updateConfig() error {
	cfg, err := g.repository.Config()
	if err != nil {
		return err
	}

	cfg.Raw.Sections = append(cfg.Raw.Sections, &config.Section{
		Name: "diff",
		Options: config.Options{
			&config.Option{
				Key:   "renameLimit",
				Value: "999999",
			},
		},
	})

	err = g.repository.SetConfig(cfg)
	if err != nil {
		return err
	}

	return nil
}

func (g *Project) GetContributors() ([]string, error) {
	err := g.fetchCommits()
	if err != nil {
		return nil, err
	}

	contributors := make(map[string]int)
	for _, commit := range g.commits {
		id := fmt.Sprintf("%s %s", commit.Author.Name, commit.Author.Email)
		contributors[id]++
	}

	keys := make([]string, 0, len(contributors))
	for key := range contributors {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return contributors[keys[i]] > contributors[keys[j]] })

	res := make([]string, len(keys))
	for _, key := range keys {
		res = append(res, fmt.Sprintf("%d %s", contributors[key], key))
	}

	return res, nil
}

func (g *Project) GetReleaseNotes() ([]string, error) {
	err := g.fetchCommits()
	if err != nil {
		return nil, err
	}

	releaseNotes := make([]string, 0)

	pattern := regexp.MustCompile(`\(#\d+\)`)

	for _, commit := range g.commits {
		matches := pattern.FindAllString(commit.Message, -1)

		if len(matches) > 0 {
			releaseNotes = g.releaseNoteFromSquashCommit(releaseNotes, matches)
		} else if strings.Contains(commit.Message, "Merge pull request #") {
			releaseNotes = g.releaseNoteFromMergeCommit(releaseNotes, commit.Message)
		}
	}

	return releaseNotes, nil
}

func (g *Project) releaseNoteFromSquashCommit(releaseNotes []string, matches []string) []string {
	for _, match := range matches {
		// Get the Pull Request number from a Squash Commit message information (p.e. "(#1820)") and convert it to int
		pullRequestNumber, err := strconv.Atoi(match[2 : len(match)-1])
		if err != nil {
			continue
		}
		releaseNotes = g.readReleaseNote(releaseNotes, pullRequestNumber)
	}

	return releaseNotes
}

// releaseNoteFromMergeCommit takes a Merge Commit message
// (p.e. "Merge pull request #7494 from prnaraya/vm-force-stop"), splits it by spaces, then it fetches the part with the
// Pull Request number and removes the prefix "#". Finally, it executes readReleaseNote to read the release note of that
// Pull Request on GitHub and append it in the given release notes list.
func (g *Project) releaseNoteFromMergeCommit(releaseNotes []string, line string) []string {
	pr := strings.Split(line, " ")
	pullRequestNumber, err := strconv.Atoi(strings.TrimPrefix(pr[3], "#"))
	if err != nil {
		return releaseNotes
	}

	return g.readReleaseNote(releaseNotes, pullRequestNumber)
}

func (g *Project) readReleaseNote(releaseNotes []string, pullRequestNumber int) []string {
	note, err := g.github.getReleaseNote(pullRequestNumber)
	if err != nil {
		return releaseNotes
	}
	if note != "" {
		releaseNotes = append(releaseNotes, note)
	}

	return releaseNotes
}

func (g *Project) GetNumChanges() (int, error) {
	err := g.fetchCommits()
	if err != nil {
		return -1, err
	}

	return len(g.commits), nil
}

func (g *Project) GetTypeOfChanges() (string, error) {
	typeOfChanges, err := gitCommand("-C", g.Directory, "diff", "--shortstat", fmt.Sprintf("%s..%s", g.PreviousTag, g.CurrentTag))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(typeOfChanges), nil
}

func (g *Project) SwitchToTag(tag string) error {
	tagRef, err := g.repository.Tag(tag)
	if err != nil {
		return err
	}
	err = g.worktree.Checkout(&git.CheckoutOptions{Branch: tagRef.Name()})
	return err
}

func (g *Project) CheckCurrentTagExists() error {
	_, err := g.repository.Tag(g.CurrentTag)
	if err != nil {
		return err
	}

	return nil
}

func (g *Project) fetchCommits() error {
	// skip if commits field is already populated
	if g.commits != nil && len(g.commits) != 0 {
		return nil
	}

	fullLog, err := gitCommand("-C", g.Directory, "log", "--pretty=format:%H", fmt.Sprintf("%s..%s", g.PreviousTag, g.CurrentTag))
	if err != nil {
		return err
	}

	g.commits = make([]*object.Commit, 0)

	for _, l := range strings.Split(fullLog, "\n") {
		com, err := g.repository.CommitObject(plumbing.NewHash(l))
		if err != nil {
			return err
		}
		g.commits = append(g.commits, com)
	}

	return nil
}

func gitCommand(arg ...string) (string, error) {
	log.Printf("executing 'git %v", arg)
	cmd := exec.Command("git", arg...)
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ERROR: git command output: %s : %s ", string(bytes), err)
		return "", err
	}
	return string(bytes), nil
}

func (g *Project) VerifySemverTag() error {
	tagSemver, err := semver.NewVersion(g.CurrentTag)
	if err != nil {
		return err
	}

	g.PreviousTag, err = g.github.calculatePreviousRelease(tagSemver, g.CurrentTag)
	if err != nil {
		return err
	}

	if g.PreviousTag == "" {
		log.Printf("No previous release tag found for tag [%s]", g.CurrentTag)
	} else {
		log.Printf("Previous Tag [%s]", g.PreviousTag)
	}

	return nil
}
