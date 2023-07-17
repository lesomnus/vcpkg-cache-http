package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
)

type AppConfig struct {
	Host string `json:"host"`
	Port uint   `json:"port"`

	Store *StoreConfig `json:"store,omitempty"`

	NoColor bool `json:"no_color"`
	LogJson bool `json:"log_json"`

	ReadOnly  bool `json:"read_only"`
	WriteOnly bool `json:"write_only"`
}

type StoreConfig struct {
	Kind string            `json:"kind"`
	Path string            `json:"path"`
	Opts map[string]string `json:"opts"`
}

func (c *StoreConfig) String() string {
	rst := fmt.Sprintf("%s:%s", c.Kind, c.Path)
	for k, v := range c.Opts {
		if v == "" {
			rst += fmt.Sprintf(",%s", k)
		} else {
			rst += fmt.Sprintf(",%s=%s", k, v)
		}
	}

	return rst
}

func ParseStoreConfig(s string) (*StoreConfig, error) {
	entries := strings.SplitN(s, ":", 2)
	if entries[0] == "" {
		return nil, errors.New("kind must be specified")
	}

	if strings.ContainsAny(entries[0], ",=") {
		return nil, errors.New("invalid kind value")
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
	case "files":
		return NewFsStore(conf.Path)

	case "archives":
		p := conf.Path
		if p == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("get home directory: %w", err)
			}

			p = filepath.Join(home, ".cache", "vcpkg", "archives")
		}
		return NewFsStore(p, WithPathResolve(func(desc Description) string {
			return filepath.Join(desc.Hash[0:2], desc.Hash+".zip")
		}))

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

	flags.Usage = func() {
		fmt.Printf("Usage: %s [Flags] [Store]\n", args[0])

		fmt.Printf(`
Store:
  Specify the location to store the binary cache in the format:

    kind[:[path][,opt[=val]]]

  Available stores are:
    
    files:[vcpkg-cache]
      Stores to a directory at the given path. This is a default store.

    archives:[${HOME}/.cache/vcpkg/archives]
      Use vcpkg "files" provider at the given path as a store.

`)

		fmt.Println("Flags:")
		flags.PrintDefaults()
	}

	flags.StringVar(&conf_path, "conf", "", "path to a config file")
	flags.StringVar(&conf_given.Host, "host", "0.0.0.0", "host to listen")
	flags.UintVar(&conf_given.Port, "port", uint(15151), "port to listen")
	flags.BoolVar(&conf_given.NoColor, "no-color", !isatty.IsTerminal(os.Stdout.Fd()), "disable color print; set by default if output is not a terminal")
	flags.BoolVar(&conf_given.LogJson, "log-json", false, "log in JSON format")
	flags.BoolVar(&conf_given.ReadOnly, "read-only", false, "enable read-only mode, restricting write operations")
	flags.BoolVar(&conf_given.WriteOnly, "write-only", false, "enable write-only mode, restricting read operations")
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
		case "read-only":
			conf.ReadOnly = conf_given.ReadOnly
		case "write-only":
			conf.WriteOnly = conf_given.WriteOnly
		}
	})

	return conf, nil
}

func ParseArgsStrict(args []string) (*AppConfig, error) {
	conf, err := ParseArgs(args)
	if err != nil {
		return nil, err
	}

	if conf.ReadOnly && conf.WriteOnly {
		return nil, errors.New("read-only and write-only cannot be set together")
	}

	return conf, nil
}
