package crud

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Crudify []string `json:"crudify,omitempty"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := &Config{
		Crudify:    []string{},
	}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}