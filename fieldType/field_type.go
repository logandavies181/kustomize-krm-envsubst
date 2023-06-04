package fieldtype

import (
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/yannh/kubeconform/pkg/registry"
)

type FieldType string

const (
	Unknown FieldType = "unknown"

	String  FieldType = "string"
	Array   FieldType = "array"
	Object  FieldType = "object"
	Integer FieldType = "integer"
	Number  FieldType = "number"
	Bool    FieldType = "bool"
	Null    FieldType = "null"
)

const (
	nativeReg = "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{.NormalizedKubernetesVersion}}-standalone{{.StrictSuffix}}/{{.ResourceKind}}{{.KindSuffix}}.json"
)

func GetFieldType(path []string) (FieldType, error) {
	reg, err := registry.New(nativeReg, "", true, false, false)
	if err != nil {
		return Unknown, err
	}

	if len(path) < 2 {
		return Unknown, fmt.Errorf("No GroupVersion or Kind")
	}

	_, data, err := reg.DownloadSchema(path[1], path[0], "master")
	if err != nil {
		return Unknown, err
	}

	sch, err := jsonschema.CompileString("", string(data))
	if err != nil {
		return Unknown, err
	}

	return walk(sch, path[2:]), nil
}

// we're only using this to work out if we need to stringify or numberify
// a field, so we prefer string if it's available and multiple scalar types
// are found
func walk(sch *jsonschema.Schema, path []string) FieldType {
	if sch.Ref != nil {
		sch = sch.Ref
	}

	// we've reach the end, let's try to find the type
	if len(path) == 0 {
		types := newHashset()
		if len(sch.Types) > 0 {
			types.add(sch.Types...)
		} else {
			for _, val := range sch.AnyOf {
				types.add(val.Types...)
			}

			if sch.Items != nil {
				types.add(string(Array))
			}
			if sch.Properties != nil {
				types.add(string(Object))
			}
		}

		if len(types) == 0 {
			return Unknown
		}

		switch {
		case types.has(string(Number)) && types.has(string(String)):
			// this looks silly but basically don't try to do anything
			// smart here - let the user put in something that will
			// be parsed as the correct type
			//
			// it's probably an IntOrString field and most tools
			// are smart about parsing those
			return Unknown

		case types.has(string(Integer)):
			return Integer
		case types.has(string(Number)):
			return Number
		case types.has(string(String)):
			return String
		case types.has(string(Bool)):
			return Bool
		default:
			// don't do anything smart if we somehow end up with a
			// non-scalar type or null
			return Unknown
		}
	}

	next := path[0]
	switch next {
	case "[]":
		switch val := sch.Items.(type) {
		case nil:
			if sch.Items2020 != nil {
				return walk(sch.Items2020, path[1:])
			}
		case *jsonschema.Schema:
			return walk(val, path[1:])
		case []*jsonschema.Schema:
			// suuurely if we encounter this, the length is at least 1
			// if not, return Unknown ¯\_(ツ)_/¯
			if len(val) == 0 {
				return Unknown
			}

			return walk(val[0], path[1:])

		default:
			return Unknown
		}
	default:
		// if it's not an array, assume it's an object key

		if sub, ok := sch.Properties[next]; !ok {
			// try additional Properties
			switch val := sch.AdditionalProperties.(type) {
			case *jsonschema.Schema:
				return walk(val, path[1:])

			default:
				return Unknown
			}
		} else {
			return walk(sub, path[1:])
		}
	}

	return Unknown
}
