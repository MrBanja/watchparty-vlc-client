package logging

import (
	"io"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"go.uber.org/zap"
)

func Middleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		next = handlers.CustomLoggingHandler(os.Stdout, next, formatter(logger))

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info(
				">>>",
				zap.String("Method", r.Method),
				zap.String("URL", r.RequestURI),
			)
			next.ServeHTTP(w, r)
		})
	}
}

func formatter(logger *zap.Logger) handlers.LogFormatter {
	return func(writer io.Writer, params handlers.LogFormatterParams) {
		logger.Info(
			"<<<",
			zap.String("URL", params.URL.String()),
			zap.Int("Status", params.StatusCode),
		)
	}
}
