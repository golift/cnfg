package cnfg

import (
	"reflect"
	"strings"
)

// Pairs represents pairs of environment variables.
type Pairs map[string]string

const pairSize = 2

// Get allows getting only specific env variables by prefix.
// The prefix is trimmed before returning.
func (p *Pairs) Get(prefix string) Pairs {
	m := make(Pairs)

	for k, v := range *p {
		if strings.HasPrefix(k, prefix) {
			m[strings.SplitN(strings.TrimPrefix(k, prefix+LevelSeparator), LevelSeparator, pairSize)[0]] = v
		}
	}

	return m
}

func (p Pairs) Set(k, v string) {
	p[k] = v
}

func (p Pairs) Merge(pairs Pairs) {
	for k, v := range pairs {
		p[k] = v
	}
}

// UnmarshalMap parses and processes a map of key/value pairs as though they
// were environment variables. Useful for testing, or unmarshaling values
// from places other than environment variables.
// This version of UnmarshalMap assumes default tag ("xml") and no prefix: "".
func UnmarshalMap(pairs map[string]string, i interface{}) (bool, error) {
	return (&ENV{Tag: ENVTag}).UnmarshalMap(pairs, i)
}

// UnmarshalMap parses and processes a map of key/value pairs as though they
// were environment variables. Useful for testing, or unmarshaling values
// from places other than environment variables.
// Use this version of UnmarshalMap if you need to change the tag or prefix.
func (e *ENV) UnmarshalMap(pairs map[string]string, i interface{}) (bool, error) {
	value := reflect.ValueOf(i)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return false, ErrInvalidInterface
	}

	if e.Tag == "" {
		e.Tag = ENVTag
	}

	return (&parser{Tag: e.Tag, Vals: pairs}).Struct(value, e.Pfx)
}

// MapEnvPairs turns the pairs returned by os.Environ() into a map[string]string.
// Providing a prefix returns only variables with that prefix.
func MapEnvPairs(prefix string, pairs []string) Pairs {
	m := make(Pairs)

	for _, pair := range pairs {
		split := strings.SplitN(pair, "=", pairSize)
		if len(split) == pairSize && (prefix == "" || strings.HasPrefix(split[0], prefix)) {
			m[split[0]] = split[1]
		}
	}

	return m
}

func (p Pairs) Slice() []string {
	output := make([]string, len(p))
	i := 0

	for k, v := range p {
		output[i] = k + "=" + v
		i++
	}

	return output
}
