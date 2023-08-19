package http

import (
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/httpserver"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/usecase"
)

type shortenerController struct {
	shortener usecase.Shortener
}

func NewShortenerController(router *httpserver.Router, shortener usecase.Shortener) *shortenerController {
	c := &shortenerController{
		shortener: shortener,
	}

	router.HandleFunc("/", httpserver.POST(c.createShortlink))
	router.HandleFunc("/{id}", httpserver.GET(c.getShortlink))

	return c
}

func (c *shortenerController) createShortlink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	uri, err := url.Parse(string(body))
	if err != nil {
		log.Printf("Error parsing URL: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if uri.Scheme == "" || uri.Host == "" {
		log.Printf("Provided URL is incomplete (%s)", string(body))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	link, err := c.shortener.CreateShortlink(ctx, 0, uri.String())
	if err != nil {
		log.Printf("Error creating shortlink: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(link.Short))
}

func (c *shortenerController) getShortlink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.Header.Get("X-Wildcard-id")

	link, err := c.shortener.GetShortlink(ctx, id)
	if err != nil {
		log.Printf("Error getting shortlink: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	if link == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Location", link.Long)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
