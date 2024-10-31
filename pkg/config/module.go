package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	J "cuelang.org/go/encoding/json"
	"cuelang.org/go/encoding/yaml"
)

//go:embed schema.cue
var schemaFile string

//go:embed default.yaml
var DEFAULT []byte

func readFile(ctx *cue.Context, path string) (*cue.Value, error) {
	// Check if this is a valid file
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("does not exist")
	}

	extension := filepath.Ext(path)
	switch extension {
	case ".json":
		dataFile, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		dataExpr, err := J.Extract(path, dataFile)
		if err != nil {
			return nil, err
		}

		value := ctx.BuildExpr(dataExpr)
		if err := value.Err(); err != nil {
			return nil, err
		}

		return &value, nil
	case ".yaml":
		yamlFile, err := yaml.Extract(path, nil)
		if err != nil {
			return nil, err
		}

		value := ctx.BuildFile(yamlFile)
		if err := value.Err(); err != nil {
			return nil, err
		}

		return &value, nil
	}

	return nil, fmt.Errorf(
		"not in a valid format",
	)
}

// Process reads the provided configuration files in order, compiles them,
// and unifies them with the configuration file schema. If no configuration
// files are provided, the default configuration is used.
func Process(configPaths []string) (*Config, error) {
	ctx := cuecontext.New()

	// Compile the schema
	schema := ctx.CompileString(schemaFile)
	err := schema.Err()
	if err != nil {
		return nil, err
	}

	if len(configPaths) == 0 {
		// Load default config
		yamlFile, err := yaml.Extract("<default>", []byte(DEFAULT))
		if err != nil {
			return nil, err
		}

		value := ctx.BuildFile(yamlFile)
		if err := value.Err(); err != nil {
			return nil, err
		}

		schema = schema.Unify(value)
		if err := schema.Err(); err != nil {
			return nil, fmt.Errorf(
				"invalid default config file: %v",
				err,
			)
		}
	}

	for _, path := range configPaths {
		value, err := readFile(ctx, path)
		if err != nil {
			return nil, fmt.Errorf(
				"could not process config file %s: %v",
				path,
				err,
			)
		}

		schema = schema.Unify(*value)
		if err := schema.Err(); err != nil {
			return nil, fmt.Errorf(
				"could not merge config file %s: %v",
				path,
				err,
			)
		}

		// Check if the config file is valid
		err = schema.Validate()
		if err != nil {
			return nil, fmt.Errorf(
				"config file %s is not valid: %v",
				path,
				err,
			)
		}
	}

	if err := schema.Validate(); err != nil {
		return nil, err
	}

	data, err := schema.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf(
			"could not aggregate config: %v",
			err,
		)
	}

	config := Config{}
	err = json.Unmarshal(data, &config)
	return &config, err
}
