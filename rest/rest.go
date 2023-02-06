package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jaanek/hassext/emodul"
	"github.com/zerodha/logf"
)

type Rest struct {
	server    *http.Server
	em        emodul.EModul
	lo        logf.Logger
	host      string
	port      int
	jwtSecret string
}

func NewRest(lo logf.Logger, em emodul.EModul, host string, port int, jwtSecret string) *Rest {
	return &Rest{
		em:        em,
		lo:        lo,
		host:      host,
		port:      port,
		jwtSecret: jwtSecret,
	}
}

func (r *Rest) Start(ctx context.Context) {
	// start the http server
	r.server = &http.Server{
		Addr:    fmt.Sprintf("%v:%v", r.host, r.port),
		Handler: NewRouter(r),
		// ReadTimeout:  15 * time.Second,
		// WriteTimeout: 10 * time.Second,
		// IdleTimeout:  5 * time.Second,
	}
	r.lo.Info("HTTP server starting", "addr", r.server.Addr)
	err := r.server.ListenAndServe()
	if err != nil {
		r.lo.Error("http serve", "error", err)
	}
	r.lo.Info("Http server finished!")
}

func (r *Rest) Shutdown(ctx context.Context) {
	r.server.Shutdown(ctx)
}
