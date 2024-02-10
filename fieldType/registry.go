package fieldtype

import (
	"embed"
	"fmt"
	"os"
	"strings"
)

//go:embed schemas/*
var schemafs embed.FS

type reg struct{}
	// nativeRegUrl = "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{.NormalizedKubernetesVersion}}-standalone{{.StrictSuffix}}/{{.ResourceKind}}{{.KindSuffix}}.json"
	// crdsRegUrl   = "https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json"

func (r reg) DownloadSchema(kind, groupVersion, _ string) (string, []byte, error) {
	kind = strings.ToLower(kind)
	var path string
	if !strings.Contains(groupVersion, ".") {
		// assume native. Could be like v1 or like apps/v1
		groupVersionParts := strings.Split(groupVersion, "/")
		if len(groupVersionParts) == 1 {
			path = fmt.Sprintf("schemas/native/master-standalone/%s-%s.json", kind, groupVersion)
		} else {
			group := groupVersionParts[0]
			version := groupVersionParts[1]
			path = fmt.Sprintf("schemas/native/master-standalone/%s-%s-%s.json", kind, group, version)
		}
	} else {
		groupVersionParts := strings.Split(groupVersion, "/")
		group := groupVersionParts[0]
		version := groupVersionParts[1]
		path = fmt.Sprintf("schemas/crds/%s/%s_%s.json", group, kind, version)
	}

	dat, err := schemafs.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, kind, groupVersion, path)
	}

	return "", dat, err
}
