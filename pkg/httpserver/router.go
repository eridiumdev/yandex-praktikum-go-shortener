package httpserver

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type (
	Router struct {
		routes []*route
	}
	route struct {
		pattern   string
		length    int
		regex     *regexp.Regexp
		f         http.HandlerFunc
		wildcards []wildcard
	}
	wildcard struct {
		name string
	}
	match struct {
		name  string
		value string
	}
)

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) HandleFunc(pattern string, f http.HandlerFunc) {
	route := &route{
		pattern: pattern,
		f:       f,
	}
	route.prepare()

	r.routes = append(r.routes, route)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	uri := req.URL.Path

	maxLength := 0
	var ids []match
	var finalRoute *route

	for _, r := range r.routes {
		if matched, matches := r.matches(uri); matched && r.length > maxLength {
			finalRoute = r
			maxLength = r.length
			ids = matches
		}
	}

	if finalRoute == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	for _, id := range ids {
		req.Header.Set("X-Wildcard-"+id.name, id.value)
	}

	finalRoute.f.ServeHTTP(w, req)
}

func (r *route) prepare() {
	p := r.pattern
	r.length = strings.Count(strings.ReplaceAll(fmt.Sprintf("%s/", p), "//", "/"), "/")

	for strings.Contains(p, "{") && strings.Contains(p, "}") {
		start := strings.Index(p, "{")
		end := strings.Index(p, "}")
		r.wildcards = append(r.wildcards, wildcard{name: p[start+1 : end]})
		p = fmt.Sprintf("%s([a-zA-Z0-9]+)%s", p[:start], p[end+1:])
	}

	p = fmt.Sprintf("%s", p)
	r.regex = regexp.MustCompile(p)
}

func (r *route) matches(uri string) (bool, []match) {
	var matched bool
	var matches []match

	for i, m := range r.regex.FindStringSubmatch(uri) {
		matched = true
		if i == 0 {
			continue
		}

		matches = append(matches, match{
			name:  r.wildcards[i-1].name,
			value: m,
		})
	}

	return matched, matches
}
