package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

type AppConfig struct {
	Host string `json:"host"`
	Port uint   `json:"port"`

	Store *StoreConfig `json:"store,omitempty"`

	NoColor bool `json:"no_color"`
	LogJson bool `json:"log_json"`
}

type StoreConfig struct {
	Kind string            `json:"kind"`
	Path string            `json:"path"`
	Opts map[string]string `json:"opts"`
}

func ParseStoreConfig(s string) (*StoreConfig, error) {
	entries := strings.SplitN(s, ":", 2)
	if entries[0] == "" {
		return nil, errors.New("kind must be specified")
	}

	if len(entries) != 2 {
		return &StoreConfig{
			Kind: entries[0],
			Opts: map[string]string{},
		}, nil
	}

	kind := entries[0]
	opts := strings.Split(entries[1], ",")

	conf := &StoreConfig{
		Kind: kind,
		Path: opts[0],
		Opts: map[string]string{},
	}

	for _, opt := range opts[1:] {
		kv := strings.SplitN(opt, "=", 2)
		if kv[0] == "" {
			continue
		}

		v := ""
		if len(kv) == 2 {
			v = kv[1]
		}

		conf.Opts[kv[0]] = v
	}

	return conf, nil
}

func NewStore(conf *StoreConfig) (Store, error) {
	switch conf.Kind {
	case "file":
		return NewFsStore(conf.Path)

	default:
		return nil, fmt.Errorf("kind not supported: %s", conf.Kind)
	}
}

func ParseArgs(args []string) (*AppConfig, error) {
	var (
		flags = flag.NewFlagSet(args[0], flag.ExitOnError)
		conf  = &AppConfig{}

		conf_path  = ""
		conf_given = AppConfig{}
	)

	flags.StringVar(&conf_path, "conf", "", "path to a config file")
	flags.StringVar(&conf_given.Host, "host", "0.0.0.0", "host to listen")
	flags.UintVar(&conf_given.Port, "port", uint(15151), "port to listen")
	flags.BoolVar(&conf_given.NoColor, "no-color", !isatty.IsTerminal(os.Stdout.Fd()), "disable color print; set by default if output is not a terminal")
	flags.BoolVar(&conf_given.LogJson, "log-json", false, "log in JSON format")
	flags.Parse(args[1:])

	switch flags.NArg() {
	case 0:
		break

	case 1:
		store, err := ParseStoreConfig(flags.Args()[0])
		if err != nil {
			return nil, fmt.Errorf("parse store config: %w", err)
		}

		conf.Store = store

	default:
		return nil, errors.New("expected only 1 positional argument")
	}

	if conf_path == "" {
		*conf = conf_given
		return conf, nil
	}

	data, err := os.ReadFile(conf_path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config at %s: %w", conf_path, err)
	}
	if err := json.Unmarshal(data, &conf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config at %s: %w", conf_path, err)
	}

	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "host":
			conf.Host = conf_given.Host
		case "port":
			conf.Port = conf_given.Port
		case "no-color":
			conf.NoColor = conf_given.NoColor
		case "log-json":
			conf.LogJson = conf_given.LogJson
		}
	})

	return conf, nil
}
