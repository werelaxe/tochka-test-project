package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
)

type Config struct {
	PostgresConfig PostgresConfig
	Host           string
	Port           uint32
	TemplatesPath  string
	StaticPath     string
	AddExamples    bool
}

func ParseConfig(path string) (*Config, error) {
	var config Config
	rawConfig, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.New("file reading error: " + err.Error())
	}
	err = json.Unmarshal(rawConfig, &config)
	if err != nil {
		return nil, errors.New("unmarshalling error: " + err.Error())
	}
	return &config, nil
}
