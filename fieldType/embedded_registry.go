package fieldtype

import (
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"

	jsonschema6 "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yannh/kubeconform/pkg/loader"
	"github.com/yannh/kubeconform/pkg/registry"
)

//go:embed master-standalone-strict
var schemafs embed.FS

type embeddedFallbackRegistry struct {
	reg registry.Registry
	fs fs.FS
}

func newEmbeddedFallbackRegistry(url string) (embeddedFallbackRegistry, error) {
	reg, err := registry.New(url, "", true, false, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not set up registry for native schemas")
	}
	return embeddedFallbackRegistry{
		reg: reg,
		fs: schemafs,
	}, nil
}

func (er embeddedFallbackRegistry) DownloadSchema(kind, apiVersion, k8sVersion string) (string, any, error) {
	urlString, schema, wrappedErr := er.reg.DownloadSchema(kind, apiVersion, k8sVersion)
	switch wrappedErr.(type) {
	case *loader.NotFoundError:
		u, err := url.Parse(urlString)
		if err != nil {
			return urlString, nil, err
		}

		_, fname := path.Split(u.Path)
		f, err := er.fs.Open(fname)
		if err != nil {
			return urlString, nil, wrappedErr
		}

		schema, err := jsonschema6.UnmarshalJSON(f)
		return urlString, schema, err
	default:
		return urlString, schema, wrappedErr
	}
}
