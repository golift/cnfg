package cnfgfile

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"strings"

	toml "github.com/pelletier/go-toml"
	yaml "gopkg.in/yaml.v2"
)

// Unmarshal parses a configuration file (of any format) into a config struct.
// This is a shorthand method for calling Unmarshal against the json, xml, yaml
// or toml packages. If the file name contains an appropriate suffix it is
// unmarshaled with the corresponding package. If the suffix is missing, TOML
// is assumed. Works with multiple files, so you can have stacked configurations.
func Unmarshal(c interface{}, configFile ...string) error {
	if len(configFile) < 1 {
		return fmt.Errorf("must provide at least 1 file to unmarshal")
	}

	for _, f := range configFile {
		buf, err := ioutil.ReadFile(f)

		switch {
		case err != nil:
			return err
		case strings.Contains(f, ".json"):
			err = json.Unmarshal(buf, c)
		case strings.Contains(f, ".xml"):
			err = xml.Unmarshal(buf, c)
		case strings.Contains(f, ".yaml"):
			err = yaml.Unmarshal(buf, c)
		default:
			err = toml.Unmarshal(buf, c)
		}

		if err != nil {
			return err
		}
	}

	return nil
}
