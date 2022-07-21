package banncheck

import (
	"embed"

	banncheck "kubevirt.io/kubevirt/tools/analyzers/banncheck/banncheck"

	"golang.org/x/tools/go/analysis"
)

//go:embed banncheck.json
var configFS embed.FS

var Analyzer *analysis.Analyzer

func init() {
	Analyzer = banncheck.NewAnalyzerWithFS(configFS)
	Analyzer.Flags.Lookup("configs").Value.Set("banncheck.json")
}
