package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/usecase"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
)

type ShortenerController struct {
	shortener usecase.Shortener

	log *logger.Logger
}

func NewShortenerController(router *gin.Engine, shortener usecase.Shortener, log *logger.Logger) *ShortenerController {
	c := &ShortenerController{
		shortener: shortener,
		log:       log,
	}

	router.GET("/ping", c.ping)

	router.POST("/", c.createShortlink)
	router.GET("/:id", c.getShortlink)

	router.POST("/api/shorten", c.shortenLink)
	router.POST("/api/shorten/batch", c.shortenLinksBatch)
	router.GET("/api/user/urls", c.listShortlinks)

	return c
}

func (ct *ShortenerController) ping(c *gin.Context) {
	ctx := c

	select {
	case <-ctx.Done():
		ct.log.Info(c).Msg("donezo")
		return
	case <-time.After(time.Second * 3):
		ct.log.Info(c).Msg("slept")
	}

	//err := ct.shortener.Ping(c)
	//if err != nil {
	//	c.Status(ct.errorStatus(err))
	//	c.String(err.Error())
	//return
	//}

	c.String(http.StatusOK, "MKAY")
}

func (ct *ShortenerController) createShortlink(c *gin.Context) {
	ctx := c

	userUID, err := ct.userUID(ctx)
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		ct.log.Error(ctx, err).Msg("extract request body")
		c.Status(http.StatusBadRequest)
		return
	}

	link, err := ct.shortener.CreateShortlink(ctx, userUID, 0, string(body))

	switch {
	case err == nil:
		c.String(http.StatusCreated, link.Short)
	case errors.Is(err, repository.ErrURLConflict):
		c.Status(http.StatusConflict)
	default:
		ct.log.Error(ctx, err).Msg("create shortlink")
		c.String(ct.errorStatus(err), err.Error())
	}
}

type (
	shortenLinkRequest struct {
		URL string `json:"url"`
	}
	shortenLinkResponse struct {
		Result string `json:"result"`
	}
)

func (ct *ShortenerController) shortenLink(c *gin.Context) {
	ctx := c

	userUID, err := ct.userUID(ctx)
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	var req shortenLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ct.log.Error(ctx, err).Msg("parse JSON request")
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	link, err := ct.shortener.CreateShortlink(ctx, userUID, 0, req.URL)
	var status int

	switch {
	case err == nil:
		status = http.StatusCreated
	case errors.Is(err, repository.ErrURLConflict):
		status = http.StatusConflict
	default:
		ct.log.Error(ctx, err).Msg("create shortlink")
		c.String(ct.errorStatus(err), err.Error())
		return
	}

	result := shortenLinkResponse{Result: link.Short}
	c.JSON(status, result)
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

func (ct *ShortenerController) shortenLinksBatch(c *gin.Context) {
	ctx := c

	userUID, err := ct.userUID(ctx)
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	var req shortenLinksBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ct.log.Error(ctx, err).Msg("parse JSON request")
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	var data []usecase.CreateShortlinksInLink
	for _, link := range req {
		data = append(data, usecase.CreateShortlinksInLink{
			URL:           link.OriginalURL,
			CorrelationID: link.CorrelationID,
		})
	}

	links, err := ct.shortener.CreateShortlinks(ctx, usecase.CreateShortlinksIn{
		UserUID: userUID,
		Length:  0,
		Links:   data,
	})
	if err != nil {
		ct.log.Error(ctx, err).Msg("create shortlink")
		c.String(ct.errorStatus(err), err.Error())
		return
	}

	var result shortenLinksBatchResponse
	for _, link := range links {
		result = append(result, shortenLinksBatchResponseLink{
			CorrelationID: link.CorrelationID,
			ShortURL:      link.Short,
		})
	}

	c.JSON(http.StatusCreated, result)
}

type (
	getShortlinkRequest struct {
		LinkUID string `uri:"id" binding:"required" validate:"len=5"`
	}
)

func (ct *ShortenerController) getShortlink(c *gin.Context) {
	ctx := c

	var req getShortlinkRequest
	if err := c.ShouldBindUri(&req); err != nil {
		ct.log.Error(ctx, err).Msg("parse URI request")
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	link, err := ct.shortener.GetShortlink(ctx, req.LinkUID)
	if err != nil {
		ct.log.Error(ctx, err).Msg("get shortlink")
		c.String(ct.errorStatus(err), err.Error())
		return
	}
	if link == nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Location", link.Long)
	c.Status(http.StatusTemporaryRedirect)
}

type (
	listShortlinksResponse     []listShortlinksResponseLink
	listShortlinksResponseLink struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
)

func (ct *ShortenerController) listShortlinks(c *gin.Context) {
	ctx := c

	userUID, err := ct.userUID(ctx)
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	links, err := ct.shortener.ListUserShortlinks(ctx, userUID)
	if err != nil {
		ct.log.Error(ctx, err).Msg("list user shortlinks")
		c.String(ct.errorStatus(err), err.Error())
		return
	}
	if len(links) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	var result listShortlinksResponse

	for _, link := range links {
		result = append(result, listShortlinksResponseLink{
			ShortURL:    link.Short,
			OriginalURL: link.Long,
		})
	}
	c.JSON(http.StatusOK, result)
}

func (ct *ShortenerController) userUID(c context.Context) (string, error) {
	authToken := c.Value(string(entity.AuthTokenCtxKey))

	token, ok := authToken.(*entity.AuthToken)
	if !ok || token.UserUID == "" {
		return "", ErrUnauthenticatedUser
	}

	return token.UserUID, nil
}

func (ct *ShortenerController) errorStatus(err error) int {
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
