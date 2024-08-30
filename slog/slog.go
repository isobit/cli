// +build go1.21

package slog

import (
	"os"
	"io"
	"log/slog"
)

type SlogOptions struct {
	LogLevel    slog.Level     `cli:"env=LOG_LEVEL"`
	LogJson     bool           `cli:"env=LOG_JSON"`
}

func (opts *SlogOptions) ConfigureWithHandlerOptions(w io.Writer, handlerOpts *slog.HandlerOptions) {
	if handlerOpts == nil {
		handlerOpts = &slog.HandlerOptions{}
	}
	handlerOpts.Level = opts.LogLevel

	var handler slog.Handler
	if opts.LogJson {
		handler = slog.NewJSONHandler(w, handlerOpts)
	} else {
		handler = slog.NewTextHandler(w, handlerOpts)
	}
	slog.SetDefault(slog.New(handler))
}

func (opts *SlogOptions) Configure() {
	opts.ConfigureWithHandlerOptions(os.Stderr, nil)
}
