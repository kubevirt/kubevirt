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

package cloudinit_test

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/libvmi"
	cloudinitdata "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
)

var _ = Describe("cloudinit.ReadCloudInitVolumeDataSource", func() {
	var root string
	BeforeEach(func() { root = GinkgoT().TempDir() })
	It("returns nil when the VMI has no cloud-init volume", func() {
		vmi := libvmi.New()
		data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(BeNil())
	})
	Context("NoCloud", func() {
		It("reads raw user and network payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithNoCloudNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.DataSource).To(Equal(cloudinit.DataSourceNoCloud))
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("decodes base64 user and network payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudEncodedUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithNoCloudEncodedNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("prefers raw payloads to malformed base64", func() {
			// Preserve existing raw-data precedence during this refactor.
			// Returning an error for malformed base64 would be more prudent
			// than silently ignoring it when raw data is also present.
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithNoCloudNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			volume := libvmi.GetCloudInitVolume(vmi).CloudInitNoCloud
			volume.UserDataBase64 = "!"
			volume.NetworkDataBase64 = "!"
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("accepts network-only source data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(BeEmpty())
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("rejects empty source data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("userDataBase64, userData, networkDataBase64 or networkData is required for a cloud-init data source"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("rejects malformed user base64", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			libvmi.GetCloudInitVolume(vmi).CloudInitNoCloud.UserDataBase64 = "!"
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("illegal base64 data at input byte 0"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("rejects malformed network base64", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)
			libvmi.GetCloudInitVolume(vmi).CloudInitNoCloud.NetworkDataBase64 = "!"
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("illegal base64 data at input byte 0"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("uses userData when userdata cannot be read", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/userData", "secret-user")

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("secret-user"))
		})
		It("prefers lowercase networkdata when both network filenames contain data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudNetworkDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/networkdata", "primary-network")
			writeFixture(root, "cloudinitdisk/networkData", "secondary-network")

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NetworkData).To(Equal("primary-network"))
		})
		It("keeps inline data when the only requested secret is missing", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("inline"),
					cloudinitdata.WithNoCloudUserDataSecretName("missing-user-secret"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
		})
		It("rejects two missing secret payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("inline"),
					cloudinitdata.WithNoCloudUserDataSecretName("missing-user-secret"),
					cloudinitdata.WithNoCloudNetworkDataSecretName("missing-network-secret"),
				),
			)
			_, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("no cloud-init data-source found at volume: cloudinitdisk"))
		})
		It("accepts a network secret when the user secret is missing", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("inline"),
					cloudinitdata.WithNoCloudUserDataSecretName("missing-user-secret"),
					cloudinitdata.WithNoCloudNetworkDataSecretName("network-secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/networkdata", "fake\nnetwork\ndata\n")
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("prefers lowercase cloudinitdisk/userdata even if empty", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("inline"),
					cloudinitdata.WithNoCloudUserDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/userdata", "")
			writeFixture(root, "cloudinitdisk/userData", "ignored")
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
		})
		It("orders selected public keys and ignores projected bookkeeping", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
				withNoCloudSSHPublicKey("first"),
				withConfigDriveSSHPublicKey("ignored"),
				withAccessCredential(v1.AccessCredential{}),
				withNoCloudSSHPublicKey(""),
				withNoCloudSSHPublicKey("second"),
			)

			writeFixture(root, "first-access-cred/z", "last")
			writeFixture(root, "first-access-cred/a", "first")
			writeFixture(root, "first-access-cred/empty", "")
			writeFixture(root, "second-access-cred/..version/key", "second")
			symlinkFixture(root, "second-access-cred/..data", "..version")
			symlinkFixture(root, "second-access-cred/link", "..data/key")
			symlinkFixture(root, "first-access-cred/..data", "missing")
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.PublicSSHKeys).To(Equal(map[string]string{"0": "first", "1": "last", "2": "second"}))
		})
		It("returns a credential directory read error", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
				withNoCloudSSHPublicKey("keys"),
			)

			writeFixture(root, "keys-access-cred", "regular file")
			_, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(filepath.Join(root, "keys-access-cred")))
		})
		It("reports the path of a credential file read error", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
				withNoCloudSSHPublicKey("keys"),
			)

			Expect(os.Mkdir(filepath.Join(root, "keys-access-cred"), 0700)).To(Succeed())
			symlinkFixture(root, "keys-access-cred/public", "missing")
			_, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to read public keys found at volume: " + filepath.Join(root, "keys-access-cred/public")))
		})
	})
	Context("ConfigDrive", func() {
		It("reads raw user and network payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithConfigDriveNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.DataSource).To(Equal(cloudinit.DataSourceConfigDrive))
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("decodes base64 user and network payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveEncodedUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithConfigDriveEncodedNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("prefers raw payloads to malformed base64", func() {
			// Preserve existing raw-data precedence during this refactor.
			// Returning an error for malformed base64 would be more prudent
			// than silently ignoring it when raw data is also present.
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithConfigDriveNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			volume := libvmi.GetCloudInitVolume(vmi).CloudInitConfigDrive
			volume.UserDataBase64 = "!"
			volume.NetworkDataBase64 = "!"
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("accepts network-only source data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(BeEmpty())
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("rejects empty source data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("userDataBase64, userData, networkDataBase64 or networkData is required for a cloud-init data source"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("rejects malformed user base64", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			libvmi.GetCloudInitVolume(vmi).CloudInitConfigDrive.UserDataBase64 = "!"
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("illegal base64 data at input byte 0"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("rejects malformed network base64", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)
			libvmi.GetCloudInitVolume(vmi).CloudInitConfigDrive.NetworkDataBase64 = "!"
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("illegal base64 data at input byte 0"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("uses userData when userdata cannot be read", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/userData", "secret-user")

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("secret-user"))
		})
		It("prefers lowercase networkdata when both network filenames contain data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveNetworkDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/networkdata", "primary-network")
			writeFixture(root, "cloudinitdisk/networkData", "secondary-network")

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NetworkData).To(Equal("primary-network"))
		})
		It("keeps inline data when the only requested secret is missing", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("inline"),
					cloudinitdata.WithConfigDriveUserDataSecretName("missing-user-secret"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
		})
		It("rejects two missing secret payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("inline"),
					cloudinitdata.WithConfigDriveUserDataSecretName("missing-user-secret"),
					cloudinitdata.WithConfigDriveNetworkDataSecretName("missing-network-secret"),
				),
			)
			_, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("no cloud-init data-source found at volume: cloudinitdisk"))
		})
		It("accepts a network secret when the user secret is missing", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("inline"),
					cloudinitdata.WithConfigDriveUserDataSecretName("missing-user-secret"),
					cloudinitdata.WithConfigDriveNetworkDataSecretName("network-secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/networkdata", "fake\nnetwork\ndata\n")
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("prefers lowercase cloudinitdisk/userdata even if empty", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("inline"),
					cloudinitdata.WithConfigDriveUserDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/userdata", "")
			writeFixture(root, "cloudinitdisk/userData", "ignored")
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
		})
		It("orders selected public keys and ignores projected bookkeeping", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
				withConfigDriveSSHPublicKey("first"),
				withNoCloudSSHPublicKey("ignored"),
				withAccessCredential(v1.AccessCredential{}),
				withConfigDriveSSHPublicKey(""),
				withConfigDriveSSHPublicKey("second"),
			)

			writeFixture(root, "first-access-cred/z", "last")
			writeFixture(root, "first-access-cred/a", "first")
			writeFixture(root, "first-access-cred/empty", "")
			writeFixture(root, "second-access-cred/..version/key", "second")
			symlinkFixture(root, "second-access-cred/..data", "..version")
			symlinkFixture(root, "second-access-cred/link", "..data/key")
			symlinkFixture(root, "first-access-cred/..data", "missing")
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.PublicSSHKeys).To(Equal(map[string]string{"0": "first", "1": "last", "2": "second"}))
		})
		It("returns a credential directory read error", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
				withConfigDriveSSHPublicKey("keys"),
			)

			writeFixture(root, "keys-access-cred", "regular file")
			_, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(filepath.Join(root, "keys-access-cred")))
		})
		It("reports the path of a credential file read error", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
				withConfigDriveSSHPublicKey("keys"),
			)

			Expect(os.Mkdir(filepath.Join(root, "keys-access-cred"), 0700)).To(Succeed())
			symlinkFixture(root, "keys-access-cred/public", "missing")
			_, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to read public keys found at volume: " + filepath.Join(root, "keys-access-cred/public")))
		})
	})
	Context("NoCloud metadata", func() {
		It("uses the firmware UUID in metadata", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.InstanceID).To(Equal("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"))
		})
		It("prefers the cluster instancetype annotation to the namespaced annotation", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithAnnotation(v1.InstancetypeAnnotation, "namespaced"),
				libvmi.WithAnnotation(v1.ClusterInstancetypeAnnotation, "cluster"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.InstanceType).To(Equal("cluster"))
		})
		It("uses the namespaced instancetype when no cluster annotation exists", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithAnnotation(v1.InstancetypeAnnotation, "namespaced"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.InstanceType).To(Equal("namespaced"))
		})
		It("preserves an explicitly empty firmware UUID when reading", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID(""),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.InstanceID).To(BeEmpty())
		})
		It("uses the VMI name without its domain suffix as the metadata hostname", func() {
			vmi := libvmi.New(
				libvmi.WithName("guest.example.com"),
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.LocalHostname).To(Equal("guest"))
		})
		It("uses the VMI name truncated to 63 characters as the metadata hostname", func() {
			// Preserve existing silent truncation during this refactor. Returning
			// an error would be more prudent than discarding the hostname's tail.
			vmi := libvmi.New(
				libvmi.WithName(strings.Repeat("a", 70)),
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.LocalHostname).To(Equal(strings.Repeat("a", 63)))
		})
		It("honors an explicitly configured hostname", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)
			vmi.Spec.Hostname = "guest-host"
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.LocalHostname).To(Equal("guest-host"))
		})
	})
	Context("ConfigDrive metadata", func() {
		It("uses the firmware UUID in metadata", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.UUID).To(Equal("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"))
		})
		It("prefers the cluster instancetype annotation to the namespaced annotation", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithAnnotation(v1.InstancetypeAnnotation, "namespaced"),
				libvmi.WithAnnotation(v1.ClusterInstancetypeAnnotation, "cluster"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.InstanceType).To(Equal("cluster"))
		})
		It("uses the namespaced instancetype when no cluster annotation exists", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithAnnotation(v1.InstancetypeAnnotation, "namespaced"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.InstanceType).To(Equal("namespaced"))
		})
		It("preserves an explicitly empty firmware UUID when reading", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID(""),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.UUID).To(BeEmpty())
		})
		It("uses the VMI name without its domain suffix as the metadata hostname", func() {
			vmi := libvmi.New(
				libvmi.WithName("guest.example.com"),
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.Hostname).To(Equal("guest"))
		})
		It("uses the VMI name truncated to 63 characters as the metadata hostname", func() {
			// Preserve existing silent truncation during this refactor. Returning
			// an error would be more prudent than discarding the hostname's tail.
			vmi := libvmi.New(
				libvmi.WithName(strings.Repeat("a", 70)),
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.Hostname).To(Equal(strings.Repeat("a", 63)))
		})
		It("honors an explicitly configured hostname", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)
			vmi.Spec.Hostname = "guest-host"
			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.Hostname).To(Equal("guest-host"))
		})
	})
	It("returns data from the first cloud-init volume", func() {
		vmi := libvmi.New(
			libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
			libvmi.WithCloudInitNoCloud(cloudinitdata.WithNoCloudUserData("first")),
		)
		second := libvmi.New(libvmi.WithCloudInitNoCloud(
			cloudinitdata.WithNoCloudUserData("second"),
			cloudinitdata.WithNoCloudUserDataSecretName("secret"),
		))
		secondVolume := *libvmi.GetCloudInitVolume(second)
		secondVolume.Name = "second"
		third := libvmi.New(libvmi.WithCloudInitConfigDrive(
			cloudinitdata.WithConfigDriveUserData("third"),
		))
		thirdVolume := *libvmi.GetCloudInitVolume(third)
		thirdVolume.Name = "third"
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, secondVolume, thirdVolume)
		writeFixture(root, "second/userdata", "resolved")
		data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)
		Expect(err).NotTo(HaveOccurred())
		Expect(data.UserData).To(Equal("first"))
		Expect(data.DataSource).To(Equal(cloudinit.DataSourceNoCloud))
	})

})

var _ = Describe("BuildIsoPlan", func() {
	Context("NoCloud", func() {
		It("builds the exact ISO layout and serialized metadata", func() {
			data := &cloudinit.CloudInitData{
				DataSource:  cloudinit.DataSourceNoCloud,
				UserData:    "fake\nuser\ndata\n",
				NetworkData: "fake\nnetwork\ndata\n",
				NoCloudMetaData: &cloudinit.NoCloudMetadata{
					InstanceID: "cf73b4f4-6975-4d93-b51c-b7ae392bdea5", LocalHostname: "vm",
				},
			}
			plan, err := cloudinit.BuildIsoPlan(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(plan).To(Equal(&cloudinit.IsoPlan{
				VolumeID: "cidata",
				Files: []cloudinit.IsoFile{
					{Path: "user-data", Data: []byte("fake\nuser\ndata\n")},
					{Path: "meta-data", Data: []byte(`{"instance-id":"cf73b4f4-6975-4d93-b51c-b7ae392bdea5","local-hostname":"vm"}`)},
					{Path: "network-config", Data: []byte("fake\nnetwork\ndata\n")},
				},
			}))
		})

		It("omits the network file when no network data is supplied", func() {
			data := &cloudinit.CloudInitData{
				DataSource: cloudinit.DataSourceNoCloud,
				UserData:   "fake\nuser\ndata\n",
				NoCloudMetaData: &cloudinit.NoCloudMetadata{
					InstanceID: "cf73b4f4-6975-4d93-b51c-b7ae392bdea5", LocalHostname: "vm",
				},
			}
			plan, err := cloudinit.BuildIsoPlan(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.Files).To(Equal([]cloudinit.IsoFile{
				{Path: "user-data", Data: []byte("fake\nuser\ndata\n")},
				{Path: "meta-data", Data: []byte(`{"instance-id":"cf73b4f4-6975-4d93-b51c-b7ae392bdea5","local-hostname":"vm"}`)},
			}))
		})

		It("includes an empty user-data file for network-only input", func() {
			data := &cloudinit.CloudInitData{DataSource: cloudinit.DataSourceNoCloud, NetworkData: "fake\nnetwork\ndata\n", NoCloudMetaData: &cloudinit.NoCloudMetadata{InstanceID: "cf73b4f4-6975-4d93-b51c-b7ae392bdea5"}}
			plan, err := cloudinit.BuildIsoPlan(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.Files[0]).To(Equal(cloudinit.IsoFile{Path: "user-data", Data: []byte{}}))
			Expect(plan.Files[2]).To(Equal(cloudinit.IsoFile{Path: "network-config", Data: []byte("fake\nnetwork\ndata\n")}))
		})

		It("rejects generation without user or network data", func() {
			data := &cloudinit.CloudInitData{DataSource: cloudinit.DataSourceNoCloud}
			_, err := cloudinit.BuildIsoPlan(data)
			Expect(err).To(MatchError("UserData or NetworkData is required for cloud-init data source"))
		})

	})
	Context("ConfigDrive", func() {
		It("builds the exact ISO layout and serialized metadata", func() {
			data := &cloudinit.CloudInitData{
				DataSource:  cloudinit.DataSourceConfigDrive,
				UserData:    "fake\nuser\ndata\n",
				NetworkData: "fake\nnetwork\ndata\n",
				ConfigDriveMetaData: &cloudinit.ConfigDriveMetadata{
					InstanceID: "vm.ns", Hostname: "vm", UUID: "cf73b4f4-6975-4d93-b51c-b7ae392bdea5",
				},
			}
			plan, err := cloudinit.BuildIsoPlan(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(plan).To(Equal(&cloudinit.IsoPlan{
				VolumeID: "config-2",
				Files: []cloudinit.IsoFile{
					{Path: "openstack/latest/user_data", Data: []byte("fake\nuser\ndata\n")},
					{Path: "openstack/latest/meta_data.json", Data: []byte(`{"instance_id":"vm.ns","hostname":"vm","uuid":"cf73b4f4-6975-4d93-b51c-b7ae392bdea5"}`)},
					{Path: "openstack/latest/network_data.json", Data: []byte("fake\nnetwork\ndata\n")},
				},
			}))
		})

		It("omits the network file when no network data is supplied", func() {
			data := &cloudinit.CloudInitData{
				DataSource: cloudinit.DataSourceConfigDrive,
				UserData:   "fake\nuser\ndata\n",
				ConfigDriveMetaData: &cloudinit.ConfigDriveMetadata{
					InstanceID: "vm.ns", Hostname: "vm", UUID: "cf73b4f4-6975-4d93-b51c-b7ae392bdea5",
				},
			}
			plan, err := cloudinit.BuildIsoPlan(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.Files).To(Equal([]cloudinit.IsoFile{
				{Path: "openstack/latest/user_data", Data: []byte("fake\nuser\ndata\n")},
				{Path: "openstack/latest/meta_data.json", Data: []byte(`{"instance_id":"vm.ns","hostname":"vm","uuid":"cf73b4f4-6975-4d93-b51c-b7ae392bdea5"}`)},
			}))
		})

		It("includes an empty user_data file for network-only input", func() {
			data := &cloudinit.CloudInitData{DataSource: cloudinit.DataSourceConfigDrive, NetworkData: "fake\nnetwork\ndata\n", ConfigDriveMetaData: &cloudinit.ConfigDriveMetadata{InstanceID: "vm.ns"}}
			plan, err := cloudinit.BuildIsoPlan(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.Files[0]).To(Equal(cloudinit.IsoFile{Path: "openstack/latest/user_data", Data: []byte{}}))
			Expect(plan.Files[2]).To(Equal(cloudinit.IsoFile{Path: "openstack/latest/network_data.json", Data: []byte("fake\nnetwork\ndata\n")}))
		})

		It("rejects generation without user or network data", func() {
			data := &cloudinit.CloudInitData{DataSource: cloudinit.DataSourceConfigDrive}
			_, err := cloudinit.BuildIsoPlan(data)
			Expect(err).To(MatchError("UserData or NetworkData is required for cloud-init data source"))
		})

	})
	It("serializes NoCloud metadata", func() {
		data := &cloudinit.CloudInitData{
			DataSource: cloudinit.DataSourceNoCloud,
			UserData:   "fake\nuser\ndata\n",
			NoCloudMetaData: &cloudinit.NoCloudMetadata{
				InstanceID: "provided", InstanceType: "provided-type", LocalHostname: "host",
				PublicSSHKeys: map[string]string{"0": "key"},
			},
		}
		plan, err := cloudinit.BuildIsoPlan(data)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(plan.Files[1].Data)).To(MatchJSON(`{"instance-id":"provided","instance-type":"provided-type","local-hostname":"host","public-keys":{"0":"key"}}`))
	})

	It("serializes ConfigDrive metadata", func() {
		data := &cloudinit.CloudInitData{
			DataSource: cloudinit.DataSourceConfigDrive,
			UserData:   "fake\nuser\ndata\n",
			ConfigDriveMetaData: &cloudinit.ConfigDriveMetadata{
				InstanceID: "provided", InstanceType: "provided-type",
				UUID: "cf73b4f4-6975-4d93-b51c-b7ae392bdea5", LocalHostname: "local", Hostname: "host",
				PublicSSHKeys: map[string]string{"0": "key"},
			},
		}
		plan, err := cloudinit.BuildIsoPlan(data)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(plan.Files[1].Data)).To(MatchJSON(`{"instance_type":"provided-type","instance_id":"provided","local_hostname":"local","hostname":"host","uuid":"cf73b4f4-6975-4d93-b51c-b7ae392bdea5","public_keys":{"0":"key"}}`))
	})

	It("includes ConfigDrive devices metadata", func() {
		devices := []cloudinit.DeviceData{
			{
				Type: cloudinit.HostDevMetadataType, Bus: "pci", Address: "0000:65:10:0",
				Serial: "serial", NumaNode: 1, AlignedCPUs: []uint32{0, 1}, Tags: []string{"tag"},
			},
			{
				Type: cloudinit.NICMetadataType, Bus: "pci", Address: "0000:01:00:0",
				MAC: "02:00:00:84:e9:58", Tags: []string{"tag"},
			},
		}
		data := &cloudinit.CloudInitData{
			DataSource:          cloudinit.DataSourceConfigDrive,
			UserData:            "fake\nuser\ndata\n",
			ConfigDriveMetaData: &cloudinit.ConfigDriveMetadata{InstanceID: "provided"},
			DevicesData:         &devices,
		}
		plan, err := cloudinit.BuildIsoPlan(data)
		Expect(err).NotTo(HaveOccurred())
		Expect(data.ConfigDriveMetaData.Devices).To(BeNil())
		Expect(string(plan.Files[1].Data)).To(MatchJSON(`{
			"instance_id": "provided",
			"devices": [
				{"type":"hostdev","bus":"pci","address":"0000:65:10:0","serial":"serial","numaNode":1,"alignedCPUs":[0,1],"tags":["tag"]},
				{"type":"nic","bus":"pci","address":"0000:01:00:0","mac":"02:00:00:84:e9:58","tags":["tag"]}
			]
		}`))
	})

	DescribeTable("preserves hook-generated nil metadata as JSON null",
		func(source cloudinit.DataSourceType, volumeID, userPath, metadataPath string) {
			data := &cloudinit.CloudInitData{
				DataSource: source,
				UserData:   "#cloud-config\n",
				// NoCloudMetaData and ConfigDriveMetaData are intentionally nil, as if broken by a hook.
			}

			plan, err := cloudinit.BuildIsoPlan(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(plan).To(Equal(&cloudinit.IsoPlan{
				VolumeID: volumeID,
				Files: []cloudinit.IsoFile{
					{Path: userPath, Data: []byte("#cloud-config\n")},
					{Path: metadataPath, Data: []byte("null")},
				},
			}))
			Expect(data.NoCloudMetaData).To(BeNil())
			Expect(data.ConfigDriveMetaData).To(BeNil())
		},
		Entry("NoCloud", cloudinit.DataSourceNoCloud,
			"cidata", "user-data", "meta-data"),
		Entry("ConfigDrive", cloudinit.DataSourceConfigDrive,
			"config-2", "openstack/latest/user_data",
			"openstack/latest/meta_data.json"),
	)

	It("rejects unsupported sources", func() {
		plan, err := cloudinit.BuildIsoPlan(&cloudinit.CloudInitData{DataSource: "unknown", UserData: "fake\nuser\ndata\n"})
		Expect(err).To(MatchError("Invalid cloud-init data source: 'unknown'"))
		Expect(plan).To(BeNil())
	})

})

func withAccessCredential(credential v1.AccessCredential) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.AccessCredentials = append(vmi.Spec.AccessCredentials, credential)
	}
}

func withNoCloudSSHPublicKey(secretName string) libvmi.Option {
	return withAccessCredential(v1.AccessCredential{
		SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
			Source: v1.SSHPublicKeyAccessCredentialSource{
				Secret: &v1.AccessCredentialSecretSource{SecretName: secretName},
			},
			PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
				NoCloud: &v1.NoCloudSSHPublicKeyAccessCredentialPropagation{},
			},
		},
	})
}

func withConfigDriveSSHPublicKey(secretName string) libvmi.Option {
	return withAccessCredential(v1.AccessCredential{
		SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
			Source: v1.SSHPublicKeyAccessCredentialSource{
				Secret: &v1.AccessCredentialSecretSource{SecretName: secretName},
			},
			PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
				ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
			},
		},
	})
}

func writeFixture(root, name, data string) {
	GinkgoHelper()
	path := filepath.Join(root, name)
	Expect(os.MkdirAll(filepath.Dir(path), 0700)).To(Succeed())
	Expect(os.WriteFile(path, []byte(data), 0600)).To(Succeed())
}

func symlinkFixture(root, name, target string) {
	GinkgoHelper()
	path := filepath.Join(root, name)
	Expect(os.MkdirAll(filepath.Dir(path), 0700)).To(Succeed())
	Expect(os.Symlink(target, path)).To(Succeed())
}

var _ = Describe("ReadCloudInitVolumeDataSource", func() {
	Context("NoCloud", func() {
		It("reads user and network data from mounted secrets", func() {
			root := GinkgoT().TempDir()
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserDataSecretName("user"),
					cloudinitdata.WithNoCloudNetworkDataSecretName("network"),
				),
			)
			writeFixture(root, "cloudinitdisk/..version/userdata", "secret-user")
			symlinkFixture(root, "cloudinitdisk/..data", "..version")
			symlinkFixture(root, "cloudinitdisk/userdata", "..data/userdata")
			writeFixture(root, "cloudinitdisk/networkdata", "secret-network")

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.DataSource).To(Equal(cloudinit.DataSourceNoCloud))
			Expect(data.UserData).To(Equal("secret-user"))
			Expect(data.NetworkData).To(Equal("secret-network"))
		})
	})

	Context("ConfigDrive", func() {
		It("reads user and network data from mounted secrets", func() {
			root := GinkgoT().TempDir()
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserDataSecretName("user"),
					cloudinitdata.WithConfigDriveNetworkDataSecretName("network"),
				),
			)
			writeFixture(root, "cloudinitdisk/..version/userdata", "secret-user")
			symlinkFixture(root, "cloudinitdisk/..data", "..version")
			symlinkFixture(root, "cloudinitdisk/userdata", "..data/userdata")
			writeFixture(root, "cloudinitdisk/networkdata", "secret-network")

			data, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.DataSource).To(Equal(cloudinit.DataSourceConfigDrive))
			Expect(data.UserData).To(Equal("secret-user"))
			Expect(data.NetworkData).To(Equal("secret-network"))
		})
	})
})

func setCloudInitDirectory() string {
	GinkgoHelper()
	root := GinkgoT().TempDir()
	previous := filepath.Dir(cloudinit.GetIsoFilePath(cloudinit.DataSourceNoCloud, "", ""))
	DeferCleanup(cloudinit.SetLocalDirectory, previous)
	Expect(cloudinit.SetLocalDirectory(root)).To(Succeed())
	return root
}

var _ = Describe("GenerateIso", func() {
	It("preserves the published ISO and cleans staging and input files when generation fails", func() {
		cache := setCloudInitDirectory()
		vmDir := filepath.Join(cache, "default", "guest")
		vmi := libvmi.New(libvmi.WithName("guest"), libvmi.WithNamespace("default"))
		data := &cloudinit.CloudInitData{
			DataSource: cloudinit.DataSourceNoCloud,
			UserData:   "fake\nuser\ndata\n",
		}
		writeFixture(vmDir, "noCloud.iso", "existing ISO")
		writeFixture(vmDir, "noCloud.iso.staging", "stale staging file")
		GinkgoT().Setenv("PATH", "no-iso-binary")

		Expect(cloudinit.GenerateIso(vmi, data)).NotTo(Succeed())

		content, err := os.ReadFile(filepath.Join(vmDir, "noCloud.iso"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("existing ISO"))
		entries, err := os.ReadDir(vmDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].Name()).To(Equal("noCloud.iso"))
	})
})
