package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
	"log"
	"strings"
)

func gitHubInitClient(token string) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	githubClient = github.NewClient(tc)
}

func (p *project) gitHubGetReleaseNote(number int) (string, error) {
	log.Printf("Searching for `%s` release note for PR #%d", p.name, number)
	pr, _, err := githubClient.PullRequests.Get(context.Background(), "kubevirt", p.name, number)
	if err != nil {
		return "", err
	}

	for _, label := range pr.Labels {
		if label.Name != nil && *label.Name == "release-note-none" {
			return "", nil
		}
	}

	if pr.Body == nil || *pr.Body == "" {
		return "", err
	}

	body := strings.Split(*pr.Body, "\n")

	for i, line := range body {
		note, err := gitHubParseReleaseNote(i, line, body)
		if err == nil {
			note = fmt.Sprintf("[PR #%d][%s] %s", number, *pr.User.Login, note)
			return note, nil
		}
	}

	return "", nil
}

func gitHubParseReleaseNote(index int, line string, body []string) (string, error) {
	if strings.Contains(line, "```release-note") {
		releaseNoteIndex := index + 1
		if len(body) > releaseNoteIndex {
			note := strings.TrimSpace(body[releaseNoteIndex])
			// best effort at fixing some format errors I find
			note = strings.ReplaceAll(note, "\r\n", "")
			note = strings.ReplaceAll(note, "\r", "")
			note = strings.TrimPrefix(note, "- ")
			note = strings.TrimPrefix(note, "-")

			// best effort at catching "none" if the label didn't catch it
			if !strings.Contains(note, "NONE") && strings.ToLower(note) != "none" {
				return note, nil
			}
		}
	}

	return "", fmt.Errorf("release note not found")
}

func (p *project) gitHubGetBranches() ([]*github.Branch, error) {
	if len(p.allBranches) != 0 {
		return p.allBranches, nil
	}

	branches, _, err := githubClient.Repositories.ListBranches(context.Background(), "kubevirt", p.name, &github.BranchListOptions{
		ListOptions: github.ListOptions{
			PerPage: 10000,
		},
	})
	if err != nil {
		return nil, err
	}
	p.allBranches = branches

	return p.allBranches, nil

}

func (p *project) gitHubGetReleases() ([]*github.RepositoryRelease, error) {
	if len(p.allReleases) != 0 {
		return p.allReleases, nil
	}

	releases, _, err := githubClient.Repositories.ListReleases(context.Background(), "kubevirt", p.name, &github.ListOptions{PerPage: 10000})

	if err != nil {
		return nil, err
	}
	p.allReleases = releases

	return p.allReleases, nil
}
