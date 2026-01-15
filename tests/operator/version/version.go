/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package version

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/google/go-github/v32/github"
	. "github.com/onsi/ginkgo/v2"
)

func DetectLatestUpstreamOfficialTag() (string, error) {
	client := github.NewClient(&http.Client{
		Timeout: 5 * time.Second,
	})

	targetTag, err := getTagHint()
	if err != nil {
		return "", err
	}

	// Fetch all releases
	releases, _, err := client.Repositories.ListReleases(context.Background(), "kubevirt", "kubevirt", &github.ListOptions{PerPage: 100})
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}

	return detectLatestUpstreamOfficialTagFromReleases(targetTag, releases)
}

// detectLatestUpstreamOfficialTagFromReleases is a testable helper function that finds the previous Y release
func detectLatestUpstreamOfficialTagFromReleases(targetTag string, releases []*github.RepositoryRelease) (string, error) {
	// Parse target version
	targetVersionStr := strings.TrimPrefix(targetTag, "v")
	targetVersion, err := semver.NewVersion(targetVersionStr)
	if err != nil {
		return "", fmt.Errorf("invalid target tag: %w", err)
	}

	var previousMinorReleases []*semver.Version
	for _, release := range releases {
		if *release.Draft ||
			*release.Prerelease ||
			len(release.Assets) == 0 {

			continue
		}

		tagName := release.GetTagName()
		if tagName == "" {
			continue
		}

		versionStr := strings.TrimPrefix(tagName, "v")
		v, err := semver.NewVersion(versionStr)
		if err != nil {
			continue
		}

		// If the targetVersion is preRelease (alpha/beta) then use the targetVersion because it is
		// the previous version.
		if targetVersion.PreRelease != "" && v.Major == targetVersion.Major && v.Minor == targetVersion.Minor {
			return tagName, nil
		}

		// Only include releases from the previous minor version
		// Same major version, minor version is exactly 1 less
		if (v.Major == targetVersion.Major && v.Minor == targetVersion.Minor-1) || v.Major < targetVersion.Major {
			previousMinorReleases = append(previousMinorReleases, v)
		}
	}

	if len(previousMinorReleases) == 0 {
		return "", fmt.Errorf("no previous minor release found for %s", targetTag)
	}

	// Sort by version and get the latest (last one)
	sort.Sort(semver.Versions(previousMinorReleases))

	latestTag := fmt.Sprintf("v%s", previousMinorReleases[len(previousMinorReleases)-1].String())
	By(fmt.Sprintf("Choosing tag %s influenced by tag hint %s", latestTag, targetTag))
	return latestTag, nil
}

func getTagHint() (string, error) {
	//git describe --tags --abbrev=0 "$(git rev-parse HEAD)"
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmdOutput, err := cmd.Output()
	if err != nil {
		return "", err
	}

	cmd = exec.Command("git", "describe", "--tags", "--abbrev=0", strings.TrimSpace(string(cmdOutput)))
	cmdOutput, err = cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.Split(string(cmdOutput), "-rc")[0]), nil
}
