package fieldtype

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"

	jsonschema6 "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yannh/kubeconform/pkg/loader"
	"github.com/yannh/kubeconform/pkg/registry"
)

var schemafs embed.FS

type embeddedFallbackRegistry struct {
	reg registry.Registry
	fs  fs.FS
}

func newEmbeddedFallbackRegistry(url string) (embeddedFallbackRegistry, error) {
	reg, err := registry.New(url, "", true, false, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not set up registry for native schemas")
	}
	return embeddedFallbackRegistry{
		reg: reg,
		fs:  schemafs,
	}, nil
}

func (er embeddedFallbackRegistry) DownloadSchema(kind, apiVersion, k8sVersion string) (string, any, error) {
	if kind == "" {
		return "", nil, loader.NewNotFoundError(fmt.Errorf("No kind supplied"))
	}

	parts := []string{
		kind,
	}
	if apiVersion != "" {
		parts = append(parts, strings.ReplaceAll(apiVersion, "/", "-"))
	}
	fname := strings.ToLower(
		fmt.Sprintf(
			"master-standalone-stric/%s.json",
			strings.Join(parts, "-"),
		),
	)

	f, err := er.fs.Open(fname)
	if err == nil {
		schema, err := jsonschema6.UnmarshalJSON(f)
		return fname, schema, err
	}

	return er.reg.DownloadSchema(kind, apiVersion, k8sVersion)
}
