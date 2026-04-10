package gateway

import (
	"net/http"
	"net/http/pprof"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

// registerPprofEndpoints mounts standard net/http/pprof endpoints into the provided mux.
// It uses runtime.ServeMux from grpc-gateway.
func registerPprofEndpoints(mux *runtime.ServeMux) error {
	// Root index
	if err := mux.HandlePath("GET", "/debug/pprof/", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		pprof.Index(w, r)
	}); err != nil {
		return err
	}

	// Specific profiles
	profiles := []string{"allocs", "block", "cmdline", "goroutine", "heap", "mutex", "profile", "threadcreate", "trace"}
	for _, p := range profiles {
		profile := p // capture loop variable
		if err := mux.HandlePath("GET", "/debug/pprof/"+profile, func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
			if profile == "cmdline" {
				pprof.Cmdline(w, r)
			} else if profile == "profile" {
				pprof.Profile(w, r)
			} else if profile == "trace" {
				pprof.Trace(w, r)
			} else {
				pprof.Handler(profile).ServeHTTP(w, r)
			}
		}); err != nil {
			return err
		}
	}

	return nil
}
