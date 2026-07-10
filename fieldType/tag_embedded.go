//go:build embeddedSchemas

package fieldtype

import (
	"embed"
)

//go:embed master-standalone-strict
var _schemafs embed.FS

func init() {
	schemafs = _schemafs
	crdsFirst = true
}
