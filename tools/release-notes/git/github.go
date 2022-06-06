package git

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/Masterminds/semver"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

type gh struct {
	githubClient *github.Client

	owner string
	name  string

	releases []*github.RepositoryRelease
}

func initGitHub(owner, name, token string) *gh {
	return &gh{
		githubClient: initClient(token),
		owner:        owner,
		name:         name,
	}
}

func initClient(token string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}

func (g *gh) getReleaseNote(number int) (string, error) {
	log.Printf("Searching for `%s` release note for PR #%d", g.name, number)
	pr, _, err := g.githubClient.PullRequests.Get(context.Background(), "kubevirt", g.name, number)
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
		note, err := parseReleaseNote(i, line, body)
		if err == nil {
			note = fmt.Sprintf("[PR #%d][%s] %s", number, *pr.User.Login, note)
			return note, nil
		}
	}

	return "", nil
}

func parseReleaseNote(index int, line string, body []string) (string, error) {
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

func (g *gh) calculatePreviousRelease(tagSemver *semver.Version, currentTag string) (string, error) {
	releases, err := g.getReleases()
	if err != nil {
		return "", err
	}

	for _, release := range releases {
		if *release.TagName == currentTag {
			log.Printf("WARNING: Release tag [%s] already exists", currentTag)
		}
	}

	vs := getVersions(releases)
	for _, v := range vs {
		if v.LessThan(tagSemver) {
			return fmt.Sprintf("v%v", v), nil
		}
	}

	return "", errors.New(fmt.Sprintf("could not find a release that preceds %s", currentTag))
}

func (g *gh) getReleases() ([]*github.RepositoryRelease, error) {
	if len(g.releases) != 0 {
		return g.releases, nil
	}

	releases, _, err := g.githubClient.Repositories.ListReleases(context.Background(), g.owner, g.name, &github.ListOptions{PerPage: 10000})

	if err != nil {
		return nil, err
	}
	g.releases = releases

	return g.releases, nil
}

func getVersions(releases []*github.RepositoryRelease) []*semver.Version {
	var vs []*semver.Version

	for _, release := range releases {
		if (release.Draft != nil && *release.Draft) ||
			(release.Prerelease != nil && *release.Prerelease) {

			continue
		}
		v, err := semver.NewVersion(*release.TagName)
		if err != nil {
			// not an official release if it's not semver compatible.
			continue
		}
		vs = append(vs, v)
	}

	// descending order from most recent.
	sort.Sort(sort.Reverse(semver.Collection(vs)))

	return vs
}
