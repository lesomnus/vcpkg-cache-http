package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
)

func main() {
	conf, err := ParseArgsStrict(os.Args)
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
	defer func() {
		err := store.Close()
		if err != nil {
			l.Error().Err(err).Msg("failed to close the store")
		}
	}()

	handler := &Handler{
		Store: store,
		Log:   l,

		IsReadable: true,
		IsWritable: true,
	}

	if conf.ReadOnly {
		handler.IsWritable = false
		l.Info().Msg("upload disabled")
	} else if conf.WriteOnly {
		handler.IsReadable = false
		l.Info().Msg("download disabled")
	}

	addr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	server_closed := make(chan struct{})
	defer close(server_closed)

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)

		for {
			select {
			case <-server_closed:
				return

			case sig := <-signals:
				l.Warn().Str("signal", sig.String()).Msg("shutdown the server")
				if err := server.Shutdown(context.Background()); err != nil {
					l.Error().Err(err).Msg("failed to shutdown the server")
				}
			}
		}
	}()

	l.Info().Str("addr", addr).Msg("start server")
	if err := server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			l.Info().Msg("server closed gracefully")
		} else {
			l.Error().Err(err).Msg("unexpected server close")
		}
	}
}
