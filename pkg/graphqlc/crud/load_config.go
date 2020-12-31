package crud

import (
	"encoding/json"
	"io/ioutil"
)

func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	config := &Config{
		Types: make(map[string]TypeSpec),
	}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	//return nil, fmt.Errorf("%# v\n", pretty.Formatter(config))
	return config, nil
}
