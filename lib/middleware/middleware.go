package middleware

import (
	"net/http"

	"go.uber.org/zap"
)

type Logger struct {
	logger *zap.Logger
}

func NewMiddlewareLogger(log *zap.Logger) *Logger {
	return &Logger{logger: log}
}

func (l *Logger) Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.logger.Info("Request", zap.String("URI", r.RequestURI), zap.String("Addr", r.RemoteAddr))
		handler.ServeHTTP(w, r)
	})
}

func SetContentTypeJSON(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handler.ServeHTTP(w, r)
	})
}
