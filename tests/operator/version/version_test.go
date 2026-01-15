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
	"github.com/google/go-github/v32/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	isDraft, isPrerelease   = true, true
	notDraft, notPrerelease = false, false
)

// Helper function to create mock releases
func createMockRelease(tagName string, isDraft, isPrerelease bool, assetCount int) *github.RepositoryRelease {
	release := &github.RepositoryRelease{
		TagName:    pointer.P(tagName),
		Draft:      pointer.P(isDraft),
		Prerelease: pointer.P(isPrerelease),
	}

	// Add mock assets
	for range assetCount {
		release.Assets = append(release.Assets, &github.ReleaseAsset{})
	}

	return release
}

var _ = Describe("detectLatestUpstreamOfficialTagFromReleases", func() {

	Context("when target is a GA release on release branch", func() {
		It("should return latest patch from previous minor version (v1.8.0 -> v1.7.x)", func() {
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.7.3", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.2", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.1", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.0", notDraft, notPrerelease, 1),
				createMockRelease("v1.6.5", notDraft, notPrerelease, 1),
				createMockRelease("v1.6.4", notDraft, notPrerelease, 1),
			}

			result, err := detectLatestUpstreamOfficialTagFromReleases("v1.8.0", mockReleases)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("v1.7.3"))
		})
	})

	Context("when target is a prerelease (alpha/beta) on main branch", func() {
		It("should return latest GA from same minor version (v1.8.0-alpha.0 -> v1.8.x)", func() {
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.8.2", notDraft, notPrerelease, 1),
				createMockRelease("v1.8.1", notDraft, notPrerelease, 1),
				createMockRelease("v1.8.0", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.3", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.2", notDraft, notPrerelease, 1),
			}

			result, err := detectLatestUpstreamOfficialTagFromReleases("v1.8.0-alpha.0", mockReleases)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("v1.8.2"))
		})

		It("should return latest GA from same minor version (v1.7.0-beta.0 -> v1.7.x)", func() {
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.7.0", notDraft, notPrerelease, 1),
				createMockRelease("v1.6.3", notDraft, notPrerelease, 1),
				createMockRelease("v1.6.2", notDraft, notPrerelease, 1),
			}

			result, err := detectLatestUpstreamOfficialTagFromReleases("v1.7.0-beta.0", mockReleases)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("v1.7.0"))
		})
	})

	Context("when filtering releases", func() {
		It("should skip draft releases", func() {
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.7.3", isDraft, notPrerelease, 1), // draft
				createMockRelease("v1.7.2", notDraft, notPrerelease, 1),
				createMockRelease("v1.6.3", notDraft, notPrerelease, 1),
			}

			result, err := detectLatestUpstreamOfficialTagFromReleases("v1.8.0", mockReleases)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("v1.7.2"))
		})

		It("should skip prerelease versions", func() {
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.7.3-rc.1", notDraft, isPrerelease, 1), // prerelease
				createMockRelease("v1.7.2", notDraft, notPrerelease, 1),
				createMockRelease("v1.6.3", notDraft, notPrerelease, 1),
			}

			result, err := detectLatestUpstreamOfficialTagFromReleases("v1.8.0", mockReleases)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("v1.7.2"))
		})

		It("should skip releases without assets", func() {
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.7.3", notDraft, notPrerelease, 0), // no assets
				createMockRelease("v1.7.2", notDraft, notPrerelease, 1),
				createMockRelease("v1.6.3", notDraft, notPrerelease, 1),
			}

			result, err := detectLatestUpstreamOfficialTagFromReleases("v1.8.0", mockReleases)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("v1.7.2"))
		})
	})

	Context("when handling major version changes", func() {
		It("should return latest from previous major version if no previous minor exists", func() {
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.9.5", notDraft, notPrerelease, 1),
				createMockRelease("v1.9.4", notDraft, notPrerelease, 1),
			}

			result, err := detectLatestUpstreamOfficialTagFromReleases("v2.0.0", mockReleases)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("v1.9.5"))
		})

		It("should prefer previous minor over older major version", func() {
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v2.0.3", notDraft, notPrerelease, 1),
				createMockRelease("v2.0.2", notDraft, notPrerelease, 1),
				createMockRelease("v1.9.5", notDraft, notPrerelease, 1),
			}

			result, err := detectLatestUpstreamOfficialTagFromReleases("v2.1.0", mockReleases)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("v2.0.3"))
		})
	})

	Context("error cases", func() {
		It("should return error when target tag is invalid", func() {
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.7.0", notDraft, notPrerelease, 1),
			}

			_, err := detectLatestUpstreamOfficialTagFromReleases("invalid-tag", mockReleases)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid target tag"))
		})

		It("should return error when no previous minor release found", func() {
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.8.0", notDraft, notPrerelease, 1),
				createMockRelease("v1.9.0", notDraft, notPrerelease, 1),
			}

			_, err := detectLatestUpstreamOfficialTagFromReleases("v1.7.0", mockReleases)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no previous minor release found"))
		})

		It("should return error when no releases available", func() {
			mockReleases := []*github.RepositoryRelease{}

			_, err := detectLatestUpstreamOfficialTagFromReleases("v1.8.0", mockReleases)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no previous minor release found"))
		})
	})

	Context("sorting behavior", func() {
		It("should correctly sort and return highest patch version", func() {
			// Releases in unsorted order
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.7.1", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.15", notDraft, notPrerelease, 1), // highest
				createMockRelease("v1.7.9", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.2", notDraft, notPrerelease, 1),
			}

			result, err := detectLatestUpstreamOfficialTagFromReleases("v1.8.0", mockReleases)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("v1.7.15"))
		})
	})

	Context("realistic kubevirt release scenarios", func() {
		It("should handle the v1.7.0 GA window bug scenario", func() {
			// https://github.com/kubevirt/kubevirt/issues/16542
			// Scenario: main has v1.7.0-beta.0, but v1.7.0 GA was released on release-1.7
			// Before fix: would return v1.6.3
			// After fix: should return v1.7.0
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.7.0", notDraft, notPrerelease, 1), // Latest GA from release-1.7
				createMockRelease("v1.6.3", notDraft, notPrerelease, 1),
				createMockRelease("v1.6.2", notDraft, notPrerelease, 1),
				createMockRelease("v1.6.1", notDraft, notPrerelease, 1),
				createMockRelease("v1.6.0", notDraft, notPrerelease, 1),
			}

			result, err := detectLatestUpstreamOfficialTagFromReleases("v1.7.0-beta.0", mockReleases)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("v1.7.0"), "Should select v1.7.0 GA, not fall back to v1.6.3")
		})

		It("should handle multiple patches after GA release", func() {
			// Scenario: v1.8.0-alpha.0 on main, multiple v1.7.x patches released
			mockReleases := []*github.RepositoryRelease{
				createMockRelease("v1.7.5", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.4", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.3", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.2", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.1", notDraft, notPrerelease, 1),
				createMockRelease("v1.7.0", notDraft, notPrerelease, 1),
				createMockRelease("v1.6.3", notDraft, notPrerelease, 1),
			}

			result, err := detectLatestUpstreamOfficialTagFromReleases("v1.8.0-alpha.0", mockReleases)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("v1.7.5"))
		})
	})
})
