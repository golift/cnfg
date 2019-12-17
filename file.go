package config

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"strings"

	"github.com/BurntSushi/toml"
	yaml "gopkg.in/yaml.v2"
)

// ParseFile parses a configuration file (of any format) into a config struct.
func ParseFile(c interface{}, configFile string) error {
	switch buf, err := ioutil.ReadFile(configFile); {
	case err != nil:
		return err
	case strings.Contains(configFile, ".json"):
		return json.Unmarshal(buf, c)
	case strings.Contains(configFile, ".xml"):
		return xml.Unmarshal(buf, c)
	case strings.Contains(configFile, ".yaml"):
		return yaml.Unmarshal(buf, c)
	default:
		return toml.Unmarshal(buf, c)
	}
}
