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
 * Copyright 2022
 *
 */

package sev

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_SEV               = "sev"
	COMMAND_FETCH_CERT_CHAIN  = "fetch-cert-chain"
	COMMAND_SETUP_SESSION     = "setup-session"
	COMMAND_QUERY_MEASUREMENT = "query-measurement"
	COMMAND_INJECT_SECRET     = "inject-secret"
	output_perm               = 0750

	sessionFlag = "session"
	dhcertFlag  = "dhcert"
	secretFlag  = "secret"
	headerFlag  = "header"
	outputFlag  = "output"
)

var (
	session   string
	godh      string
	secret    string
	header    string
	outpath   string
	vmiName   string
	namespace string
)

type SEVCommand struct {
	clientConfig clientcmd.ClientConfig
	command      string
}

func NewSEVCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	sevCmd := &cobra.Command{
		Use:     "sev (VMI)",
		Short:   "Interact with the SEV platform",
		Example: usage(COMMAND_SEV),
		Args:    templates.ExactArgs(COMMAND_SEV, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := SEVCommand{command: COMMAND_SEV, clientConfig: clientConfig}
			return c.Run(args)
		},
	}
	sevCmd.SetUsageTemplate(templates.MainUsageTemplate())
	sevCmd.AddCommand(
		NewFetchCertChainCommand(clientConfig),
		NewSetupSessionCommand(clientConfig),
		NewQueryMeasurementCommand(clientConfig),
		NewInjectSecretCommand(clientConfig),
	)

	return sevCmd
}

func NewFetchCertChainCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fetch-cert-chain (VMI)",
		Short:   "Fetch Certificate Chain",
		Long:    `Fetch the platform Diffie-Hellman (PDH) and the complete certificate chain from the SEV platform`,
		Args:    templates.ExactArgs(COMMAND_FETCH_CERT_CHAIN, 1),
		Example: usage(COMMAND_FETCH_CERT_CHAIN),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := SEVCommand{
				command:      COMMAND_FETCH_CERT_CHAIN,
				clientConfig: clientConfig,
			}
			return c.Run(args)
		},
	}

	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&outpath, outputFlag, "",
		"Filepath where the certificate chain is to be written")

	return cmd
}

func NewSetupSessionCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "setup-session (VMI)",
		Short:   "Pass session launch parameters into the VM",
		Long:    `Set up a secure communication channel between the guest and the SEV platform.`,
		Args:    templates.ExactArgs(COMMAND_SETUP_SESSION, 1),
		Example: usage(COMMAND_SETUP_SESSION),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := SEVCommand{
				command:      COMMAND_SETUP_SESSION,
				clientConfig: clientConfig,
			}
			return c.Run(args)
		},
	}

	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&session, sessionFlag, "",
		"(Required) Base64 encoded session launch blob")
	cmd.MarkFlagRequired(sessionFlag)
	cmd.Flags().StringVar(&godh, dhcertFlag, "",
		"(Required) Base64 encoded guest owner certicate")
	cmd.MarkFlagRequired(dhcertFlag)

	return cmd
}

func NewQueryMeasurementCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-measurement (VMI)",
		Short: "Query the launch measurement from the SEV platform",
		Long: `Query the launch measurement from the SEV platform
The guest owner can verify the measurement against the measurement provided by the platform owner`,
		Args:    templates.ExactArgs(COMMAND_QUERY_MEASUREMENT, 1),
		Example: usage(COMMAND_QUERY_MEASUREMENT),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := SEVCommand{
				command:      COMMAND_QUERY_MEASUREMENT,
				clientConfig: clientConfig,
			}
			return c.Run(args)
		},
	}

	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&outpath, outputFlag, "",
		"Filepath where platform measurement info is to be written")
	return cmd
}

func NewInjectSecretCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "inject-secret (VMI)",
		Short:   "Inject the guest owner's secret into the running VM",
		Long:    `Inject the guest owner's secret into the running VM`,
		Args:    templates.ExactArgs(COMMAND_INJECT_SECRET, 1),
		Example: usage(COMMAND_INJECT_SECRET),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := SEVCommand{
				command:      COMMAND_INJECT_SECRET,
				clientConfig: clientConfig,
			}
			return c.Run(args)
		},
	}

	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&secret, secretFlag, "", "(Required) Guest owner secret key")
	cmd.MarkFlagRequired(secretFlag)
	cmd.Flags().StringVar(&header, headerFlag, "", "(Required) Guest owner secret key header")
	cmd.MarkFlagRequired(headerFlag)

	return cmd
}

func usage(cmd string) string {
	if cmd == COMMAND_SEV {
		return fmt.Sprintf(" {{ProgramName}} sev <subcommand> myvmi")
	}
	return fmt.Sprintf(" {{ProgramName}} sev %s myvmi", cmd)
}

func writeOutput(cmd string, data []byte) error {
	var logFmt, errFmt string
	switch cmd {
	case COMMAND_FETCH_CERT_CHAIN:
		logFmt = "[SEV]Writing platform certificate chain to %s"
		errFmt = "Error %s/%s: Failed to write certificate chain to %s: %v"
	case COMMAND_QUERY_MEASUREMENT:
		logFmt = "[SEV]Writing platform measurement info to %s"
		errFmt = "Error %s/%s: Failed to write platform measurement info to %s: %v"
	// Only FETCH_CERT_CHAIN and QUERY_MEASUREMENT support writing to a file
	default:
		return nil
	}

	if outpath != "" {
		file, err := os.OpenFile(outpath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, output_perm)
		if err != nil {
			return fmt.Errorf(errFmt, namespace, vmiName, outpath, err)
		}
		defer file.Close()

		_, err = file.Write(data)
		if err != nil {
			return fmt.Errorf(errFmt, namespace, vmiName, outpath, err)
		}
		log.Log.Infof(logFmt, outpath)
	} else {
		fmt.Printf("%s\n", string(data))
	}

	return nil
}

func (sc *SEVCommand) Run(args []string) error {
	var err error
	vmiName = args[0]
	namespace, _, err = sc.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(sc.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	switch sc.command {
	case COMMAND_SEV:
		return fmt.Errorf("Error %s/%s: A subcommand is needed to execute the sev command.\nRun 'virtctl sev --help' for more info", namespace, vmiName)

	case COMMAND_FETCH_CERT_CHAIN:
		sevPlatformInfo, err := virtClient.VirtualMachineInstance(namespace).SEVFetchCertChain(vmiName)
		if err != nil {
			return fmt.Errorf("Error %s/%s: %v", namespace, vmiName, err)
		}
		data, err := json.MarshalIndent(sevPlatformInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("Error %s/%s: Cannot marshal SEV platform info: %v", namespace, vmiName, err)
		}
		err = writeOutput(COMMAND_FETCH_CERT_CHAIN, data)
		if err != nil {
			return err
		}

	case COMMAND_SETUP_SESSION:
		sevSessionOptions := &v1.SEVSessionOptions{
			Session: session,
			DHCert:  godh,
		}
		err := virtClient.VirtualMachineInstance(namespace).SEVSetupSession(vmiName, sevSessionOptions)
		if err != nil {
			return fmt.Errorf("Error %s/%s: %v", namespace, vmiName, err)
		}

	case COMMAND_QUERY_MEASUREMENT:
		sevMeasurementInfo, err := virtClient.VirtualMachineInstance(namespace).SEVQueryLaunchMeasurement(vmiName)
		if err != nil {
			return fmt.Errorf("Error %s/%s: %v", namespace, vmiName, err)
		}
		data, err := json.MarshalIndent(sevMeasurementInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("Error %s/%s: Cannot marshal SEV platform measurement: %v", namespace, vmiName, err)
		}
		err = writeOutput(COMMAND_QUERY_MEASUREMENT, data)
		if err != nil {
			return err
		}

	case COMMAND_INJECT_SECRET:
		sevSecretOptions := &v1.SEVSecretOptions{
			Header: header,
			Secret: secret,
		}
		err := virtClient.VirtualMachineInstance(namespace).SEVInjectLaunchSecret(vmiName, sevSecretOptions)
		if err != nil {
			return fmt.Errorf("Error %s/%s: %v", namespace, vmiName, err)
		}
	}

	return nil
}
