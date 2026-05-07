// Package logger expone un slog.Logger configurado para escribir en consola
// (formato humano) y, opcionalmente, en un archivo JSON estructurado.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// New construye un logger que escribe en stdout en formato texto y, si
// filePath no es vacío, también en filePath en formato JSON. El nivel se
// aplica a ambos.
func New(level slog.Level, filePath string) (*slog.Logger, io.Closer, error) {
	consoleH := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})

	if filePath == "" {
		return slog.New(consoleH), noopCloser{}, nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return nil, nil, err
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	fileH := slog.NewJSONHandler(f, &slog.HandlerOptions{Level: level})
	return slog.New(&fanoutHandler{handlers: []slog.Handler{consoleH, fileH}}), f, nil
}

// fanoutHandler reenvía cada record a múltiples handlers.
type fanoutHandler struct {
	handlers []slog.Handler
}

func (h *fanoutHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (h *fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, hh := range h.handlers {
		if err := hh.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (h *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		out[i] = hh.WithAttrs(attrs)
	}
	return &fanoutHandler{handlers: out}
}

func (h *fanoutHandler) WithGroup(name string) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		out[i] = hh.WithGroup(name)
	}
	return &fanoutHandler{handlers: out}
}

type noopCloser struct{}

func (noopCloser) Close() error { return nil }
