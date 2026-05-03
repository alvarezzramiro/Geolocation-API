// api/router.go
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter construye el router HTTP con todos los endpoints y middlewares.
func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Middlewares — se ejecutan en orden para cada request
	r.Use(middleware.Logger)    // loguea método, path y tiempo de respuesta
	r.Use(middleware.Recoverer) // si un handler hace panic, responde 500 en vez de crashear

	r.Get("/nodes", h.GetNodes)
	r.Get("/route", h.GetRoute)
	r.Get("/route/by-intersection", h.GetRouteByIntersection)

	return r
}
