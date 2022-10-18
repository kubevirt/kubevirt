package hyperconverged

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/blang/semver/v4"
	jsonpatch "github.com/evanphx/json-patch"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
)

const (
	upgradeChangesFileLocation = "./upgradePatches.json"
)

type hcoCRPatch struct {
	// SemverRange is a set of conditions which specify which versions satisfy the range
	// (see https://github.com/blang/semver#ranges as a reference).
	SemverRange string `json:"semverRange"`
	// JSONPatch contains a sequence of operations to apply to the HCO CR during upgrades
	// (see: https://datatracker.ietf.org/doc/html/rfc6902 as the format reference).
	JSONPatch jsonpatch.Patch `json:"jsonPatch"`
}

type objectToBeRemoved struct {
	// SemverRange is a set of conditions which specify which versions satisfy the range
	// (see https://github.com/blang/semver#ranges as a reference).
	SemverRange string `json:"semverRange"`
	// GroupVersionKind unambiguously identifies the kind of the object to be removed
	GroupVersionKind schema.GroupVersionKind `json:"groupVersionKind"`
	// objectKey contains name and namespace of the object to be removed.
	ObjectKey types.NamespacedName `json:"objectKey"`
}

type UpgradePatches struct {
	// hcoCRPatchList is a list of upgrade patches.
	// Each hcoCRPatch consists in a semver range of affected source versions and a json patch to be applied during the upgrade if relevant.
	HCOCRPatchList []hcoCRPatch `json:"hcoCRPatchList"`
	// ObjectsToBeRemoved is a list of objects to be removed on upgrades.
	// Each objectToBeRemoved consists in a semver range of affected source versions and schema.GroupVersionKind and types.NamespacedName of the object to be eventually removed during the upgrade.
	ObjectsToBeRemoved []objectToBeRemoved `json:"objectsToBeRemoved"`
}

var (
	hcoUpgradeChanges     UpgradePatches
	hcoUpgradeChangesRead = false
)

var getUpgradeChangesFileLocation = func() string {
	return upgradeChangesFileLocation
}

func readUpgradePatchesFromFile(req *common.HcoRequest) error {
	if hcoUpgradeChangesRead {
		return nil
	}
	hcoUpgradeChanges = UpgradePatches{}
	fileLocation := getUpgradeChangesFileLocation()

	file, err := os.Open(fileLocation)
	if err != nil {
		req.Logger.Error(err, "Can't open the upgradeChanges yaml file", "file name", fileLocation)
		return err
	}

	jsonBytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonBytes, &hcoUpgradeChanges)
	if err != nil {
		return err
	}
	hcoUpgradeChangesRead = true
	return nil
}

func validateUpgradePatches(req *common.HcoRequest) error {
	err := readUpgradePatchesFromFile(req)
	if err != nil {
		return err
	}
	for _, p := range hcoUpgradeChanges.HCOCRPatchList {
		if verr := validateUpgradePatch(req, p); verr != nil {
			return verr
		}
	}
	for _, r := range hcoUpgradeChanges.ObjectsToBeRemoved {
		if verr := validateUpgradeLeftover(req, r); verr != nil {
			return verr
		}
	}
	return nil
}

func validateUpgradePatch(req *common.HcoRequest, p hcoCRPatch) error {
	_, err := semver.ParseRange(p.SemverRange)
	if err != nil {
		return err
	}

	for _, patch := range p.JSONPatch {
		path, err := patch.Path()
		if err != nil {
			return err
		}
		if !strings.HasPrefix(path, "/spec/") {
			return errors.New("can only modify spec fields")
		}
	}
	specBytes, err := json.Marshal(req.Instance)
	if err != nil {
		return err
	}
	_, err = p.JSONPatch.Apply(specBytes)
	// tolerate jsonpatch test failures
	if err != nil && !errors.Is(err, jsonpatch.ErrTestFailed) {
		return err
	}
	return nil
}

func validateUpgradeLeftover(req *common.HcoRequest, r objectToBeRemoved) error {
	_, err := semver.ParseRange(r.SemverRange)
	if err != nil {
		return err
	}

	if r.GroupVersionKind.Kind == "" {
		return errors.New("missing object kind")
	}
	if r.GroupVersionKind.Version == "" {
		return errors.New("missing object API version")
	}
	if r.ObjectKey.Name == "" {
		return errors.New("missing object name")
	}
	return nil
}
