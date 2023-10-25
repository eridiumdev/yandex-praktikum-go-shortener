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

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/controller/http/middleware"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/crypto"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/usecase"
)

const dummyUserID = "user1"

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

			repo := repository.NewInMemShortlinkRepo(nil)
			uc := usecase.NewShortener(config.Shortener{
				BaseURL:       "http://127.0.0.1",
				DefaultLength: 5,
			}, repo)
			NewShortenerController(srv, uc)

			reqBody := bytes.NewBufferString(tt.body)
			r := httptest.NewRequest(http.MethodPost, "/", reqBody)
			addAuthCookie(r, dummyUserID)

			resp, err := srv.Test(r)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			err = resp.Body.Close()
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

			repo := repository.NewInMemShortlinkRepo(nil)
			uc := usecase.NewShortener(config.Shortener{
				BaseURL:       "http://127.0.0.1",
				DefaultLength: 5,
			}, repo)
			NewShortenerController(srv, uc)

			reqBody := bytes.NewBufferString(tt.req)
			r := httptest.NewRequest(http.MethodPost, "/api/shorten", reqBody)
			r.Header.Set("Content-Type", "application/json")
			addAuthCookie(r, dummyUserID)

			resp, err := srv.Test(r)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			err = resp.Body.Close()
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
			ID:     "link1",
			UserID: dummyUserID,
			Short:  "http://127.0.0.1/link1",
			Long:   "https://example.org",
		},
		{
			ID:     "link2",
			UserID: dummyUserID,
			Short:  "http://127.0.0.1/link2",
			Long:   "https://google.com",
		},
	} {
		err := repo.SaveShortlink(ctx, &link)
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
		//{
		//	name: "empty id",
		//	id:   "",
		//	want: want{
		//		code: 405,
		//	},
		//},
		//{
		//	name: "not found",
		//	id:   "link3",
		//	want: want{
		//		code: 404,
		//	},
		//},
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
			require.NoError(t, err)

			uc := usecase.NewShortener(config.Shortener{
				BaseURL:       "http://127.0.0.1",
				DefaultLength: 5,
			}, repo)
			NewShortenerController(srv, uc)

			r := httptest.NewRequest(http.MethodGet, "/"+tt.id, nil)
			addAuthCookie(r, dummyUserID)

			resp, err := srv.Test(r)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.want.code, resp.StatusCode)

			if tt.want.code == 307 {
				require.NotEmpty(t, resp.Header.Get("Location"))
				assert.Equal(t, tt.want.redirect, resp.Header.Get("Location"))
			}
		})
	}
}

func prepareRouter() (*fiber.App, error) {
	srv := fiber.New()

	mockCipher := crypto.NewMock()
	authMiddleware, err := middleware.CookieAuth(middleware.CookieAuthConfig{
		Cipher: mockCipher,
	})
	if err != nil {
		return nil, err
	}

	srv.Use(authMiddleware)
	return srv, nil
}

func addAuthCookie(r *http.Request, userID string) {
	r.AddCookie(&http.Cookie{
		Name:    middleware.CookieAuthName,
		Value:   userID,
		Expires: time.Now().Add(middleware.CookieAuthAge),
	})
}
