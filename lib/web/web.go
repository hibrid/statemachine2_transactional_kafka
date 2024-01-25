package web

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/hibrid/statemachine2_transactional_kafka/lib/app"
	"go.uber.org/zap"
)

type errobj struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

// NoCache adds the no cache headers to a web response
func NoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "max-age=0, no-cache, no-store")
	w.Header().Set("Pragma", "no-cache")
}

// NoExtOrNotFound is a response handler for returning a 404
func NoExtOrNotFound(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}
}

// NoSessionLogger because at some point we might want to add, sessions to authenticate requests
func NoSessionLogger(logAfter time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrw := wraprw(w, r)
			next.ServeHTTP(wrw, r)
			end := time.Now()
			diff := end.Sub(start)
			if diff > logAfter {
				app.Log.Info("Web Request",
					zap.String("path", r.RequestURI),
					zap.Int("status", wrw.Status()),
					zap.Int("sz", wrw.BytesWritten()),
					zap.Duration("time", diff),
				)
			}
		})
	}
}

// BadInput is a type for implementing an error interface
type BadInput string

func (bi BadInput) Error() string {
	return string(bi)
}

type errorResponse struct {
	Error errobj `json:"error"`
}

// Err is a response handler for converting gRPC errors to web errors
func Err(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	clog := app.Logger().WithOptions(zap.AddCallerSkip(1))
	var errMsg errobj

	//parse errors and assing an http code or text. maybe send to a parser later?
	if _, ok := err.(BadInput); ok {
		errMsg.Code = 400
	} else if err == sql.ErrNoRows { //for when we start doing db queries
		errMsg.Code = 404
		errMsg.Message = "not found"
	} else {
		errMsg.Code = 500
		errMsg.Message = err.Error()
	}

	w.WriteHeader(200)
	render.JSON(w, r, errorResponse{errMsg})
	//maybe do something interesting with different error types?
	switch errMsg.Code {
	case 500:
		clog.Error("Returning error to client", zap.Int("code", int(errMsg.Code)), zap.String("message", errMsg.Message))
	default:
		clog.Debug("Returning error to client", zap.Int("code", int(errMsg.Code)), zap.String("message", errMsg.Message))
	}
	return true
}
