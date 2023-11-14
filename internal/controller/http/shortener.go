package http

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/usecase"
)

type ShortenerController struct {
	shortener usecase.Shortener
}

func NewShortenerController(router *fiber.App, shortener usecase.Shortener) *ShortenerController {
	c := &ShortenerController{
		shortener: shortener,
	}

	router.Get("/ping", c.ping)

	router.Post("/", c.createShortlink)
	router.Get("/:id<len(5)>", c.getShortlink)

	router.Post("/api/shorten", c.shortenLink)
	router.Post("/api/shorten/batch", c.shortenLinksBatch)
	router.Get("/api/user/urls", c.listShortlinks)

	return c
}

func (ctrl *ShortenerController) ping(c *fiber.Ctx) error {
	ctx := c.Context()

	err := ctrl.shortener.Ping(ctx)
	if err != nil {
		c.Status(ctrl.errorStatus(err))
		return c.SendString(err.Error())
	}

	c.Status(http.StatusOK)
	return nil
}

func (ctrl *ShortenerController) createShortlink(c *fiber.Ctx) error {
	ctx := c.Context()

	userUID, err := ctrl.userUID(c.UserContext())
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return nil
	}

	body := c.Body()

	link, err := ctrl.shortener.CreateShortlink(ctx, userUID, 0, string(body))

	switch {
	case err == nil:
		c.Status(http.StatusCreated)
	case errors.Is(err, repository.ErrURLConflict):
		c.Status(http.StatusConflict)
	case err != nil:
		log.Printf("Error creating shortlink: %s", err)
		c.Status(ctrl.errorStatus(err))
		return c.SendString(err.Error())
	}

	return c.SendString(link.Short)
}

type (
	shortenLinkRequest struct {
		URL string `json:"url"`
	}
	shortenLinkResponse struct {
		Result string `json:"result"`
	}
)

func (ctrl *ShortenerController) shortenLink(c *fiber.Ctx) error {
	ctx := c.Context()

	userUID, err := ctrl.userUID(c.UserContext())
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return nil
	}

	var req shortenLinkRequest

	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing JSON request: %s", err)
		c.Status(http.StatusBadRequest)
		return c.SendString(err.Error())
	}

	link, err := ctrl.shortener.CreateShortlink(ctx, userUID, 0, req.URL)

	switch {
	case err == nil:
		c.Status(http.StatusCreated)
	case errors.Is(err, repository.ErrURLConflict):
		c.Status(http.StatusConflict)
	case err != nil:
		log.Printf("Error creating shortlink: %s", err)
		c.Status(ctrl.errorStatus(err))
		return c.SendString(err.Error())
	}

	result := shortenLinkResponse{Result: link.Short}
	resp, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error preparing JSON response: %s", err)
		c.Status(http.StatusInternalServerError)
		return c.SendString(err.Error())
	}

	c.Set("Content-Type", "application/json")

	return c.Send(resp)
	// v--- Does not pass Yandex test
	//return c.JSON(shortenLinkResponse{Result: link.Short})
}

type (
	shortenLinksBatchRequest     []shortenLinksBatchRequestLink
	shortenLinksBatchRequestLink struct {
		CorrelationID string `json:"correlation_id"`
		OriginalURL   string `json:"original_url"`
	}
	shortenLinksBatchResponse     []shortenLinksBatchResponseLink
	shortenLinksBatchResponseLink struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}
)

func (ctrl *ShortenerController) shortenLinksBatch(c *fiber.Ctx) error {
	ctx := c.Context()

	userUID, err := ctrl.userUID(c.UserContext())
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return nil
	}

	var req shortenLinksBatchRequest

	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing JSON request: %s", err)
		c.Status(http.StatusBadRequest)
		return c.SendString(err.Error())
	}

	var data []usecase.CreateShortlinksInLink
	for _, link := range req {
		data = append(data, usecase.CreateShortlinksInLink{
			URL:           link.OriginalURL,
			CorrelationID: link.CorrelationID,
		})
	}

	links, err := ctrl.shortener.CreateShortlinks(ctx, usecase.CreateShortlinksIn{
		UserUID: userUID,
		Length:  0,
		Links:   data,
	})
	if err != nil {
		log.Printf("Error creating shortlink: %s", err)
		c.Status(ctrl.errorStatus(err))
		return c.SendString(err.Error())
	}

	var result shortenLinksBatchResponse
	for _, link := range links {
		result = append(result, shortenLinksBatchResponseLink{
			CorrelationID: link.CorrelationID,
			ShortURL:      link.Short,
		})
	}

	resp, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error preparing JSON response: %s", err)
		c.Status(http.StatusInternalServerError)
		return c.SendString(err.Error())
	}

	c.Status(http.StatusCreated)
	c.Set("Content-Type", "application/json")

	return c.Send(resp)
}

func (ctrl *ShortenerController) getShortlink(c *fiber.Ctx) error {
	ctx := c.Context()

	linkUID := c.Params("id")
	if linkUID == "" {
		c.Status(http.StatusBadRequest)
		return nil
	}

	link, err := ctrl.shortener.GetShortlink(ctx, linkUID)
	if err != nil {
		log.Printf("Error getting shortlink: %s", err)
		c.Status(ctrl.errorStatus(err))
		return c.SendString(err.Error())
	}
	if link == nil {
		c.Status(http.StatusNotFound)
		return nil
	}

	c.Set("Location", link.Long)
	c.Status(http.StatusTemporaryRedirect)
	return nil
}

type (
	listShortlinksResponse     []listShortlinksResponseLink
	listShortlinksResponseLink struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
)

func (ctrl *ShortenerController) listShortlinks(c *fiber.Ctx) error {
	ctx := c.Context()

	userUID, err := ctrl.userUID(c.UserContext())
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return nil
	}

	links, err := ctrl.shortener.ListUserShortlinks(ctx, userUID)
	if err != nil {
		log.Printf("Error listing shortlinks: %s", err)
		c.Status(ctrl.errorStatus(err))
		return c.SendString(err.Error())
	}
	if len(links) == 0 {
		c.Status(http.StatusNoContent)
		return nil
	}

	var result listShortlinksResponse

	for _, link := range links {
		result = append(result, listShortlinksResponseLink{
			ShortURL:    link.Short,
			OriginalURL: link.Long,
		})
	}

	resp, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error preparing JSON response: %s", err)
		c.Status(http.StatusInternalServerError)
		return c.SendString(err.Error())
	}

	c.Set("Content-Type", "application/json")

	return c.Send(resp)
}

func (ctrl *ShortenerController) userUID(ctx context.Context) (string, error) {
	authToken := ctx.Value(entity.AuthTokenCtxKey)

	token, ok := authToken.(*entity.AuthToken)
	if !ok || token.UserUID == "" {
		return "", ErrUnauthenticatedUser
	}

	return token.UserUID, nil
}

func (ctrl *ShortenerController) errorStatus(err error) int {
	switch {
	case errors.Is(err, usecase.ErrInvalidURL):
		fallthrough
	case errors.Is(err, usecase.ErrIncompleteURL):
		return http.StatusBadRequest
	case errors.Is(err, usecase.ErrUIDConflict):
		return http.StatusConflict
	case errors.Is(err, usecase.ErrDBUnavailable):
		fallthrough
	default:
		return http.StatusInternalServerError
	}
}
