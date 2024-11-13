package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Path    string
	Targets []*Target
}

type Target struct {
	Name         string
	Uri          string
	Method       string
	Period       int
	ResponseCode int
}

const (
	DefaultPeriod       = 30
	DefaultResponseCode = 200
	DefaultMethod       = http.MethodHead
)

func ParseConfig(path string) (*Config, []error) {
	config := Config{Path: path}

	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, []error{fmt.Errorf("ParseConfig(%s): failed to parse: %v", path, err)}
	}

	if errs := validateConfig(&config); errs != nil {
		return nil, errs
	}

	return &config, nil
}

func validateConfig(config *Config) []error {
	var errs []error

	prefix := "validateConfig(" + config.Path + ")"

	if len(config.Targets) == 0 {
		errs = append(errs, fmt.Errorf("%s: at least one target must be specified", prefix))
	}

	for idx, target := range config.Targets {
		if strings.TrimSpace(target.Name) == "" {
			errs = append(errs, fmt.Errorf("%s: target at index %d must have a name", prefix, idx))
		}

		if strings.TrimSpace(target.Uri) == "" {
			errs = append(errs, fmt.Errorf("%s: target at index %d must have a uri", prefix, idx))
		}

		if target.ResponseCode == 0 {
			target.ResponseCode = DefaultResponseCode
		}

		if target.Period == 0 {
			target.Period = DefaultPeriod
		}

		if strings.TrimSpace(target.Method) == "" {
			target.Method = DefaultMethod
		}
	}

	return errs
}
