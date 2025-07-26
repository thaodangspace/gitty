package middleware

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func Logger(next http.Handler) http.Handler {
	return middleware.RequestLogger(&middleware.DefaultLogFormatter{
		Logger:  log.Default(),
		NoColor: false,
	})(next)
}

func Recoverer(next http.Handler) http.Handler {
	return middleware.Recoverer(next)
}