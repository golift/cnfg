package cnfg

import (
	"encoding"
	"os"
	"reflect"
	"slices"
	"strings"
	"unicode"
)

// Map keys are recovered by peeling a typed suffix from each child env name.
// The first (shortest) boundary whose tail is a complete path down the value
// type wins, so nested fields that reuse a tag do not become phantom keys.
// Pairs.Get is not used: it treats the exact prefix as a match and can only
// see the first "_" token, so keys such as read_only never round-trip.

type envField struct {
	tokens []string
	typ    reflect.Type
}

func derefType(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	return typ
}

func pointerTo(typ reflect.Type) reflect.Type {
	if typ.Kind() == reflect.Pointer {
		return typ
	}

	return reflect.PointerTo(typ)
}

func implementsENVUnmarshaler(typ reflect.Type) bool {
	typ = derefType(typ)
	if typ == nil {
		return false
	}

	return pointerTo(typ).Implements(reflect.TypeFor[ENVUnmarshaler]())
}

func implementsEnvScalar(typ reflect.Type) bool {
	typ = derefType(typ)
	if typ == nil {
		return false
	}

	ptr := pointerTo(typ)

	return ptr.Implements(reflect.TypeFor[ENVUnmarshaler]()) ||
		ptr.Implements(reflect.TypeFor[encoding.TextUnmarshaler]()) ||
		ptr.Implements(reflect.TypeFor[encoding.BinaryUnmarshaler]())
}

func isByteSlice(typ reflect.Type) bool {
	// parser.Slice only treats the unnamed []uint8 / []byte type as a blob.
	return typ != nil && typ.String() == "[]uint8"
}

func isScalarEnvType(typ reflect.Type) bool {
	typ = derefType(typ)
	if typ == nil || implementsEnvScalar(typ) || isByteSlice(typ) {
		return true
	}

	switch typ.Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return false
	default:
		return true
	}
}

func isIndexedType(typ reflect.Type) bool {
	typ = derefType(typ)
	if typ == nil || isByteSlice(typ) || isScalarEnvType(typ) {
		return false
	}

	kind := typ.Kind()

	return kind == reflect.Slice || kind == reflect.Array
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	return true
}

func envFieldName(field reflect.StructField, tag string, low bool) (string, bool, bool) {
	if !field.IsExported() {
		return "", false, true
	}

	name, _, _ := strings.Cut(field.Tag.Get(tag), ",")
	if name == "-" {
		return "", false, true
	}

	if field.Anonymous && name == "" {
		return "", true, false
	}

	if name == "" {
		name = field.Name
	}

	if !low {
		name = strings.ToUpper(name)
	}

	return name, false, false
}

func envFields(typ reflect.Type, tag string, low bool) []envField {
	return collectEnvFields(typ, tag, low, make(map[reflect.Type]struct{}))
}

func collectEnvFields(typ reflect.Type, tag string, low bool, seen map[reflect.Type]struct{}) []envField {
	typ = derefType(typ)
	if typ == nil || typ.Kind() != reflect.Struct {
		return nil
	}

	if _, dup := seen[typ]; dup {
		return nil
	}

	seen[typ] = struct{}{}

	var out []envField

	for idx := range typ.NumField() {
		field := typ.Field(idx)

		name, embed, skip := envFieldName(field, tag, low)
		if skip {
			continue
		}

		if embed {
			out = append(out, collectEnvFields(field.Type, tag, low, seen)...)

			continue
		}

		out = append(out, envField{
			tokens: strings.Split(name, LevelSeparator),
			typ:    field.Type,
		})
	}

	return out
}

func tokensPrefix(parts, prefix []string) bool {
	if len(prefix) == 0 || len(parts) < len(prefix) {
		return false
	}

	for idx, token := range prefix {
		if parts[idx] != token {
			return false
		}
	}

	return true
}

func fieldPathOK(typ reflect.Type, parts []string, tag string, low bool) bool {
	typ = derefType(typ)
	if len(parts) == 0 {
		return !isIndexedType(typ)
	}

	if isScalarEnvType(typ) {
		return false
	}

	switch typ.Kind() {
	case reflect.Struct:
		return structPathOK(typ, parts, tag, low)
	case reflect.Slice, reflect.Array:
		if !isDigits(parts[0]) {
			return false
		}

		return fieldPathOK(typ.Elem(), parts[1:], tag, low)
	case reflect.Map:
		return true
	default:
		return false
	}
}

func structPathOK(typ reflect.Type, parts []string, tag string, low bool) bool {
	for _, field := range envFields(typ, tag, low) {
		if !tokensPrefix(parts, field.tokens) {
			continue
		}

		if fieldPathOK(field.typ, parts[len(field.tokens):], tag, low) {
			return true
		}
	}

	return false
}

func peelMapKey(remainder string, valType reflect.Type, tag string, low bool) (string, bool) {
	if remainder == "" {
		return "", false
	}

	valType = derefType(valType)
	if isScalarEnvType(valType) {
		return remainder, true
	}

	if valType.Kind() == reflect.Map {
		key, _, _ := strings.Cut(remainder, LevelSeparator)

		return key, key != ""
	}

	parts := strings.Split(remainder, LevelSeparator)
	for idx := 1; idx <= len(parts); idx++ {
		if fieldPathOK(valType, parts[idx:], tag, low) {
			return strings.Join(parts[:idx], LevelSeparator), true
		}
	}

	return "", false
}

func dropUnmarshalerDescendants(keys []string, valType reflect.Type) []string {
	if !implementsENVUnmarshaler(valType) {
		return keys
	}

	out := make([]string, 0, len(keys))

	for _, key := range keys {
		skip := false

		for _, other := range keys {
			if other != key && strings.HasPrefix(key, other+LevelSeparator) {
				skip = true

				break
			}
		}

		if !skip {
			out = append(out, key)
		}
	}

	return out
}

func (p *parser) mapKeys(prefix string, valType reflect.Type) []string {
	child := prefix + LevelSeparator
	seen := make(map[string]struct{})

	var keys []string

	for name := range p.Vals {
		if !strings.HasPrefix(name, child) {
			continue
		}

		remainder := strings.TrimPrefix(name, child)
		key, matched := peelMapKey(remainder, valType, p.Tag, p.Low)

		if p.Vals[name] == "" && name == child+remainder &&
			(!matched || (isIndexedType(valType) && key != remainder)) {
			key = remainder
			matched = true
		}

		if !matched {
			continue
		}

		if _, dup := seen[key]; dup {
			continue
		}

		seen[key] = struct{}{}

		keys = append(keys, key)
	}

	keys = dropUnmarshalerDescendants(keys, valType)
	slices.Sort(keys)

	return keys
}

func (p *parser) unsetMapEnv(exact string) {
	_ = os.Unsetenv(exact)

	prefix := exact + LevelSeparator
	for name := range p.Vals {
		if name == exact || strings.HasPrefix(name, prefix) {
			_ = os.Unsetenv(name)
		}
	}
}

func (p *parser) setMapEntry(field reflect.Value, tag, key string, delenv bool) (bool, error) {
	exact := strings.Join([]string{tag, key}, LevelSeparator)
	val, hasExact := p.Vals[exact]

	if delenv {
		p.unsetMapEnv(exact)
	}

	keyval := reflect.Indirect(reflect.New(field.Type().Key()))
	if _, err := p.Anything(keyval, tag, key, true, false); err != nil {
		return false, err
	}

	if hasExact && val == "" {
		field.SetMapIndex(keyval, reflect.Value{})

		return true, nil
	}

	valval := reflect.Indirect(reflect.New(field.Type().Elem()))

	exists, err := p.Anything(valval, exact, val, hasExact, delenv)
	if err != nil || !exists {
		return exists, err
	}

	field.SetMapIndex(keyval, valval)

	return true, nil
}
