package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
)

func main() {
	conf, err := ParseArgs(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
		return
	}

	var l zerolog.Logger
	if conf.LogJson {
		l = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		l = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, NoColor: conf.NoColor}).With().Timestamp().Logger()
	}

	if conf.Store == nil {
		conf.Store = &StoreConfig{
			Kind: "files",
			Path: "vcpkg-cache",
			Opts: map[string]string{},
		}
		l.Info().Str("store", conf.Store.String()).Msg("use default store")
	}

	store, err := NewStore(conf.Store)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to initialize a store")
		return
	}

	addr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
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
