package http

import (
	"net/http"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/httpserver"
)

type Handler struct {
	Router *httpserver.Router
}

func NewHandler() *Handler {
	return &Handler{
		Router: httpserver.NewRouter(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Router.ServeHTTP(w, r)
}
