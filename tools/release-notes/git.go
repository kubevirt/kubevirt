package main

import (
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func (p *project) gitCheckoutUpstream() error {
	_, err := os.Stat(p.repoDir)
	if err == nil {
		_, err := gitCommand("-C", p.repoDir, "status")
		if err == nil {
			// checkout already exists, updating
			return p.gitUpdateFromUpstream()
		}
	}

	return p.gitCloneUpstream()
}

func (p *project) gitUpdateFromUpstream() error {
	_, err := gitCommand("-C", p.repoDir, "checkout", "main")
	if err != nil {
		_, err = gitCommand("-C", p.repoDir, "checkout", "master")
		if err != nil {
			return err
		}
	}

	_, err = gitCommand("-C", p.repoDir, "pull")
	if err != nil {
		return err
	}
	return nil
}

func (p *project) gitCloneUpstream() error {
	// start fresh because checkout doesn't exist or is corrupted
	os.RemoveAll(p.repoDir)
	err := os.MkdirAll(p.repoDir, 0755)
	if err != nil {
		return err
	}

	// add upstream remote branch
	_, err = gitCommand("clone", p.repoUrl, p.repoDir)
	if err != nil {
		return err
	}

	_, err = gitCommand("-C", p.repoDir, "config", "diff.renameLimit", "999999")
	if err != nil {
		return err
	}

	return nil
}

func (p *project) gitGetContributors(span string) ([]string, error) {
	contributorStr, err := gitCommand("-C", p.repoDir, "shortlog", "-sne", span)
	if err != nil {
		return nil, err
	}

	return strings.Split(contributorStr, "\n"), nil
}

func (p *project) gitReleaseNoteFromSquashCommit(releaseNotes []string, matches []string) []string {
	for _, match := range matches {
		num, err := strconv.Atoi(match[2 : len(match)-1])
		if err != nil {
			continue
		}
		releaseNotes = p.gitReadReleaseNote(releaseNotes, num)
	}

	return releaseNotes
}

func (p *project) gitReleaseNoteFromMergeCommit(releaseNotes []string, line string) []string {
	pr := strings.Split(line, " ")

	num, err := strconv.Atoi(strings.TrimPrefix(pr[4], "#"))
	if err != nil {
		return releaseNotes
	}
	return p.gitReadReleaseNote(releaseNotes, num)
}

func (p *project) gitReadReleaseNote(releaseNotes []string, num int) []string {
	note, err := p.gitHubGetReleaseNote(num)
	if err != nil {
		return releaseNotes
	}
	if note != "" {
		releaseNotes = append(releaseNotes, note)
	}

	return releaseNotes
}

func (p *project) gitGetReleaseNotes(span string) ([]string, error) {
	fullLogStr, err := gitCommand("-C", p.repoDir, "log", "--oneline", span)
	if err != nil {
		return nil, err
	}

	var releaseNotes []string

	fullLogLines := strings.Split(fullLogStr, "\n")
	pattern := regexp.MustCompile(`\(#\d+\)`)

	for _, line := range fullLogLines {
		matches := pattern.FindAllString(line, -1)

		if len(matches) > 0 {
			releaseNotes = p.gitReleaseNoteFromSquashCommit(releaseNotes, matches)
		} else if strings.Contains(line, "Merge pull request #") {
			releaseNotes = p.gitReleaseNoteFromMergeCommit(releaseNotes, line)
		}
	}

	return releaseNotes, nil
}

func (p *project) gitGetNumChanges(span string) (int, error) {
	logStr, err := gitCommand("-C", p.repoDir, "log", "--oneline", span)
	if err != nil {
		return -1, err
	}

	return strings.Count(logStr, "\n"), nil
}

func (p *project) gitGetTypeOfChanges(span string) (string, error) {
	typeOfChanges, err := gitCommand("-C", p.repoDir, "diff", "--shortstat", span)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(typeOfChanges), nil
}

func (p *project) switchToBranch(branch string) error {
	_, err := gitCommand("-C", p.repoDir, "checkout", branch)
	if err != nil {
		return err
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
