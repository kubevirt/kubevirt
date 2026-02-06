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
	release, _, err := DetectLatestYAndZOfficialTags()
	return release, err
}

func DetectLatestYAndZOfficialTags() (string, string, error) {
	client := github.NewClient(&http.Client{
		Timeout: 5 * time.Second,
	})

	targetTag, err := getTagHint()
	if err != nil {
		return "", "", err
	}

	// Fetch all releases
	releases, _, err := client.Repositories.ListReleases(context.Background(), "kubevirt", "kubevirt", &github.ListOptions{PerPage: 100})
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch releases: %w", err)
	}

	return detectLatestYAndZReleases(targetTag, releases)
}

// detectLatestYAndZReleases finds both the latest Y release (previous minor version) and Z release (previous patch version)
// Returns (latestY, latestZ, error). If no Z release exists, latestZ will be empty string.
// For example, if targetTag is v1.3.2:
//   - latestY would be the latest v1.2.x release
//   - latestZ would be the latest v1.3.x release where x < 2
func detectLatestYAndZReleases(targetTag string, releases []*github.RepositoryRelease) (string, string, error) {
	// Parse target version
	targetVersionStr := strings.TrimPrefix(targetTag, "v")
	targetVersion, err := semver.NewVersion(targetVersionStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid target tag: %w", err)
	}

	var previousMinorReleases []*semver.Version
	var previousPatchReleases []*semver.Version
	var sameMinorReleases []*semver.Version

	isPreRelease := targetVersion.PreRelease != ""

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

		// If the targetVersion is preRelease (alpha/beta), collect all releases from same minor version
		if isPreRelease && v.Major == targetVersion.Major && v.Minor == targetVersion.Minor {
			sameMinorReleases = append(sameMinorReleases, v)
		}

		// Collect Y releases: previous minor version
		// Same major version, minor version is exactly 1 less
		if (v.Major == targetVersion.Major && v.Minor == targetVersion.Minor-1) || v.Major < targetVersion.Major {
			previousMinorReleases = append(previousMinorReleases, v)
		}

		// Collect Z releases: same minor version, lower patch
		if v.Major == targetVersion.Major && v.Minor == targetVersion.Minor && v.Patch < targetVersion.Patch {
			previousPatchReleases = append(previousPatchReleases, v)
		}
	}

	// If targetVersion is preRelease, and we found releases from the same minor version,
	// return the latest one for both Y and Z
	if isPreRelease && len(sameMinorReleases) > 0 {
		sort.Sort(semver.Versions(sameMinorReleases))
		latestTag := fmt.Sprintf("v%s", sameMinorReleases[len(sameMinorReleases)-1].String())
		By(fmt.Sprintf("Choosing Y and Z release tag %s for prerelease %s", latestTag, targetTag))
		return latestTag, latestTag, nil
	}

	if len(previousMinorReleases) == 0 {
		return "", "", fmt.Errorf("no previous minor release found for %s", targetTag)
	}

	// Sort by version and get the latest (last one)
	sort.Sort(semver.Versions(previousMinorReleases))
	latestYTag := fmt.Sprintf("v%s", previousMinorReleases[len(previousMinorReleases)-1].String())

	latestZTag := ""
	if len(previousPatchReleases) > 0 {
		sort.Sort(semver.Versions(previousPatchReleases))
		latestZTag = fmt.Sprintf("v%s", previousPatchReleases[len(previousPatchReleases)-1].String())
		By(fmt.Sprintf("Choosing Y release tag %s and Z release tag %s influenced by tag hint %s", latestYTag, latestZTag, targetTag))
	} else {
		By(fmt.Sprintf("Choosing Y release tag %s (no Z release found) influenced by tag hint %s", latestYTag, targetTag))
	}

	return latestYTag, latestZTag, nil
}

// detectLatestYRelease finds the latest release from the previous minor version (Y release)
// For example, if targetTag is v1.3.0, this will find the latest v1.2.x release
func detectLatestYRelease(targetTag string, releases []*github.RepositoryRelease) (string, error) {
	yRelease, _, err := detectLatestYAndZReleases(targetTag, releases)
	return yRelease, err
}

// detectLatestZRelease finds the latest release from the same minor version but previous patch (Z release)
// For example, if targetTag is v1.3.2, this will find the latest v1.3.x release where x < 2
func detectLatestZRelease(targetTag string, releases []*github.RepositoryRelease) (string, error) {
	_, zRelease, err := detectLatestYAndZReleases(targetTag, releases)
	if err != nil {
		return "", err
	}
	if zRelease == "" {
		return "", fmt.Errorf("no previous patch release found for %s", targetTag)
	}
	return zRelease, nil
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
