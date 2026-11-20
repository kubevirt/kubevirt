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

package virthandler

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestContainerFileLabelFromAttr(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "mcs categories",
			raw:  "system_u:system_r:container_t:s0:c715,c719\x00",
			want: "system_u:object_r:container_file_t:s0:c715,c719",
		},
		{
			name: "level without categories",
			raw:  "system_u:system_r:container_t:s0\n",
			want: "system_u:object_r:container_file_t:s0",
		},
		{
			name: "empty",
			raw:  "",
			want: "",
		},
		{
			name: "not a context",
			raw:  "not-a-label",
			want: "",
		},
		{
			name: "missing type",
			raw:  "system_u:system_r::s0",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := containerFileLabelFromAttr(tc.raw); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestContainerFileLabel(t *testing.T) {
	orig := readProcAttr
	t.Cleanup(func() { readProcAttr = orig })

	readProcAttr = func(int) ([]byte, error) {
		return nil, os.ErrNotExist
	}
	label, err := containerFileLabel(1)
	if err != nil || label != "" {
		t.Fatalf("missing attr: label %q err %v", label, err)
	}

	readProcAttr = func(int) ([]byte, error) {
		return []byte("not-a-label"), nil
	}
	if _, err := containerFileLabel(1); err == nil {
		t.Fatal("expected an error for an unrecognized context")
	}
}

func TestLauncherPathKeepsMountRoot(t *testing.T) {
	got := launcherPath("/proc/42/root", qemuCapsTargetDir)
	want := "/proc/42/root/var/run/kubevirt-private/libvirt/qemu/cache/capabilities"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCopyQEMUCapabilityFiles(t *testing.T) {
	source := t.TempDir()
	mount := t.TempDir()
	hashName := "3c76bc41d59c0c7314b1ae8e63f4f765d2cf16abaeea081b3ca1f5d8732f7bb1.xml"
	body := []byte("<qemuCaps>cached</qemuCaps>")
	if err := os.WriteFile(filepath.Join(source, hashName), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "notes.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}

	label := "system_u:object_r:container_file_t:s0:c715,c719"
	var labeled []string
	restore := stubRelabel(t, func(path, gotLabel string) error {
		if gotLabel != label {
			t.Errorf("relabel label %q", gotLabel)
		}
		labeled = append(labeled, path)
		return nil
	})
	defer restore()

	copied, err := copyQEMUCapabilityFiles(source, mount, label)
	if err != nil {
		t.Fatal(err)
	}
	if copied != 1 {
		t.Fatalf("copied %d", copied)
	}
	dst := filepath.Join(mount, "var/run/kubevirt-private/libvirt/qemu/cache/capabilities", hashName)
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("body %q", got)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(mount, "var/run/kubevirt-private/libvirt/qemu/cache/capabilities/notes.txt")); !os.IsNotExist(err) {
		t.Fatal("non-xml file was copied")
	}
	if len(labeled) != len(qemuCapsLabelPaths)+1 {
		t.Fatalf("relabel calls %d, paths %v", len(labeled), labeled)
	}
	if labeled[len(labeled)-1] != dst {
		t.Fatalf("last relabel %q", labeled[len(labeled)-1])
	}

	labeled = nil
	copied, err = copyQEMUCapabilityFiles(source, mount, label)
	if err != nil {
		t.Fatal(err)
	}
	if copied != 0 || len(labeled) != 0 {
		t.Fatalf("second copy copied %d relabeled %d", copied, len(labeled))
	}
}

func TestCopyQEMUCapabilityFilesLeavesExistingContents(t *testing.T) {
	source := t.TempDir()
	mount := t.TempDir()
	name := "caps.xml"
	if err := os.WriteFile(filepath.Join(source, name), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	dstDir := filepath.Join(mount, "var/run/kubevirt-private/libvirt/qemu/cache/capabilities")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dstDir, name)
	if err := os.WriteFile(dst, []byte("already"), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := stubRelabel(t, func(string, string) error {
		t.Fatal("relabel should not run when the cache is already present")
		return nil
	})
	defer restore()

	copied, err := copyQEMUCapabilityFiles(source, mount, "system_u:object_r:container_file_t:s0")
	if err != nil {
		t.Fatal(err)
	}
	if copied != 0 {
		t.Fatalf("copied %d", copied)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "already" {
		t.Fatalf("existing cache was replaced with %q", got)
	}
}

func TestCopyQEMUCapabilityFilesRemovesFileWhenRelabelFails(t *testing.T) {
	source := t.TempDir()
	mount := t.TempDir()
	name := "caps.xml"
	if err := os.WriteFile(filepath.Join(source, name), []byte("cached"), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := stubRelabel(t, func(path, _ string) error {
		if filepath.Base(path) == name {
			return errors.New("chcon failed")
		}
		return nil
	})
	defer restore()

	copied, err := copyQEMUCapabilityFiles(source, mount, "system_u:object_r:container_file_t:s0:c1,c2")
	if err == nil {
		t.Fatal("expected relabel error")
	}
	if copied != 0 {
		t.Fatalf("copied %d", copied)
	}
	dst := filepath.Join(mount, "var/run/kubevirt-private/libvirt/qemu/cache/capabilities", name)
	if _, statErr := os.Stat(dst); !os.IsNotExist(statErr) {
		t.Fatalf("unlabeled cache file was left behind: %v", statErr)
	}
}

func TestCopyQEMUCapabilityFilesWithoutLabel(t *testing.T) {
	source := t.TempDir()
	mount := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "caps.xml"), []byte("cached"), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := stubRelabel(t, func(string, string) error {
		t.Fatal("relabel should not run without a label")
		return nil
	})
	defer restore()

	copied, err := copyQEMUCapabilityFiles(source, mount, "")
	if err != nil {
		t.Fatal(err)
	}
	if copied != 1 {
		t.Fatalf("copied %d", copied)
	}
}

func TestCopyQEMUCapabilityFilesMissingSource(t *testing.T) {
	copied, err := copyQEMUCapabilityFiles(filepath.Join(t.TempDir(), "missing"), t.TempDir(), "")
	if err != nil || copied != 0 {
		t.Fatalf("copied %d err %v", copied, err)
	}
}

func stubRelabel(t *testing.T, fn func(path, label string) error) func() {
	t.Helper()
	orig := relabelPath
	relabelPath = fn
	return func() { relabelPath = orig }
}
