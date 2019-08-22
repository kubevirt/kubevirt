package profile

import (
	"net/http"
	"net/http/pprof"
)

type profileConfig struct {
	pprof   bool
	cmdline bool
	profile bool
	symbol  bool
	trace   bool
}

// Option applies a configuration option to the given config.
type Option func(p *profileConfig)

func (p *profileConfig) apply(options []Option) {
	if len(options) == 0 {
		// If no options are given, default to all
		p.pprof = true
		p.cmdline = true
		p.profile = true
		p.symbol = true
		p.trace = true

		return
	}

	for _, o := range options {
		o(p)
	}
}

func defaultProfileConfig() *profileConfig {
	// Initialize config
	return &profileConfig{}
}

// RegisterHandlers registers profile Handlers with the given ServeMux.
//
// The Handlers registered are determined by the given options.
// If no options are given, all available handlers are registered by default.
func RegisterHandlers(mux *http.ServeMux, options ...Option) {
	config := defaultProfileConfig()
	config.apply(options)

	if config.pprof {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
	}
	if config.cmdline {
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	}
	if config.profile {
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	}
	if config.symbol {
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}
	if config.trace {
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
}
