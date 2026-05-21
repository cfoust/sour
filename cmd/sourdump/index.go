package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cfoust/sour/pkg/assets"

	"github.com/fxamacker/cbor/v2"
)

// dumpIndexToJSON reads a CBOR .index.source file and prints it as JSON.
func dumpIndexToJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var index assets.Index
	if err := cbor.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("decoding CBOR: %w", err)
	}

	jsonData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}
