package cnfg

import (
	"fmt"
	"reflect"
	"strings"
)

// Pairs represents pairs of environment variables.
type Pairs map[string]string

// Get allows getting only specific env variables by prefix.
// The prefix is trimmed before returning.
func (p *Pairs) Get(prefix string) Pairs {
	m := make(Pairs)

	for k, v := range *p {
		if strings.HasPrefix(k, prefix) {
			m[strings.Split(strings.TrimPrefix(k, prefix+"_"), "_")[0]] = v
		}
	}

	return m
}

// UnmarshalMap parses and processes a map of key/value pairs as though they
// were environment variables. Useful for testing, or unmarshaling values
// from places other than environment variables.
// This version of UnmarshalMap assumes default tag ("xml") and no prefix: ""
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
		return false, fmt.Errorf("can only unmarshal into pointer to struct")
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
		split := strings.SplitN(pair, "=", 2)
		if len(split) == 2 && (prefix == "" || strings.HasPrefix(split[0], prefix)) {
			m[split[0]] = split[1]
		}
	}

	return m
}
