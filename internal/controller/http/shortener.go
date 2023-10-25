package http

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/usecase"
)

type ShortenerController struct {
	shortener usecase.Shortener
}

func NewShortenerController(router *fiber.App, shortener usecase.Shortener) *ShortenerController {
	c := &ShortenerController{
		shortener: shortener,
	}

	router.Post("/", c.createShortlink)
	router.Post("/api/shorten", c.shortenLink)
	router.Get("/api/user/urls", c.listShortlinks)
	router.Get("/:id", c.getShortlink)

	return c
}

func (ctrl *ShortenerController) createShortlink(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, err := ctrl.userID(c.UserContext())
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return nil
	}

	body := c.Body()

	link, err := ctrl.shortener.CreateShortlink(ctx, userID, 0, string(body))
	if err != nil {
		log.Printf("Error creating shortlink: %s", err)
		c.Status(ctrl.errorStatus(err))
		return c.SendString(err.Error())
	}

	c.Status(http.StatusCreated)
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

	userID, err := ctrl.userID(c.UserContext())
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

	link, err := ctrl.shortener.CreateShortlink(ctx, userID, 0, req.URL)
	if err != nil {
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

	c.Status(http.StatusCreated)
	c.Set("Content-Type", "application/json")

	return c.Send(resp)
	// v--- Does not pass Yandex test
	//return c.JSON(shortenLinkResponse{Result: link.Short})
}

func (ctrl *ShortenerController) getShortlink(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, err := ctrl.userID(c.UserContext())
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return nil
	}

	linkID := c.Params("id")
	if linkID == "" {
		c.Status(http.StatusBadRequest)
		return nil
	}

	link, err := ctrl.shortener.GetShortlink(ctx, userID, linkID)
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

	userID, err := ctrl.userID(c.UserContext())
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return nil
	}

	links, err := ctrl.shortener.ListShortlinks(ctx, userID)
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

func (ctrl *ShortenerController) userID(ctx context.Context) (string, error) {
	authToken := ctx.Value(entity.AuthTokenCtxKey)

	token, ok := authToken.(*entity.AuthToken)
	if !ok || token.UserID == "" {
		return "", ErrUnauthenticatedUser
	}

	return token.UserID, nil
}

func (ctrl *ShortenerController) errorStatus(err error) int {
	switch {
	case errors.Is(err, usecase.ErrInvalidURL):
		fallthrough
	case errors.Is(err, usecase.ErrIncompleteURL):
		return http.StatusBadRequest
	case errors.Is(err, usecase.ErrIDConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
