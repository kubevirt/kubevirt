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

package portforward

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"kubevirt.io/client-go/kubecli"
)

func TestProxyFlagsAppliedBeforeRun(t *testing.T) {
	t.Setenv("VIRTCTL_PROXY_FLAGS", `["--context=from-env","--namespace=from-env"]`)
	root := &cobra.Command{Use: "virtctl", SilenceErrors: true, SilenceUsage: true}
	var context, namespace string
	root.PersistentFlags().StringVar(&context, "context", "", "")
	root.PersistentFlags().StringVar(&namespace, "namespace", "", "")
	cmd := NewCommand()
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		if context != "from-env" || namespace != "from-env" {
			t.Errorf("flags were not applied before run: %q %q", context, namespace)
		}
		if _, ok := os.LookupEnv("VIRTCTL_PROXY_FLAGS"); ok {
			t.Error("environment variable was not removed before run")
		}
		return nil
	}
	root.AddCommand(cmd)
	root.SetArgs([]string{"port-forward", "vmi/test", "22"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestProxyFlagsPrecedenceAndRepeats(t *testing.T) {
	t.Setenv("VIRTCTL_PROXY_FLAGS", `["--context=one","--context=two","--namespace=","--as-group=one","--as-group=two"]`)
	root := &cobra.Command{Use: "virtctl", SilenceErrors: true, SilenceUsage: true}
	var context, namespace string
	var groups []string
	root.PersistentFlags().StringVar(&context, "context", "default", "")
	root.PersistentFlags().StringVar(&namespace, "namespace", "default", "")
	root.PersistentFlags().StringArrayVar(&groups, "as-group", nil, "")
	cmd := NewCommand()
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		if context != "cli" || namespace != "" || !reflect.DeepEqual(groups, []string{"cli-one", "cli-two"}) {
			t.Errorf("got context=%q namespace=%q groups=%q", context, namespace, groups)
		}
		return nil
	}
	root.AddCommand(cmd)
	root.SetArgs([]string{"port-forward", "vmi/test", "22", "--context=cli", "--as-group=cli-one", "--as-group=cli-two"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestProxyFlagsReachClientConfig(t *testing.T) {
	t.Setenv("VIRTCTL_PROXY_FLAGS", `["--server=https://transported.example","--namespace=transported","--token=example"]`)
	root := &cobra.Command{Use: "virtctl", SilenceErrors: true, SilenceUsage: true}
	config := kubecli.DefaultClientConfig(root.PersistentFlags())
	cmd := NewCommand()
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		got, err := config.ClientConfig()
		if err != nil {
			t.Fatal(err)
		}
		ns, _, err := config.Namespace()
		if err != nil {
			t.Fatal(err)
		}
		if got.Host != "https://transported.example" || ns != "transported" {
			t.Errorf("host=%q namespace=%q", got.Host, ns)
		}
		return nil
	}
	root.AddCommand(cmd)
	root.SetArgs([]string{"port-forward", "vmi/test", "22"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestProxyFlagsErrorsDoNotExposeValues(t *testing.T) {
	const value = "dummy-sensitive-value"
	payload, err := json.Marshal([]string{"--insecure-skip-tls-verify=" + value})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("VIRTCTL_PROXY_FLAGS", string(payload))
	root := &cobra.Command{Use: "virtctl", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().Bool("insecure-skip-tls-verify", false, "")
	root.AddCommand(NewCommand())
	root.SetArgs([]string{"port-forward", "vmi/test", "22"})
	err = root.Execute()
	if err == nil || strings.Contains(err.Error(), value) {
		t.Errorf("expected a value-free validation error, got %v", err)
	}
}

func TestInvalidProxyFlagsAreUnset(t *testing.T) {
	for _, raw := range []string{``, `null`, `{}`, `[null]`, `[1]`, `["--unknown=x"]`, `["positional"]`, `["--context"]`} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("VIRTCTL_PROXY_FLAGS", raw)
			root := &cobra.Command{Use: "virtctl", SilenceErrors: true, SilenceUsage: true}
			root.PersistentFlags().String("context", "", "")
			cmd := NewCommand()
			ran := false
			cmd.RunE = func(_ *cobra.Command, _ []string) error { ran = true; return nil }
			root.AddCommand(cmd)
			root.SetArgs([]string{"port-forward", "vmi/test", "22"})
			err := root.Execute()
			if err == nil || !strings.Contains(err.Error(), "VIRTCTL_PROXY_FLAGS") {
				t.Errorf("expected proxy flags error, got %v", err)
			}
			if ran {
				t.Error("run called for invalid input")
			}
			if _, ok := os.LookupEnv("VIRTCTL_PROXY_FLAGS"); ok {
				t.Error("environment variable was not removed")
			}
		})
	}
}
