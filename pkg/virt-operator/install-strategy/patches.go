package installstrategy

import (
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

	v1 "kubevirt.io/client-go/api/v1"
)

type Customizer struct {
	Patches []v1.Patch

	hash string
}

func NewCustomizer(customizations v1.CustomizeComponents) (*Customizer, error) {
	hash, err := getHash(customizations)
	if err != nil {
		return &Customizer{}, err
	}

	return &Customizer{
		Patches: customizations.Patches,
		hash:    hash,
	}, nil
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

			patches = append(patches, v1.Patch{
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

func applyPatches(obj runtime.Object, patches []v1.Patch) error {
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

func applyPatch(obj runtime.Object, patch v1.Patch) error {
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

func (c *Customizer) GetPatches() []v1.Patch {
	return c.Patches
}

func (c *Customizer) GetPatchesForResource(resourceType, name string) []v1.Patch {
	allPatches := c.Patches
	patches := make([]v1.Patch, 0)

	for _, p := range allPatches {
		if strings.EqualFold(p.ResourceType, resourceType) && strings.EqualFold(p.ResourceName, name) {
			patches = append(patches, p)
		}
	}

	return patches
}

func getHash(customizations v1.CustomizeComponents) (string, error) {
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
