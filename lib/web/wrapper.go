package web

import (
	"bufio"
	"net"
	"net/http"

	"github.com/go-chi/chi/middleware"
)

type rw struct {
	middleware.WrapResponseWriter
	orig http.ResponseWriter
}

func (w *rw) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.orig.(http.Hijacker).Hijack()
}

type h2rw struct {
	middleware.WrapResponseWriter
	orig http.ResponseWriter
}

func (w *h2rw) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.orig.(http.Hijacker).Hijack()
}
func (w *h2rw) Push(target string, opts *http.PushOptions) error {
	return w.orig.(http.Pusher).Push(target, opts)
}

func wraprw(w http.ResponseWriter, r *http.Request) middleware.WrapResponseWriter {
	if _, ok := w.(http.Pusher); ok {
		return &h2rw{middleware.NewWrapResponseWriter(w, r.ProtoMajor), w}
	} else {
		return &rw{middleware.NewWrapResponseWriter(w, r.ProtoMajor), w}
	}
}
