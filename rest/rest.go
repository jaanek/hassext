package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/zerodha/logf"
)

type Rest struct {
	Lo        logf.Logger
	Host      string
	Port      int
	JwtSecret string
	Server    *http.Server
}

func (r *Rest) Start(ctx context.Context) {
	// start the http server
	r.Server = &http.Server{
		Addr:    fmt.Sprintf("%v:%v", r.Host, r.Port),
		Handler: NewRouter(r),
		// ReadTimeout:  15 * time.Second,
		// WriteTimeout: 10 * time.Second,
		// IdleTimeout:  5 * time.Second,
	}
	r.Lo.Info("HTTP server starting", "addr", r.Server.Addr)
	err := r.Server.ListenAndServe()
	if err != nil {
		r.Lo.Error("http serve", "error", err)
	}
	r.Lo.Info("Http server finished!")
}
