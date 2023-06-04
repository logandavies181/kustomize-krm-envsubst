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
	envMapping envsubst.AdvancedMapping

	AllowEmpty   bool     `yaml:"allowEmpty" json:"allowEmpty"`
	ExcludedVars []string `yaml:"excludedVariableNames" json:"excludedVariableNames"`
	IncludedVars []string `yaml:"includedVariableNames" json:"includedVariableNames"`
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

func looksLikeNumber(s string) bool {
	reg := regexp.MustCompile(`^[0-9]+(\.[0-9])*$`)
	return reg.MatchString(s)
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
	key, err := in.Key.String()
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
	str, err := in.String()
	if err != nil {
		return nil, fmt.Errorf("Could not parse node into string: %v", err)
	}

	substed, err := envsubst.EvalAdvanced(str, envsubst.AdvancedMapping(c.envMapping))
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
		if looksLikeNumber(substed) {
			substed = `"` + substed + `"`
		}
	}
	node, err = yaml.Parse(substed)
	if err != nil {
		return nil, fmt.Errorf("Could not parse node after envsubsting: %v", err)
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

// environmentSubstitute performs an environment substitution. It
// also inspects the args to infer if the value is intended to be
// a string that represents an integer. If so, it will wrap it in
// quotes
func environmentSubstitute(s string, nodeInfo envsubst.NodeInfo) string {
	looksLikeQuotedInt := false
	for _, arg := range nodeInfo.Args() {
		if len(arg) > 2 && arg[0] == byte('"') && arg[0] == arg[len(arg)-1] {
			looksLikeQuotedInt = yaml.IsIdxNumber(arg[1 : len(arg)-1])
		}
	}

	mapped := os.Getenv(s)
	ret := nodeInfo.Result(mapped)

	// check if args look like quoted integers
	// AND ( if the function has triggered OR the function had no effect )
	if looksLikeQuotedInt && (!contains(nodeInfo.Args(), ret) || ret == mapped) {
		ret = fmt.Sprintf(`"%s"`, ret)
	}

	return ret
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
			var c Config
			err = yaml.Unmarshal(fdata, &c)
			if err == nil {
				config.AllowEmpty = c.AllowEmpty
				config.IncludedVars = c.IncludedVars
				config.ExcludedVars = c.ExcludedVars
			}
		}
	}

	fn := func(items []*yaml.RNode) ([]*yaml.RNode, error) {
		config.envMapping = func(s string, nodeInfo envsubst.NodeInfo) (string, bool) {
			// IncludedVars and ExcludedVars are mutually exclusive
			// IncludedVars takes precedent
			// TODO: readme

			if len(config.IncludedVars) == 0 {
				if contains(config.ExcludedVars, s) {
					return nodeInfo.Orig(), false
				}

				return environmentSubstitute(s, nodeInfo), false
			}

			if !contains(config.IncludedVars, s) {
				return nodeInfo.Orig(), false
			}

			return environmentSubstitute(s, nodeInfo), false
		}

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
