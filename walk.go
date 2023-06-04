package main

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// visitFields is a copy of yaml.(*RNode).VisitFields that preserves
// the FieldPath of the parent
func visitFields(rn *yaml.RNode, fn func(node *yaml.MapNode) error) error {
	// get the list of srcFieldNames
	srcFieldNames, err := rn.Fields()
	if err != nil {
		return err
	}

	// visit each field
	for _, fieldName := range srcFieldNames {
		if err := fn(field(rn, fieldName)); err != nil {
			return err
		}
	}
	return nil
}

// field is a copy of yaml(*RNode).Field that preserves the FieldPath
// of the parent
func field(rn *yaml.RNode, field string) *yaml.MapNode {
	if rn.YNode().Kind != yaml.MappingNode {
		return nil
	}
	var result *yaml.MapNode
	visitMappingNodeFields(rn.Content(), func(key, value *yaml.Node) {
		valOut := yaml.NewRNode(value)
		valOut.AppendToFieldPath(rn.FieldPath()...)

		result = &yaml.MapNode{Key: yaml.NewRNode(key), Value: valOut}
	}, field)
	return result
}

// visitMappingNodeFields calls fn for fields in the content, in content order.
// The caller is responsible to ensure the node is a mapping node. If fieldNames
// are specified, then fn is called only for the fields that match the given
// fieldNames.
func visitMappingNodeFields(content []*yaml.Node, fn func(key, value *yaml.Node), fieldNames ...string) {
	switch len(fieldNames) {
	case 0: // visit all fields
		visitFieldsWhileTrue(content, func(key, value *yaml.Node, _ int) bool {
			fn(key, value)
			return true
		})
	case 1: // visit single field
		visitFieldsWhileTrue(content, func(key, value *yaml.Node, _ int) bool {
			if key == nil {
				return true
			}
			if fieldNames[0] == key.Value {
				fn(key, value)
				return false
			}
			return true
		})
	default: // visit specified fields
		fieldsStillToVisit := make(map[string]bool, len(fieldNames))
		for _, fieldName := range fieldNames {
			fieldsStillToVisit[fieldName] = true
		}
		visitFieldsWhileTrue(content, func(key, value *yaml.Node, _ int) bool {
			if key == nil {
				return true
			}
			if fieldsStillToVisit[key.Value] {
				fn(key, value)
				delete(fieldsStillToVisit, key.Value)
			}
			return len(fieldsStillToVisit) > 0
		})
	}
}

// visitFieldsWhileTrue calls fn for the fields in content, in content order,
// until either fn returns false or all fields have been visited. The caller
// should ensure that content is from a mapping node, or fits the same expected
// pattern (consecutive key/value entries in the slice).
func visitFieldsWhileTrue(content []*yaml.Node, fn func(key, value *yaml.Node, keyIndex int) bool) {
	for i := 0; i < len(content); i += 2 {
		continueVisiting := fn(content[i], content[i+1], i)
		if !continueVisiting {
			return
		}
	}
}

// visitElements is a copy of yaml.(*RNode).VisitElements that preserves
// the FieldPath of the parent
func visitElements(rn *yaml.RNode, fn func(node *yaml.RNode) error) error {
	elements, err := elements(rn)
	if err != nil {
		return err
	}

	for i := range elements {
		if err := fn(elements[i]); err != nil {
			return err
		}
	}
	return nil
}

// elements is a copy of yaml.(*RNode).Elements that preserves the FieldPath
// of the parent
func elements(rn *yaml.RNode) ([]*yaml.RNode, error) {
	if err := yaml.ErrorIfInvalid(rn, yaml.SequenceNode); err != nil {
		return nil, err
	}
	var elements []*yaml.RNode
	for i := 0; i < len(rn.Content()); i++ {
		elem := yaml.NewRNode(rn.Content()[i])
		elem.AppendToFieldPath(rn.FieldPath()...)

		elements = append(elements, elem)
	}
	return elements, nil
}
