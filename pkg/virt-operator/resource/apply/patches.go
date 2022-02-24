package apply

import (
	// #nosec sha1 is used to calculate a hash for patches and not for cryptographic
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
)

type Customizer struct {
	Patches []v1.CustomizeComponentsPatch

	hash string
}

func NewCustomizer(customizations v1.CustomizeComponents) (*Customizer, error) {
	hash, err := getHash(customizations)
	if err != nil {
		return &Customizer{}, err
	}

	patches := customizations.Patches
	flagPatches := flagsToPatches(customizations.Flags)
	patches = append(patches, flagPatches...)

	return &Customizer{
		Patches: patches,
		hash:    hash,
	}, nil
}

func flagsToPatches(flags *v1.Flags) []v1.CustomizeComponentsPatch {
	patches := []v1.CustomizeComponentsPatch{}
	if flags == nil {
		return patches
	}

	patches = addFlagsPatch(components.VirtAPIName, "Deployment", flags.API, patches)
	patches = addFlagsPatch(components.VirtControllerName, "Deployment", flags.Controller, patches)
	patches = addFlagsPatch(components.VirtHandlerName, "DaemonSet", flags.Handler, patches)

	return patches
}

func addFlagsPatch(name, resource string, flags map[string]string, patches []v1.CustomizeComponentsPatch) []v1.CustomizeComponentsPatch {
	if len(flags) == 0 {
		return patches
	}

	return append(patches, v1.CustomizeComponentsPatch{
		ResourceName: name,
		ResourceType: resource,
		Patch:        fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":%q,"command":["%s","%s"]}]}}}}`, name, name, strings.Join(flagsToArray(flags), `","`)),
		Type:         v1.StrategicMergePatchType,
	})
}

func flagsToArray(flags map[string]string) []string {
	farr := make([]string, 0)

	for flag, v := range flags {
		farr = append(farr, fmt.Sprintf("--%s", strings.ToLower(flag)))
		if v != "" {
			farr = append(farr, v)
		}
	}

	return farr
}

func (c *Customizer) Hash() string {
	return c.hash
}

func (c *Customizer) GenericApplyPatches(objects interface{}) error {
	switch reflect.TypeOf(objects).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(objects)
		for i := 0; i < s.Len(); i++ {
			o := s.Index(i)
			obj, ok := o.Interface().(runtime.Object)
			if !ok {
				return errors.New("Slice must contain objects of type 'runtime.Object'")
			}

			kind := obj.GetObjectKind().GroupVersionKind().Kind

			v := reflect.Indirect(o).FieldByName("ObjectMeta").FieldByName("Name")
			name := v.String()

			patches := c.GetPatchesForResource(kind, name)

			patches = append(patches, v1.CustomizeComponentsPatch{
				Patch: fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s"}}}`, v1.KubeVirtCustomizeComponentAnnotationHash, c.hash),
				Type:  v1.StrategicMergePatchType,
			})

			err := applyPatches(obj, patches)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Customizer) Apply(targetStrategy *install.Strategy) error {
	err := c.GenericApplyPatches(targetStrategy.Deployments())
	if err != nil {
		return err
	}
	err = c.GenericApplyPatches(targetStrategy.Services())
	if err != nil {
		return err
	}
	err = c.GenericApplyPatches(targetStrategy.DaemonSets())
	if err != nil {
		return err
	}
	err = c.GenericApplyPatches(targetStrategy.ValidatingWebhookConfigurations())
	if err != nil {
		return err
	}
	err = c.GenericApplyPatches(targetStrategy.MutatingWebhookConfigurations())
	if err != nil {
		return err
	}
	err = c.GenericApplyPatches(targetStrategy.APIServices())
	if err != nil {
		return err
	}
	err = c.GenericApplyPatches(targetStrategy.CertificateSecrets())
	if err != nil {
		return err
	}

	return nil
}

func applyPatches(obj runtime.Object, patches []v1.CustomizeComponentsPatch) error {
	if len(patches) == 0 {
		return nil
	}

	for _, p := range patches {
		err := applyPatch(obj, p)
		if err != nil {
			return err
		}
	}

	return nil
}

func applyPatch(obj runtime.Object, patch v1.CustomizeComponentsPatch) error {
	if obj == nil {
		return nil
	}

	old, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	// reset the object in preparation to unmarshal, since unmarshal does not guarantee that fields
	// in obj that are removed by patch are cleared
	value := reflect.ValueOf(obj)
	value.Elem().Set(reflect.New(value.Type().Elem()).Elem())

	switch patch.Type {
	case v1.JSONPatchType:
		patch, err := jsonpatch.DecodePatch([]byte(patch.Patch))
		if err != nil {
			return err
		}
		modified, err := patch.Apply(old)
		if err != nil {
			return err
		}

		if err = json.Unmarshal(modified, obj); err != nil {
			return err
		}
	case v1.MergePatchType:
		modified, err := jsonpatch.MergePatch(old, []byte(patch.Patch))
		if err != nil {
			return err
		}

		if err := json.Unmarshal(modified, obj); err != nil {
			return err
		}
	case v1.StrategicMergePatchType:
		mergedByte, err := strategicpatch.StrategicMergePatch(old, []byte(patch.Patch), obj)
		if err != nil {
			return err
		}

		if err = json.Unmarshal(mergedByte, obj); err != nil {
			return err
		}
	default:
		return fmt.Errorf("PatchType is not supported")
	}

	return nil
}

func (c *Customizer) GetPatches() []v1.CustomizeComponentsPatch {
	return c.Patches
}

func (c *Customizer) GetPatchesForResource(resourceType, name string) []v1.CustomizeComponentsPatch {
	allPatches := c.Patches
	patches := make([]v1.CustomizeComponentsPatch, 0)

	for _, p := range allPatches {
		if valueMatchesKey(p.ResourceType, resourceType) && valueMatchesKey(p.ResourceName, name) {
			patches = append(patches, p)
		}
	}

	return patches
}

func valueMatchesKey(value, key string) bool {
	if value == "*" {
		return true
	}

	return strings.EqualFold(key, value)
}

func getHash(customizations v1.CustomizeComponents) (string, error) {
	// #nosec CWE: 326 - Use of weak cryptographic primitive (http://cwe.mitre.org/data/definitions/326.html)
	// reason: sha1 is not used for encryption but for creating a hash value
	hasher := sha1.New()

	sort.SliceStable(customizations.Patches, func(i, j int) bool {
		return len(customizations.Patches[i].Patch) < len(customizations.Patches[j].Patch)
	})

	values, err := json.Marshal(customizations)
	if err != nil {
		return "", err
	}
	hasher.Write(values)

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
