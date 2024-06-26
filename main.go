package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	fieldtype "github.com/logandavies181/kustomize-krm-envsubst/fieldType"

	"github.com/logandavies181/envsubst"
	"github.com/logandavies181/go-buildversion"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var version string // goreleaser will set this

type Config struct {
	AllowEmpty   bool              `yaml:"allowEmpty" json:"allowEmpty"`
	ExcludedVars []string          `yaml:"excludedVariableNames" json:"excludedVariableNames"`
	IncludedVars []string          `yaml:"includedVariableNames" json:"includedVariableNames"`
	Values       map[string]string `yaml:"values" json:"values"`
}

func isEmpty(str string) bool {
	switch str {
	case "", `""`, "''", "\n":
		return true
	default:
		return false
	}
}

func contains(list []string, str string) bool {
	for _, v := range list {
		if str == v {
			return true
		}
	}

	return false
}

var numberRegex = regexp.MustCompile(`^[0-9]+(\.[0-9]*)?$`)

func looksLikeNumber(s string) bool {
	return numberRegex.MatchString(s)
}

// Yaml 1.1 and earlier uses all sorts of things for booleans
var yamlBoolRegex = regexp.
	MustCompile(`^(y|Y|yes|Yes|YES|n|N|no|No|NO|true|True|TRUE|false|False|FALSE|on|On|ON|off|Off|OFF)$`)

func looksLikeBool(s string) bool {
	return yamlBoolRegex.MatchString(s)
}

func (c Config) walkSequenceNode(in *yaml.RNode) error {
	in.AppendToFieldPath("[]")

	_, err := c.Filter(in)
	if err != nil {
		return err
	}

	return nil
}

func (c Config) walkMapNode(in *yaml.MapNode) error {
	_, err := c.Filter(in.Key)
	if err != nil {
		return err
	}

	key, err := in.Key.GetString(".")
	if err != nil {
		return err
	}
	in.Value.AppendToFieldPath(strings.TrimSuffix(key, "\n"))

	_, err = c.Filter(in.Value)
	if err != nil {
		return err
	}

	return nil
}

func (c Config) processScalarNode(in *yaml.RNode) (*yaml.RNode, error) {
	var str string
	var err error

	// The difference here is that RNode.String() gives a _representation_
	// of the _node_. If the value of the node is a string, that representation could
	// include a bit extra to represent a fold, i.e. | or |- and some whitespace, which
	// we don't want.
	//
	// GetString gives us the actual value being _held_ by the node,
	// but will error if the value is not a string.
	if in.IsStringValue() {
		str, err = in.GetString(".")
	} else {
		str, err = in.String()
	}

	if err != nil {
		return nil, fmt.Errorf("Could not parse node into string: %v", err)
	}

	substed, err := envsubst.EvalAdvanced(str, envsubst.AdvancedMapping(c.advMapping))
	if err != nil {
		return nil, fmt.Errorf("Could not envsubst: %v", err)
	}

	if substed == str {
		return in, nil
	}

	if isEmpty(substed) {
		if !c.AllowEmpty {
			return nil, fmt.Errorf(
				"Value `%s` evaluated to empty string. Did you forget to set an environment variable?",
				strings.TrimSuffix(str, "\n"))
		}

		substed = `""`
	}

	substed = strings.TrimSuffix(substed, "\n")

	t, err := fieldtype.GetFieldType(in.FieldPath())
	if err != nil {
		return nil, err
	}

	var node *yaml.RNode
	switch t {
	case fieldtype.String:
		if looksLikeNumber(substed) || looksLikeBool(substed) {
			// Somehow we end up with '"58008"' if we don't do this
			node = yaml.NewStringRNode(substed)
		}
	}

	// If our shim for string numbers just above didn't activate
	// then actually create the node
	if node == nil {
		node = yaml.NewScalarRNode(substed)
	}

	// shouldn't happen but would do weird stuff
	if node.YNode().Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("Invalid output: `%s` did not evaluate to a scalar", str)
	}

	return in.Pipe(yaml.Set(node))
}

func (c Config) Filter(in *yaml.RNode) (*yaml.RNode, error) {
	if in.IsNil() {
		return nil, nil
	}

	if len(in.FieldPath()) == 0 {
		in.AppendToFieldPath(in.GetApiVersion())
		in.AppendToFieldPath(in.GetKind())
	}

	switch y := in.YNode().Kind; y {
	case yaml.MappingNode:
		err := visitFields(in, c.walkMapNode)
		if err != nil {
			return nil, err
		}
		return in, nil
	case yaml.SequenceNode:
		err := visitElements(in, c.walkSequenceNode)
		if err != nil {
			return nil, err
		}

		return in, nil
	case yaml.ScalarNode:
		return c.processScalarNode(in)
	case yaml.AliasNode, yaml.DocumentNode:
		fallthrough
	default:
		panic(fmt.Sprintf("Unknown Kind: %v", y))
	}
}

// advMapping substitutes varName for its value from the environment or from
// explicit mappings defined by the user.
//
// This function is passed to envsubst.AdvancedMapping. The first result
// is the value to interpolate. The second is whether or not to perform
// the bash-like substitution afterward.
func (c Config) advMapping(varName string, nodeInfo envsubst.NodeInfo) (string, bool) {
	if !c.shouldMap(varName) {
		return nodeInfo.Orig(), false
	}

	var mapped string
	if val, ok := c.Values[varName]; ok {
		mapped = val
	} else {
		mapped = os.Getenv(varName)
	}

	return mapped, true
}

func (c Config) shouldMap(varName string) bool {
	// IncludedVars and ExcludedVars are mutually exclusive
	// IncludedVars takes precedent

	if len(c.IncludedVars) == 0 {
		if contains(c.ExcludedVars, varName) {
			return false
		}

		return true
	}

	if !contains(c.IncludedVars, varName) {
		return false
	}

	return true
}

func main() {
	config := new(Config)

	// This is a hack, but legacy exec plugins get config from the first arg.
	// The framework is going to do its own thing here, but let's try read the first arg
	// and parse it if it exists and is a file
	if len(os.Args) > 1 {
		fname := os.Args[1]
		fdata, err := os.ReadFile(fname)
		if err == nil {
			_ = yaml.Unmarshal(fdata, config)
		}
	}

	fn := func(items []*yaml.RNode) ([]*yaml.RNode, error) {
		for i := range items {
			err := items[i].PipeE(config)
			if err != nil {
				return nil, fmt.Errorf("kustomize-krm-envsubst: %v ", // leave a space because kustomize doesn't
					err)
			}
		}
		return items, nil
	}
	p := framework.SimpleProcessor{Config: config, Filter: kio.FilterFunc(fn)}
	cmd := command.Build(p, command.StandaloneDisabled, false)
	version, _ := buildversion.BuildVersionShortE(version)
	cmd.Version = version

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
