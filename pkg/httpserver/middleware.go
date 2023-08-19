package httpserver

import "net/http"

type Middleware func(http.HandlerFunc) http.HandlerFunc

func Conveyor(h http.HandlerFunc, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h)
	}
	return h
}

func GET(h http.HandlerFunc) http.HandlerFunc {
	return httpMethod(http.MethodGet, h)
}

func POST(h http.HandlerFunc) http.HandlerFunc {
	return httpMethod(http.MethodPost, h)
}

func httpMethod(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h.ServeHTTP(w, r)
	}
}
