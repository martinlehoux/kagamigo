package kstore

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// JSONCodec encodes documents as indented JSON.
type JSONCodec struct{}

func (JSONCodec) Marshal(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}

func (JSONCodec) Unmarshal(data []byte, value any) error {
	return json.Unmarshal(data, value)
}

// YAMLCodec encodes documents as YAML.
type YAMLCodec struct{}

func (YAMLCodec) Marshal(value any) ([]byte, error) {
	return yaml.Marshal(value)
}

func (YAMLCodec) Unmarshal(data []byte, value any) error {
	return yaml.Unmarshal(data, value)
}

var (
	// JSON is the default JSON codec.
	JSON JSONCodec
	// YAML is the default YAML codec.
	YAML YAMLCodec
)
