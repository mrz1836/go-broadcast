package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// configToJSON encodes cfg as indented JSON with exactly the keys, key order
// and values of its YAML encoding: the configuration types carry YAML tags
// only, so the YAML node tree is the single source of the document's shape.
// Every scalar keeps the type YAML resolved for it (a string that looks like a
// number stays a string), so the JSON imports back as the same configuration.
func configToJSON(cfg *config.Config) ([]byte, error) {
	var node yaml.Node
	if err := node.Encode(cfg); err != nil {
		return nil, fmt.Errorf("encode configuration: %w", err)
	}

	var compact bytes.Buffer
	if err := writeJSONNode(&compact, &node); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := json.Indent(&out, compact.Bytes(), "", "  "); err != nil {
		return nil, fmt.Errorf("indent JSON: %w", err)
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// writeJSONNode writes one YAML node as compact JSON.
func writeJSONNode(buf *bytes.Buffer, n *yaml.Node) error {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) != 1 {
			return fmt.Errorf("%w: a document with %d nodes", ErrUnsupportedYAMLNode, len(n.Content))
		}
		return writeJSONNode(buf, n.Content[0])
	case yaml.MappingNode:
		return writeJSONMapping(buf, n)
	case yaml.SequenceNode:
		return writeJSONSequence(buf, n)
	case yaml.ScalarNode:
		return writeJSONScalar(buf, n)
	case yaml.AliasNode:
		return writeJSONNode(buf, n.Alias)
	default:
		return fmt.Errorf("%w: kind %d", ErrUnsupportedYAMLNode, n.Kind)
	}
}

// writeJSONMapping writes a mapping as a JSON object, keeping its key order.
func writeJSONMapping(buf *bytes.Buffer, n *yaml.Node) error {
	buf.WriteByte('{')
	for i := 0; i+1 < len(n.Content); i += 2 {
		if i > 0 {
			buf.WriteByte(',')
		}
		key := n.Content[i]
		if key.Kind != yaml.ScalarNode {
			return fmt.Errorf("%w: a mapping key of kind %d", ErrUnsupportedYAMLNode, key.Kind)
		}
		if err := writeJSONString(buf, key.Value); err != nil {
			return err
		}
		buf.WriteByte(':')
		if err := writeJSONNode(buf, n.Content[i+1]); err != nil {
			return err
		}
	}
	buf.WriteByte('}')
	return nil
}

// writeJSONSequence writes a sequence as a JSON array.
func writeJSONSequence(buf *bytes.Buffer, n *yaml.Node) error {
	buf.WriteByte('[')
	for i, item := range n.Content {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := writeJSONNode(buf, item); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

// writeJSONScalar writes a scalar as the JSON value of the type YAML
// resolved for it.
func writeJSONScalar(buf *bytes.Buffer, n *yaml.Node) error {
	switch n.ShortTag() {
	case "!!str":
		return writeJSONString(buf, n.Value)
	case "!!null":
		buf.WriteString("null")
	case "!!bool":
		b, err := strconv.ParseBool(n.Value)
		if err != nil {
			return fmt.Errorf("%w: boolean %q", ErrUnsupportedYAMLNode, n.Value)
		}
		buf.WriteString(strconv.FormatBool(b))
	case "!!int", "!!float":
		if !isJSONNumber(n.Value) {
			return fmt.Errorf("%w: %s %q has no JSON form", ErrUnsupportedYAMLNode, n.ShortTag(), n.Value)
		}
		buf.WriteString(n.Value)
	default:
		return fmt.Errorf("%w: tag %s", ErrUnsupportedYAMLNode, n.ShortTag())
	}
	return nil
}

// writeJSONString writes s as a JSON string, without escaping HTML
// characters (the document is data, not markup).
func writeJSONString(buf *bytes.Buffer, s string) error {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return fmt.Errorf("encode JSON string: %w", err)
	}
	buf.Truncate(buf.Len() - 1) // Encode ends every value with a newline
	return nil
}

// isJSONNumber reports whether s is a number JSON can carry as written
// (not YAML's .inf, .nan, hex or octal forms).
func isJSONNumber(s string) bool {
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return false
	}
	return json.Valid([]byte(s))
}
