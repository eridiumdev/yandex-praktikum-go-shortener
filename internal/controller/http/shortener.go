package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"

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
	router.Get("/:id", c.getShortlink)

	return c
}

func (ctrl *ShortenerController) createShortlink(c *fiber.Ctx) error {
	ctx := c.Context()

	body := c.Body()

	link, err := ctrl.shortener.CreateShortlink(ctx, 0, string(body))
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

	var req shortenLinkRequest

	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing JSON request: %s", err)
		c.Status(http.StatusBadRequest)
		return c.SendString(err.Error())
	}

	link, err := ctrl.shortener.CreateShortlink(ctx, 0, req.URL)
	if err != nil {
		log.Printf("Error creating shortlink: %s", err)
		c.Status(ctrl.errorStatus(err))
		return c.SendString(err.Error())
	}

	c.Status(http.StatusCreated)
	c.Set("Content-Type", "application/json")

	result := shortenLinkResponse{Result: link.Short}
	resp, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error preparing JSON response: %s", err)
		c.Status(http.StatusInternalServerError)
		return c.SendString(err.Error())
	}

	return c.Send(resp)
	// v--- Does not pass Yandex test
	//return c.JSON(shortenLinkResponse{Result: link.Short})
}

func (ctrl *ShortenerController) getShortlink(c *fiber.Ctx) error {
	ctx := c.Context()

	id := c.Params("id")
	if id == "" {
		c.Status(http.StatusBadRequest)
		return nil
	}

	link, err := ctrl.shortener.GetShortlink(ctx, id)
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
