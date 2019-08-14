package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/schema"
)

func main() {
	manifestDir := os.Args[1]

	err := schema.CheckCatalogResources(manifestDir)
	if err != nil {
		log.Fatal(err)
	}

	filepath.Walk(manifestDir, func(path string, f os.FileInfo, err error) error {
		if path == manifestDir || !f.IsDir() {
			return nil
		}

		fmt.Printf("Validating upgrade path for %s in %s\n", f.Name(), path)
		err = schema.CheckUpgradePath(path)
		if err != nil {
			log.Fatal(err)
		}
		return nil
	})
}
