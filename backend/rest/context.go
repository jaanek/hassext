package rest

import (
	"context"
	"net/http"
)

type key int

const (
	JwtUserKey key = iota
)

func SetReqValue(r *http.Request, key key, val interface{}) *http.Request {
	ctx := context.WithValue(r.Context(), key, val)
	return r.WithContext(ctx)
}

func GetReqValue(r *http.Request, key key) interface{} {
	return r.Context().Value(key)
}
