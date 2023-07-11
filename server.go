package main

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	Store Store
	Log   zerolog.Logger
}

func (s *Handler) handleGet(res http.ResponseWriter, req *http.Request, desc Description) {
	err := s.Store.Get(req.Context(), desc, res)
	if err == nil {
		return
	}

	l := log.Ctx(req.Context())
	if errors.Is(err, ErrNotExist) {
		l.Warn().Msg("not found")
		res.WriteHeader(http.StatusNotFound)
	} else {
		l.Error().Err(err).Msg("")
		res.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Handler) handlePut(res http.ResponseWriter, req *http.Request, desc Description) {
	err := s.Store.Put(req.Context(), desc, req.Body)
	if err == nil {
		res.WriteHeader(http.StatusOK)
		return
	}

	l := log.Ctx(req.Context())
	l.Warn().Msg(err.Error())
	res.WriteHeader(http.StatusInternalServerError)
}

type responseWriter struct {
	http.ResponseWriter
	status_code int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.status_code = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (s *Handler) parseDescription(res http.ResponseWriter, req *http.Request) (Description, error) {
	entries := strings.SplitN(req.URL.Path[1:], "/", 4)
	if len(entries) != 3 {
		res.WriteHeader(http.StatusNotFound)
		return Description{}, errors.New("invalid path")
	}

	switch req.Method {
	case http.MethodGet:
	case http.MethodPut:
		break

	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
		return Description{}, errors.New("invalid method")
	}

	return Description{
		Name:    entries[0],
		Version: entries[1],
		Hash:    entries[2],
	}, nil
}

func (s *Handler) ServeHTTP(r http.ResponseWriter, req *http.Request) {
	t0 := time.Now()
	res := &responseWriter{r, http.StatusOK}

	l := s.Log.With().Str("_", getTicket()).Logger()
	req = req.WithContext(l.WithContext(req.Context()))

	desc, err := s.parseDescription(res, req)
	{
		l := l.With().Str("url", req.URL.String()).Str("method", req.Method).Logger()
		if err != nil {
			l.Warn().Dur("dt", time.Since(t0)).Int("status", res.status_code).Msg("REQ " + err.Error())
			return
		}

		l.Info().Msg("")
	}

	l.Info().
		Str("name", desc.Name).
		Str("version", desc.Version).
		Str("hash", desc.Hash).
		Msg("REQ " + req.Method)

	switch req.Method {
	case http.MethodGet:
		s.handleGet(res, req, desc)

	case http.MethodPut:
		s.handlePut(res, req, desc)
	}

	l = l.With().Dur("dt", time.Since(t0)).Int("status", res.status_code).Logger()
	msg := "RES " + req.Method
	if res.status_code < 400 {
		l.Info().Msg(msg)
	} else {
		l.Warn().Msg(msg)
	}
}
