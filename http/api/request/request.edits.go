package request

import (
	"encoding/json"
	"reflect"

	"tty.fail/mrus/korda/models/assetedit"
)

var (
	editType  = reflect.TypeFor[assetedit.Input]()
	editsType = reflect.TypeFor[assetedit.Inputs]()

	editOptions = []struct {
		action assetedit.Action
		t      reflect.Type
		keys   string
	}{
		{assetedit.ActionCrop, reflect.TypeFor[assetedit.Crop](), "x, y, width, height"},
		{assetedit.ActionRotate, reflect.TypeFor[assetedit.Rotate](), "angle"},
		{assetedit.ActionMirror, reflect.TypeFor[assetedit.Mirror](), "axis"},
	}
)

const editUnionMessage = "Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"

func withPrefix(iss Issue, prefix []any) Issue {
	out := Issue{keys: iss.keys, values: append([]any(nil), iss.values...), abort: iss.abort}
	for i, k := range out.keys {
		if k == "path" {
			out.values[i] = append(append([]any{}, prefix...), out.values[i].([]any)...)
		}
	}

	return out
}

func (w *walker) sub(raw any, present bool, t reflect.Type) (any, []Issue) {
	s := &walker{mode: w.mode}
	v, _ := s.object([]any{}, raw, present, t)
	if s.err != nil && w.err == nil {
		w.err = s.err
	}

	return v, s.issues
}

func aborts(issues []Issue) bool {
	for _, iss := range issues {
		if iss.abort {
			return true
		}
	}

	return false
}

func (w *walker) editParameters(path []any, raw any, present bool) any {
	type result struct {
		value  any
		issues []Issue
	}

	results := make([]result, 0, len(editOptions))
	for _, option := range editOptions {
		v, issues := w.sub(raw, present, option.t)
		if len(issues) == 0 {
			return v
		}
		results = append(results, result{v, issues})
	}

	var open []result
	for _, r := range results {
		if !aborts(r.issues) {
			open = append(open, r)
		}
	}
	if len(open) == 1 {
		for _, iss := range open[0].issues {
			w.issues = append(w.issues, withPrefix(iss, path))
		}
		return open[0].value
	}

	errors := make([][]Issue, 0, len(results))
	for _, r := range results {
		errors = append(errors, r.issues)
	}
	w.issues = append(w.issues, newIssue("code", "invalid_union", "errors", errors).withPath(path, editUnionMessage).aborting())

	return nil
}

func (w *walker) edit(path []any, raw any, present bool) (any, bool) {
	m, ok := raw.(*Object)
	if !ok || !present {
		iss, message := invalidType("object", received(raw, present))
		w.report(path, iss, message)
		return nil, false
	}

	start := len(w.issues)
	out := NewObject()

	rawAction, hasAction := m.Get("action")
	action, _ := w.enum(join(path, "action"), rawAction, hasAction, assetedit.Action("").Enum())
	if action != nil {
		out.Set("action", action)
	}

	rawParameters, hasParameters := m.Get("parameters")
	parameters := w.editParameters(join(path, "parameters"), rawParameters, hasParameters)
	if parameters != nil {
		out.Set("parameters", parameters)
	}

	if !w.abortedSince(start) {
		encoded, _ := json.Marshal(parameters)
		reread, _, _ := decodeJSON(encoded)
		for _, option := range editOptions {
			if string(option.action) != action {
				continue
			}
			if _, issues := w.sub(reread, true, option.t); len(issues) > 0 {
				w.issues = append(w.issues, Custom(join(path, "parameters"),
					"Invalid parameters for action '"+string(option.action)+"', expecting keys: "+option.keys))
			}
		}
	}

	return out, true
}

func (w *walker) edits(path []any, raw any, present bool, r rules) (any, bool) {
	start := len(w.issues)
	v, keep := w.slice(path, raw, present, editsType, r, rules{})
	if !keep || w.abortedSince(start) {
		return v, keep
	}

	seen := map[string]bool{}
	for _, item := range v.([]any) {
		o := item.(*Object)
		action, _ := o.Get("action")
		key, _ := action.(string)
		if key == string(assetedit.ActionMirror) {
			parameters, _ := o.Get("parameters")
			encoded, _ := json.Marshal(parameters)
			key = "mirror-" + string(encoded)
		}
		if seen[key] {
			w.issues = append(w.issues, Custom(path, "Duplicate edit actions are not allowed"))
			break
		}
		seen[key] = true
	}

	return v, keep
}

func angle(raw any) string {
	if n, ok := raw.(json.Number); ok {
		if f, err := n.Float64(); err == nil && (f == 0 || f == 90 || f == 180 || f == 270) {
			return ""
		}
	}

	return "Angle must be one of the following values: 0, 90, 180, 270"
}
