package http

import (
	"log"
	"net/http"
	"net/url"

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
	router.Get("/:id", c.getShortlink)

	return c
}

func (ctrl *ShortenerController) createShortlink(c *fiber.Ctx) error {
	ctx := c.Context()

	body := c.Body()

	uri, err := url.Parse(string(body))
	if err != nil {
		log.Printf("Error parsing URL: %s", err)
		c.Status(http.StatusBadRequest)
		return nil
	}
	if uri.Scheme == "" || uri.Host == "" {
		log.Printf("Provided URL is incomplete (%s)", string(body))
		c.Status(http.StatusBadRequest)
		return nil
	}

	link, err := ctrl.shortener.CreateShortlink(ctx, 0, uri.String())
	if err != nil {
		log.Printf("Error creating shortlink: %s", err)
		c.Status(http.StatusInternalServerError)
		return c.SendString(err.Error())
	}

	c.Status(http.StatusCreated)
	return c.SendString(link.Short)
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
		c.Status(http.StatusInternalServerError)
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
