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
// Pairs.Get is not used: it treats the exact prefix as a match and can only
// see the first "_" token, so keys such as read_only never round-trip.

func derefType(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	return typ
}

func implementsEnvScalar(typ reflect.Type) bool {
	if typ == nil {
		return false
	}

	ptr := typ
	if typ.Kind() != reflect.Pointer {
		ptr = reflect.PointerTo(typ)
	}

	return ptr.Implements(reflect.TypeFor[ENVUnmarshaler]()) ||
		ptr.Implements(reflect.TypeFor[encoding.TextUnmarshaler]()) ||
		ptr.Implements(reflect.TypeFor[encoding.BinaryUnmarshaler]())
}

func isByteSlice(typ reflect.Type) bool {
	return typ != nil && typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8
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
	if !field.IsExported() && !field.Anonymous {
		return "", false, true
	}

	name, _, _ := strings.Cut(field.Tag.Get(tag), ",")
	if name == "-" {
		return "", false, true
	}

	if field.Anonymous && (name == "" || name == field.Name) {
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

func structFieldTags(typ reflect.Type, tag string, low bool) []string {
	typ = derefType(typ)
	if typ == nil || typ.Kind() != reflect.Struct {
		return nil
	}

	var out []string

	for idx := range typ.NumField() {
		field := typ.Field(idx)

		name, embed, skip := envFieldName(field, tag, low)
		if skip {
			continue
		}

		if embed {
			out = append(out, structFieldTags(field.Type, tag, low)...)

			continue
		}

		out = append(out, name)
	}

	return out
}

func structFieldTagSet(typ reflect.Type, tag string, low bool) map[string]struct{} {
	out := make(map[string]struct{})
	for _, name := range structFieldTags(typ, tag, low) {
		out[name] = struct{}{}
	}

	return out
}

func isStructFieldTag(typ reflect.Type, name, tag string, low bool) bool {
	_, ok := structFieldTagSet(typ, tag, low)[name]

	return ok
}

func peelStructMapKey(remainder string, typ reflect.Type, tag string, low bool) (string, bool) {
	fields := structFieldTagSet(typ, tag, low)
	parts := strings.Split(remainder, LevelSeparator)

	var best string

	for idx := 1; idx < len(parts); idx++ {
		if _, ok := fields[parts[idx]]; !ok {
			continue
		}

		key := strings.Join(parts[:idx], LevelSeparator)
		if len(key) >= len(best) {
			best = key
		}
	}

	return best, best != ""
}

func sliceIndexOK(elemType reflect.Type, after []string, tag string, low bool) bool {
	elemType = derefType(elemType)
	if isScalarEnvType(elemType) {
		return len(after) == 0
	}

	switch elemType.Kind() {
	case reflect.Struct:
		return len(after) == 0 || isStructFieldTag(elemType, after[0], tag, low)
	case reflect.Slice, reflect.Array:
		if len(after) == 0 {
			return true
		}

		return isDigits(after[0])
	case reflect.Map:
		return true
	default:
		return len(after) == 0
	}
}

func peelSliceMapKey(remainder string, elemType reflect.Type, tag string, low bool) (string, bool) {
	parts := strings.Split(remainder, LevelSeparator)

	for idx := 1; idx < len(parts); idx++ {
		if !isDigits(parts[idx]) {
			continue
		}

		if sliceIndexOK(elemType, parts[idx+1:], tag, low) {
			return strings.Join(parts[:idx], LevelSeparator), true
		}
	}

	return "", false
}

func peelMapKey(remainder string, valType reflect.Type, tag string, low bool) (string, bool) {
	if remainder == "" {
		return "", false
	}

	valType = derefType(valType)
	if isScalarEnvType(valType) {
		return remainder, true
	}

	switch valType.Kind() {
	case reflect.Struct:
		if key, ok := peelStructMapKey(remainder, valType, tag, low); ok {
			return key, true
		}

		return remainder, true
	case reflect.Slice, reflect.Array:
		return peelSliceMapKey(remainder, valType.Elem(), tag, low)
	case reflect.Map:
		key, _, _ := strings.Cut(remainder, LevelSeparator)

		return key, key != ""
	default:
		return remainder, true
	}
}

func (p *parser) mapKeys(prefix string, valType reflect.Type) []string {
	child := prefix + LevelSeparator
	seen := make(map[string]struct{})

	var keys []string

	for name := range p.Vals {
		if !strings.HasPrefix(name, child) {
			continue
		}

		key, ok := peelMapKey(strings.TrimPrefix(name, child), valType, p.Tag, p.Low)
		if !ok {
			continue
		}

		if _, dup := seen[key]; dup {
			continue
		}

		seen[key] = struct{}{}

		keys = append(keys, key)
	}

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
