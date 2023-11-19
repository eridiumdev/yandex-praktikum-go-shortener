package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/controller/http/middleware"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/crypto"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/usecase"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
)

const dummyUserID = "user1"

var log = logger.NewMockLogger()

func TestCreateShortlink(t *testing.T) {
	type want struct {
		code int
	}
	tests := []struct {
		name string
		body string
		want want
	}{
		{
			name: "empty url",
			body: "",
			want: want{
				code: 400,
			},
		},
		{
			name: "bad url",
			body: "askdjks#$JK@#$",
			want: want{
				code: 400,
			},
		},
		{
			name: "ok",
			body: "https://example.org",
			want: want{
				code: 201,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, err := prepareRouter()
			require.NoError(t, err)

			prepareController(srv, nil)

			reqBody := bytes.NewBufferString(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/", reqBody)
			addAuthCookie(req, dummyUserID)

			body, resp, err := sendRequest(srv, req)
			require.NoError(t, err)

			assert.Equal(t, tt.want.code, resp.StatusCode)

			if tt.want.code == 201 {
				assert.NotEmpty(t, body)

				_, err := url.Parse(string(body))
				require.NoErrorf(t, err, "url must parse")
			}
		})
	}
}

func TestShortenLink(t *testing.T) {
	type want struct {
		code int
		err  string
	}
	tests := []struct {
		name string
		req  string
		want want
	}{
		{
			name: "empty request",
			req:  ``,
			want: want{
				code: 400,
			},
		},
		{
			name: "empty url",
			req:  `{"url": ""}`,
			want: want{
				code: 400,
				err:  usecase.ErrIncompleteURL.Error(),
			},
		},
		{
			name: "bad url",
			req:  `{"url": ":asd&&!?"}`,
			want: want{
				code: 400,
				err:  usecase.ErrInvalidURL.Error(),
			},
		},
		{
			name: "incomplete url",
			req:  `{"url": "example.org"}`,
			want: want{
				code: 400,
				err:  usecase.ErrIncompleteURL.Error(),
			},
		},
		{
			name: "ok",
			req:  `{"url": "https://example.org"}`,
			want: want{
				code: 201,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, err := prepareRouter()
			require.NoError(t, err)

			prepareController(srv, nil)

			reqBody := bytes.NewBufferString(tt.req)
			req := httptest.NewRequest(http.MethodPost, "/api/shorten", reqBody)
			req.Header.Set("Content-Type", "application/json")
			addAuthCookie(req, dummyUserID)

			body, resp, err := sendRequest(srv, req)
			require.NoError(t, err)

			assert.Equal(t, tt.want.code, resp.StatusCode)

			if tt.want.err != "" {
				assert.Equal(t, tt.want.err, string(body))
			} else if tt.want.code == 201 {
				var respJSON shortenLinkResponse
				err := json.Unmarshal(body, &respJSON)
				require.NoError(t, err, "response should resemble a shortenLinkResponse")

				assert.NotEmpty(t, respJSON.Result, "response should contain a url")

				_, err = url.Parse(respJSON.Result)
				require.NoErrorf(t, err, "url must parse")
			}
		})
	}
}

func TestGetShortlink(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemShortlinkRepo(nil)

	for _, link := range []entity.Shortlink{
		{
			UID:     "link1",
			UserUID: dummyUserID,
			Short:   "http://127.0.0.1/link1",
			Long:    "https://example.org",
		},
		{
			UID:     "link2",
			UserUID: dummyUserID,
			Short:   "http://127.0.0.1/link2",
			Long:    "https://google.com",
		},
	} {
		_, err := repo.SaveShortlink(ctx, &link)
		require.NoError(t, err)
	}

	type want struct {
		code     int
		redirect string
	}
	tests := []struct {
		name string
		id   string
		want want
	}{
		{
			name: "empty id",
			id:   "",
			want: want{
				code: 404,
			},
		},
		{
			name: "not found",
			id:   "link3",
			want: want{
				code: 404,
			},
		},
		{
			name: "ok",
			id:   "link2",
			want: want{
				code:     307,
				redirect: "https://google.com",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, err := prepareRouter()
			require.NoError(t, err)

			prepareController(srv, repo)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.id, nil)
			addAuthCookie(req, dummyUserID)

			_, resp, err := sendRequest(srv, req)
			require.NoError(t, err)

			assert.Equal(t, tt.want.code, resp.StatusCode)

			if tt.want.code == 307 {
				require.NotEmpty(t, resp.Header.Get("Location"))
				assert.Equal(t, tt.want.redirect, resp.Header.Get("Location"))
			}
		})
	}
}

func prepareController(handler *gin.Engine, repo repository.ShortlinkRepo) {
	if repo == nil {
		repo = repository.NewInMemShortlinkRepo(nil)
	}
	uc := usecase.NewShortener(config.Shortener{
		BaseURL:       "http://127.0.0.1",
		DefaultLength: 5,
	}, repo, log)

	NewShortenerController(handler, uc, log)
}

func prepareRouter() (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	handler := gin.New()
	handler.ContextWithFallback = true

	mockCipher := crypto.NewMock()
	authMiddleware := middleware.CookieAuth(middleware.CookieAuthConfig{
		Cipher: mockCipher,
	}, log)

	handler.Use(authMiddleware)
	return handler, nil
}

func sendRequest(srv http.Handler, req *http.Request) ([]byte, *http.Response, error) {
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	body, err := io.ReadAll(w.Result().Body)
	if err != nil {
		return nil, nil, err
	}

	err = w.Result().Body.Close()

	return body, w.Result(), err
}

func addAuthCookie(r *http.Request, userUID string) {
	r.AddCookie(&http.Cookie{
		Name:    middleware.CookieAuthName,
		Value:   userUID,
		Expires: time.Now().Add(middleware.CookieAuthAge),
	})
}
