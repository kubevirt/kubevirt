package main

import (
	"flag"
	"os"
	"text/template"

	"k8s.io/helm/pkg/chartutil"

	"github.com/spf13/pflag"

	corev1 "k8s.io/api/core/v1"
)

const (
	BinDir          = "/opt/cni/bin"
	BinDirOpenShift = "/var/lib/cni/bin"

	PasstBindingCNIImageDefault = "ghcr.io/kubevirt/passt-binding-cni@sha256:331a8b4dee412e4e79154d480d703a40a96a216944cfd4f9884c1ac58fed480f"
)

// PlacementConfiguration defines node placement configuration
type PlacementConfiguration struct {
	// Infra defines placement configuration for control-plane nodes
	Infra *Placement `json:"infra,omitempty"`
	// Workloads defines placement configuration for worker nodes
	Workloads *Placement `json:"workloads,omitempty"`
}

type Placement struct {
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Affinity     corev1.Affinity     `json:"affinity,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
}

type TemplateData struct {
	Namespace            string
	Placement            *Placement
	PasstBindingCNIImage string
	EnableSCC            bool
	ImagePullPolicy      string
	CNIBinDir            string
}

func main() {
	inputFile := "template.yaml"
	namespace := flag.String("namespace", "kubevirt", "The Namespace")
	isOpenShift := flag.Bool("openshift", false, "is cluster type OpenShift")
	imagePullPolicy := flag.String("imagePullPolicy", "IfNotPresent", "The ImagePullPolicy")
	passtBindingCNI := flag.String("passt-binding-cni", PasstBindingCNIImageDefault, "The passt binding cni image")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.ParseErrorsWhitelist.UnknownFlags = true
	pflag.Parse()

	tmpl := template.New(inputFile).Option("missingkey=error").Funcs(
		template.FuncMap{
			"toYaml": chartutil.ToYaml,
		},
	)

	manifestTemplate := template.Must(tmpl.ParseFiles(inputFile))

	cniBinDir := BinDir
	if *isOpenShift {
		cniBinDir = BinDirOpenShift
	}

	data := TemplateData{
		Namespace:            *namespace,
		Placement:            getDefaultPlacementConfiguration().Workloads,
		PasstBindingCNIImage: *passtBindingCNI,
		ImagePullPolicy:      *imagePullPolicy,
		CNIBinDir:            cniBinDir,
		EnableSCC:            *isOpenShift,
	}

	err := manifestTemplate.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}

func getDefaultPlacementConfiguration() PlacementConfiguration {
	return PlacementConfiguration{
		Infra: &Placement{
			Affinity: corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
						{
							Weight: 10,
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "node-role.kubernetes.io/control-plane",
										Operator: corev1.NodeSelectorOpExists,
									},
								},
							},
						},
						{
							Weight: 1,
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "node-role.kubernetes.io/master",
										Operator: corev1.NodeSelectorOpExists,
									},
								},
							},
						},
					},
				},
			},
		},
		Workloads: &Placement{
			NodeSelector: map[string]string{
				corev1.LabelOSStable: "linux",
			},
			Tolerations: []corev1.Toleration{
				corev1.Toleration{
					Key:      corev1.TaintNodeUnschedulable,
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
		},
	}
}
