package cnfg

import (
	"fmt"
	"reflect"
	"strings"
)

// Pairs represents pairs of environment variables.
type Pairs map[string]string

// Filter allows getting only specific env variables by prefix.
// The prefix is trimmed before returning.
func (p *Pairs) Filter(prefix string) Pairs {
	m := make(Pairs)

	for k, v := range *p {
		if strings.HasPrefix(k, prefix) {
			t := strings.Split(strings.TrimPrefix(k, prefix+"_"), "_")[0]
			m[t] = v
		}
	}

	return m
}

// UnmarshalMap parses and processes a map of key/value pairs as though they
// were environment variables. Useful for testing, or unmarshaling values
// from places other than environment variables.
func UnmarshalMap(pairs map[string]string, i interface{}) (bool, error) {
	return (&ENV{Tag: ENVTag}).UnmarshalMap(pairs, i)
}

// UnmarshalMap parses and processes a map of key/value pairs as though they
// were environment variables. Useful for testing, or unmarshaling values
// from places other than environment variables.
func (e *ENV) UnmarshalMap(pairs map[string]string, i interface{}) (bool, error) {
	value := reflect.ValueOf(i)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return false, fmt.Errorf("can only unmarshal ENV into pointer to struct")
	}

	if e.Tag == "" {
		e.Tag = ENVTag
	}

	e.pairs = pairs

	return e.parseStruct(value, e.Pfx)
}

// MapEnvPairs turns the pairs returned by os.Environ() into a map[string]string.
// Providing a prefix returns only variables with that prefix.
func MapEnvPairs(prefix string, pairs []string) Pairs {
	m := make(Pairs)

	for _, pair := range pairs {
		split := strings.SplitN(pair, "=", 2)
		if len(split) != 2 {
			continue
		}

		fulltag := split[0]
		value := split[1]

		if prefix == "" || strings.HasPrefix(fulltag, prefix) {
			m[fulltag] = value
		}
	}

	return m
}
