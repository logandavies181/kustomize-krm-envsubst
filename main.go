package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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

func (c Config) walkSequenceNode(in *yaml.RNode) error {
	_, err := c.Filter(in)
	if err != nil {
		return err
	}

	return nil
}

func (c Config) walkMapNode(in *yaml.MapNode) error {
	if key, err := in.Key.String(); err == nil {
		if key == "annotations\n" || key == "labels\n" {
			return in.Value.VisitFields(c.walkMetadataNode)
		}
	} else {
		return err
	}

	_, err := c.Filter(in.Value)
	if err != nil {
		return err
	}

	return nil
}

// walkMetadataNode is the same as Filter for a scalar node,
// except that it ensures the value is always treated as a string
func (c Config) walkMetadataNode(in *yaml.MapNode) error {
	return c.processScalarNode(in.Value, true)
}

func (c Config) processScalarNode(in *yaml.RNode, alwaysString bool) error {
	str, err := in.String()
	if err != nil {
		return fmt.Errorf("Could not parse node into string: %v", err)
	}

	substed, err := envsubst.EvalAdvanced(str, envsubst.AdvancedMapping(c.envMapping))
	if err != nil {
		return fmt.Errorf("Could not envsubst: %v", err)
	}

	if substed == str {
		return nil
	}

	if isEmpty(substed) {
		if !c.AllowEmpty {
			return fmt.Errorf(
				"Value `%s` evaluated to empty string. Did you forget to set an environment variable?",
				strings.TrimSuffix(str, "\n"))
		}

		substed = `""`
	}

	if alwaysString {
		substed = strings.TrimSuffix(substed, "\n")
		if _, strconvErr := strconv.Atoi(string(substed[0])); strconvErr == nil {
			substed = `"` + substed + `"`
		}
	}

	strNode, err := yaml.Parse(substed)
	if err != nil {
		fmt.Fprintln(os.Stderr, substed)
		fmt.Fprintln(os.Stderr, len(substed))
		return fmt.Errorf("Could not parse node after envsubsting: %v", err)
	}

	_, err = in.Pipe(yaml.Set(strNode))

	return err
}

func (c Config) Filter(in *yaml.RNode) (*yaml.RNode, error) {
	if in.IsNil() {
		return nil, nil
	}

	switch y := in.YNode().Kind; y {
	case yaml.MappingNode:
		err := in.VisitFields(c.walkMapNode)
		if err != nil {
			return nil, err
		}
		return in, nil
	case yaml.SequenceNode:
		err := in.VisitElements(c.walkSequenceNode)
		if err != nil {
			return nil, err
		}

		return in, nil
	case yaml.ScalarNode:
		return nil, c.processScalarNode(in, false)
	case yaml.AliasNode, yaml.DocumentNode:
		fallthrough
	default:
		panic(fmt.Sprintf("Unknown Kind: %v", y))
	}
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
		config.envMapping = func(str string, nodeInfo envsubst.NodeInfo) (string, bool) {
			// IncludedVars and ExcludedVars are mutually exclusive
			// IncludedVars takes precedent
			// TODO: readme

			if len(config.IncludedVars) == 0 {
				if contains(config.ExcludedVars, str) {
					return nodeInfo.Orig(), false
				}

				return str, true
			}

			if !contains(config.IncludedVars, str) {
				return nodeInfo.Orig(), false
			}

			return str, true
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
