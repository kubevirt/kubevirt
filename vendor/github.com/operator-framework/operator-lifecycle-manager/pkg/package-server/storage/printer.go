package storage

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/kubernetes/pkg/printers"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators"
)

// translateTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

func addTableHandlers(h printers.PrintHandler) {
	podColumnDefinitions := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Catalog", Type: "string", Description: "The source catalog for this package"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	h.TableHandler(podColumnDefinitions, printPackage)
	h.TableHandler(podColumnDefinitions, printPackageList)

}

func printPackage(manifest *operators.PackageManifest, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: manifest},
	}
	row.Cells = append(row.Cells, manifest.Name, manifest.Status.CatalogSourceDisplayName, translateTimestampSince(manifest.CreationTimestamp))
	return []metav1beta1.TableRow{row}, nil
}

func printPackageList(manifestList *operators.PackageManifestList, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(manifestList.Items))
	for i := range manifestList.Items {
		r, err := printPackage(&manifestList.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}
