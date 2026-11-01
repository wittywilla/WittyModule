package service

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// What crosses between a server and a service is every field of a value, which isn't what its JSON
// form says: `json:"-"` keeps a field off the API and out of a browser, such as where a core is on
// disk, and the server still needs it. So values are written here by their Go field names, in JSON,
// with only what can't travel left out: functions, channels, and live things such as a logger.

var (
	jsonMarshaler = reflect.TypeFor[json.Marshaler]()
	jsonUnmarshal = reflect.TypeFor[json.Unmarshaler]()
	textMarshaler = reflect.TypeFor[encoding.TextMarshaler]()
	textUnmarshal = reflect.TypeFor[encoding.TextUnmarshaler]()
)

// Marshal writes a value for the other side.
func Marshal(v any) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	var b bytes.Buffer
	if err := write(&b, reflect.ValueOf(v), 0); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Unmarshal reads what Marshal wrote.
func Unmarshal(data []byte, v any) error {
	p := reflect.ValueOf(v)
	if p.Kind() != reflect.Pointer || p.IsNil() {
		return fmt.Errorf("service: can't read into a %T", v)
	}
	return read(data, p.Elem(), 0)
}

// maxDepth is deeper than any value here, and stops one that contains itself.
const maxDepth = 64

// plain says whether a type can travel at all: data, with nothing live in it. A field hidden from
// JSON travels only if it can, since hiding is also how a type marks what's live.
func plain(t reflect.Type, seen map[reflect.Type]bool) bool {
	if seen[t] {
		return true
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.Func, reflect.Chan, reflect.Interface, reflect.UnsafePointer, reflect.Uintptr:
		return false
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return plain(t.Elem(), seen)
	case reflect.Map:
		return plain(t.Key(), seen) && plain(t.Elem(), seen)
	case reflect.Struct:
		if t.Implements(jsonMarshaler) || reflect.PointerTo(t).Implements(jsonMarshaler) || t.Implements(textMarshaler) || reflect.PointerTo(t).Implements(textMarshaler) {
			return true
		}
		for i := range t.NumField() {
			f := t.Field(i)
			if f.IsExported() && !plain(f.Type, seen) {
				return false
			}
			if !f.IsExported() && !f.Anonymous {
				return false // something with a state of its own, such as a logger
			}
		}
	}
	return true
}

// travels says whether a struct's field goes across.
func travels(f reflect.StructField) bool {
	if !f.IsExported() {
		return false
	}
	switch f.Type.Kind() {
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return false
	}
	if name, _, _ := strings.Cut(f.Tag.Get("json"), ","); name == "-" {
		return plain(f.Type, map[reflect.Type]bool{})
	}
	return true
}

func leaf(t reflect.Type) bool {
	return t.Implements(jsonMarshaler) || reflect.PointerTo(t).Implements(jsonMarshaler)
}

func write(b *bytes.Buffer, v reflect.Value, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("service: a %s is too deep to send", v.Type())
	}
	t := v.Type()
	switch t.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			b.WriteString("null")
			return nil
		}
	}
	if t.Kind() != reflect.Pointer && t.Kind() != reflect.Interface && (leaf(t) || t.Implements(textMarshaler) || reflect.PointerTo(t).Implements(textMarshaler)) {
		if !v.CanAddr() {
			c := reflect.New(t).Elem()
			c.Set(v)
			v = c
		}
		return viaJSON(b, v.Addr().Interface())
	}
	switch t.Kind() {
	case reflect.Pointer:
		return write(b, v.Elem(), depth+1)
	case reflect.Interface:
		// What's behind an interface has no type on the other side, so its ordinary JSON is all that can go.
		return viaJSON(b, v.Interface())
	case reflect.Struct:
		b.WriteByte('{')
		first := true
		if err := writeFields(b, v, &first, depth); err != nil {
			return err
		}
		b.WriteByte('}')
		return nil
	case reflect.Slice:
		if v.IsNil() {
			b.WriteString("null")
			return nil
		}
		if t.Elem().Kind() == reflect.Uint8 {
			return viaJSON(b, v.Interface())
		}
		fallthrough
	case reflect.Array:
		b.WriteByte('[')
		for i := range v.Len() {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := write(b, v.Index(i), depth+1); err != nil {
				return err
			}
		}
		b.WriteByte(']')
		return nil
	case reflect.Map:
		if v.IsNil() {
			b.WriteString("null")
			return nil
		}
		b.WriteByte('{')
		it := v.MapRange()
		first := true
		for it.Next() {
			if !first {
				b.WriteByte(',')
			}
			first = false
			k, err := mapKey(it.Key())
			if err != nil {
				return err
			}
			name, _ := json.Marshal(k)
			b.Write(name)
			b.WriteByte(':')
			if err := write(b, it.Value(), depth+1); err != nil {
				return err
			}
		}
		b.WriteByte('}')
		return nil
	}
	return viaJSON(b, v.Interface())
}

// flattened says whether a field's fields are written as if they were the struct's own, as JSON does
// with an embedded struct.
func flattened(f reflect.StructField) bool {
	return f.Anonymous && f.Type.Kind() == reflect.Struct && !leaf(f.Type)
}

func writeFields(b *bytes.Buffer, v reflect.Value, first *bool, depth int) error {
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		if flattened(f) {
			if err := writeFields(b, v.Field(i), first, depth+1); err != nil {
				return err
			}
			continue
		}
		if !travels(f) {
			continue
		}
		if !*first {
			b.WriteByte(',')
		}
		*first = false
		name, _ := json.Marshal(f.Name)
		b.Write(name)
		b.WriteByte(':')
		if err := write(b, v.Field(i), depth+1); err != nil {
			return err
		}
	}
	return nil
}

func readFields(fields map[string]json.RawMessage, v reflect.Value, depth int) error {
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		if flattened(f) {
			if err := readFields(fields, v.Field(i), depth+1); err != nil {
				return err
			}
			continue
		}
		if !travels(f) {
			continue
		}
		raw, ok := fields[f.Name]
		if !ok {
			// A type with a form of its own on the sending side only.
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			raw, ok = fields[name]
		}
		if !ok {
			continue
		}
		if err := read(raw, v.Field(i), depth+1); err != nil {
			return err
		}
	}
	return nil
}

func viaJSON(b *bytes.Buffer, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b.Write(raw)
	return nil
}

func mapKey(k reflect.Value) (string, error) {
	if m, ok := k.Interface().(encoding.TextMarshaler); ok {
		b, err := m.MarshalText()
		return string(b), err
	}
	switch k.Kind() {
	case reflect.String:
		return k.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(k.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(k.Uint(), 10), nil
	}
	return "", fmt.Errorf("service: a map keyed by %s can't be sent", k.Type())
}

func read(data []byte, v reflect.Value, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("service: a %s is too deep to read", v.Type())
	}
	t := v.Type()
	if isNull(data) {
		// As encoding/json does: a struct stays as it was, so a receiver isn't wiped by an empty answer.
		switch t.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
			v.SetZero()
		}
		return nil
	}
	if t.Kind() != reflect.Pointer && t.Kind() != reflect.Interface {
		p := reflect.PointerTo(t)
		if p.Implements(jsonUnmarshal) || p.Implements(textUnmarshal) {
			return json.Unmarshal(data, v.Addr().Interface())
		}
	}
	switch t.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(t.Elem()))
		}
		return read(data, v.Elem(), depth+1)
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return fmt.Errorf("a %s: %w", t, err)
		}
		return readFields(fields, v, depth)
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return json.Unmarshal(data, v.Addr().Interface())
		}
		var list []json.RawMessage
		if err := json.Unmarshal(data, &list); err != nil {
			return fmt.Errorf("a %s: %w", t, err)
		}
		out := reflect.MakeSlice(t, len(list), len(list))
		for i, raw := range list {
			if err := read(raw, out.Index(i), depth+1); err != nil {
				return err
			}
		}
		v.Set(out)
		return nil
	case reflect.Array:
		var list []json.RawMessage
		if err := json.Unmarshal(data, &list); err != nil {
			return fmt.Errorf("a %s: %w", t, err)
		}
		for i, raw := range list {
			if i >= v.Len() {
				break
			}
			if err := read(raw, v.Index(i), depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return fmt.Errorf("a %s: %w", t, err)
		}
		out := reflect.MakeMapWithSize(t, len(fields))
		for name, raw := range fields {
			k := reflect.New(t.Key()).Elem()
			if err := setKey(k, name); err != nil {
				return err
			}
			e := reflect.New(t.Elem()).Elem()
			if err := read(raw, e, depth+1); err != nil {
				return err
			}
			out.SetMapIndex(k, e)
		}
		v.Set(out)
		return nil
	case reflect.Interface:
		if t.NumMethod() > 0 {
			return nil // something live, which can't have come from the other side
		}
	}
	return json.Unmarshal(data, v.Addr().Interface())
}

func isNull(data []byte) bool { return string(bytes.TrimSpace(data)) == "null" }

func setKey(k reflect.Value, s string) error {
	if u, ok := k.Addr().Interface().(encoding.TextUnmarshaler); ok {
		return u.UnmarshalText([]byte(s))
	}
	switch k.Kind() {
	case reflect.String:
		k.SetString(s)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		k.SetInt(n)
		return err
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		k.SetUint(n)
		return err
	}
	return fmt.Errorf("service: a map keyed by %s can't be read", k.Type())
}
