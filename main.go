package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

func ParseBackend(s string) (Store, error) {
	entries := strings.SplitN(s, ":", 2)
	if len(entries) != 2 {
		return nil, fmt.Errorf("backend format must be \"kind:path[,arg[=value]]...\"")
	}

	kind := entries[0]
	entries = strings.SplitN(entries[1], ",", 2)
	path := entries[0]
	switch kind {
	case "file":
		return NewFsStore(path)

	default:
		return nil, fmt.Errorf("kind not supported: %s", kind)
	}
}

func main() {
	var (
		host string
		port uint

		no_color bool
		log_json bool
	)

	flag.StringVar(&host, "host", "0.0.0.0", "host to listen")
	flag.UintVar(&port, "port", uint(15151), "port to listen")
	flag.BoolVar(&no_color, "no-color", !isatty.IsTerminal(os.Stdout.Fd()), "disable color print; set by default if output is not a terminal")
	flag.BoolVar(&log_json, "log-json", false, "log in JSON format")
	flag.Parse()

	var l zerolog.Logger
	if log_json {
		l = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		l = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, NoColor: no_color}).With().Timestamp().Logger()
	}

	backend := ""
	switch flag.NArg() {
	case 0:
		backend = "file:vcpkg-cache"
		l.Warn().Str("backend", backend).Msg("use default backend")
	case 1:
		backend = flag.Args()[0]

	default:
		l.Fatal().Msg("expected only 1 positional argument")
	}

	store, err := ParseBackend(backend)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to parse backend string")
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	server := &http.Server{
		Addr: addr,
		Handler: &Handler{
			Store: store,
			Log:   l,
		},
	}

	l.Info().Str("addr", addr).Msg("start server")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}
