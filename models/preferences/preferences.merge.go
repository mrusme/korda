package preferences

import (
	"encoding/json"
	"reflect"
	"strings"
)

func Merge(p Preferences, u Update) Preferences {
	apply(reflect.ValueOf(&p).Elem(), reflect.ValueOf(u))
	return p
}

func FromStored(stored string) (Preferences, error) {
	p := Defaults()
	if stored == "" {
		return p, nil
	}

	var u Update
	if err := json.Unmarshal([]byte(stored), &u); err != nil {
		return p, err
	}

	return Merge(p, u), nil
}

func Stored(p Preferences) (string, error) {
	var u Update
	diff(reflect.ValueOf(&u).Elem(), reflect.ValueOf(p), reflect.ValueOf(Defaults()))

	encoded, err := json.Marshal(u)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}

func apply(dst reflect.Value, src reflect.Value) {
	for i := range src.NumField() {
		field := src.Field(i)
		if field.Kind() != reflect.Pointer || field.IsNil() {
			continue
		}

		value := field.Elem()
		if value.Kind() == reflect.Struct && !hasLeaves(value) {
			continue
		}

		target, ok := fieldByJSON(dst, jsonName(src.Type().Field(i)))
		if !ok {
			continue
		}
		if target.Kind() == reflect.Pointer {
			if target.IsNil() {
				target.Set(reflect.New(target.Type().Elem()))
			}
			target = target.Elem()
		}

		if value.Kind() == reflect.Struct {
			apply(target, value)
			continue
		}
		target.Set(value.Convert(target.Type()))
	}
}

func hasLeaves(v reflect.Value) bool {
	for _, field := range v.Fields() {
		if field.Kind() != reflect.Pointer || field.IsNil() {
			continue
		}
		if field.Elem().Kind() != reflect.Struct || hasLeaves(field.Elem()) {
			return true
		}
	}

	return false
}

func diff(dst reflect.Value, current reflect.Value, defaults reflect.Value) bool {
	changed := false

	for i := range dst.NumField() {
		field := dst.Field(i)
		name := jsonName(dst.Type().Field(i))

		cur, ok := fieldByJSON(current, name)
		if !ok || cur.Kind() == reflect.Pointer {
			continue
		}
		def, _ := fieldByJSON(defaults, name)

		elem := reflect.New(field.Type().Elem())
		if cur.Kind() == reflect.Struct {
			if diff(elem.Elem(), cur, def) {
				field.Set(elem)
				changed = true
			}
			continue
		}

		if cur.IsZero() && cur.Kind() == reflect.String {
			continue
		}
		if reflect.DeepEqual(cur.Interface(), def.Interface()) {
			continue
		}
		elem.Elem().Set(cur.Convert(elem.Elem().Type()))
		field.Set(elem)
		changed = true
	}

	return changed
}

func fieldByJSON(v reflect.Value, name string) (reflect.Value, bool) {
	for i := range v.NumField() {
		if jsonName(v.Type().Field(i)) == name {
			return v.Field(i), true
		}
	}

	return reflect.Value{}, false
}

func jsonName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	return name
}
