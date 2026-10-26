package view

import (
	"bytes"
	"encoding/json"
	"slices"

	logicsystemconfig "tty.fail/mrus/korda/logic/root/systemconfig"
)

type node struct {
	keys   []string
	values map[string]*node
	raw    json.RawMessage
}

func parseNode(dec *json.Decoder) (*node, error) {
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 || raw[0] != '{' {
		return &node{raw: raw}, nil
	}

	inner := json.NewDecoder(bytes.NewReader(raw))
	inner.UseNumber()
	if _, err := inner.Token(); err != nil {
		return nil, err
	}
	n := &node{values: map[string]*node{}}
	for inner.More() {
		tok, err := inner.Token()
		if err != nil {
			return nil, err
		}
		key := tok.(string)
		child, err := parseNode(inner)
		if err != nil {
			return nil, err
		}
		n.keys = append(n.keys, key)
		n.values[key] = child
	}

	return n, nil
}

func parseJSON(b []byte) (*node, error) {
	return parseNode(json.NewDecoder(bytes.NewReader(b)))
}

func (n *node) write(buf *bytes.Buffer, order *node) {
	if n.values == nil {
		buf.Write(n.raw)
		return
	}

	keys := n.keys
	if order != nil && order.values != nil {
		keys = slices.Clone(order.keys)
		for _, k := range n.keys {
			if !slices.Contains(keys, k) {
				keys = append(keys, k)
			}
		}
	}

	buf.WriteByte('{')
	first := true
	for _, k := range keys {
		child, ok := n.values[k]
		if !ok {
			continue
		}
		if !first {
			buf.WriteByte(',')
		}
		first = false
		encoded, _ := json.Marshal(k)
		buf.Write(encoded)
		buf.WriteByte(':')
		var sub *node
		if order != nil && order.values != nil {
			sub = order.values[k]
		}
		child.write(buf, sub)
	}
	buf.WriteByte('}')
}

func (n *node) keep(prefix string, leaves []string) *node {
	if n.values == nil {
		if slices.Contains(leaves, prefix) {
			return n
		}
		return nil
	}

	out := &node{values: map[string]*node{}}
	for _, k := range n.keys {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		if child := n.values[k].keep(path, leaves); child != nil {
			out.keys = append(out.keys, k)
			out.values[k] = child
		}
	}
	if len(out.keys) == 0 {
		return nil
	}

	return out
}

func encode(cfg *logicsystemconfig.Config) (*node, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	return parseJSON(b)
}

func ConfigBody(cfg *logicsystemconfig.Config) ([]byte, error) {
	return json.Marshal(cfg)
}

func ConfigDefaultsBody(cfg *logicsystemconfig.Config) ([]byte, error) {
	n, err := encode(cfg)
	if err != nil {
		return nil, err
	}
	order, err := parseJSON(logicsystemconfig.DefaultsOrder)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	n.write(&buf, order)

	return buf.Bytes(), nil
}

func ConfigSubsetBody(cfg *logicsystemconfig.Config, leaves []string) ([]byte, error) {
	n, err := encode(cfg)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if kept := n.keep("", leaves); kept != nil {
		kept.write(&buf, nil)
	} else {
		buf.WriteString("{}")
	}

	return buf.Bytes(), nil
}
