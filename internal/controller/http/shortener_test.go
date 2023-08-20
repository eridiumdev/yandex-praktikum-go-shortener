package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/usecase"
)

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
			srv := fiber.New()

			repo := repository.NewInMemShortlinkRepo()
			uc := usecase.NewShortener(config.Shortener{
				BaseUrl:       "http://127.0.0.1",
				DefaultLength: 5,
			}, repo)
			NewShortenerController(srv, uc)

			reqBody := bytes.NewBufferString(tt.body)
			r := httptest.NewRequest(http.MethodPost, "/", reqBody)

			resp, err := srv.Test(r)
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

func TestGetShortlink(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemShortlinkRepo()

	for _, link := range []entity.Shortlink{
		{
			ID:    "link1",
			Short: "http://127.0.0.1/link1",
			Long:  "https://example.org",
		},
		{
			ID:    "link2",
			Short: "http://127.0.0.1/link2",
			Long:  "https://google.com",
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
		{
			name: "empty id",
			id:   "",
			want: want{
				code: 405,
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
			srv := fiber.New()

			uc := usecase.NewShortener(config.Shortener{
				BaseUrl:       "http://127.0.0.1",
				DefaultLength: 5,
			}, repo)
			NewShortenerController(srv, uc)

			r := httptest.NewRequest(http.MethodGet, "/"+tt.id, nil)

			resp, err := srv.Test(r)
			require.NoError(t, err)

			assert.Equal(t, tt.want.code, resp.StatusCode)

			if tt.want.code == 307 {
				require.NotEmpty(t, resp.Header.Get("Location"))
				assert.Equal(t, tt.want.redirect, resp.Header.Get("Location"))
			}
		})
	}
}
