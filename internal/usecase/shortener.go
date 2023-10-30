package usecase

import (
	"context"
	"log"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
)

type ShortenerUC struct {
	baseURL       string
	defaultLength int
	alphabet      []rune

	repo repository.ShortlinkRepo
	rng  *rand.Rand
}

func NewShortener(cfg config.Shortener, repo repository.ShortlinkRepo) *ShortenerUC {
	var alphabet []rune

	for c := '0'; c < '9'; c++ {
		alphabet = append(alphabet, c)
	}
	for c := 'A'; c < 'Z'; c++ {
		alphabet = append(alphabet, c)
	}
	for c := 'a'; c < 'z'; c++ {
		alphabet = append(alphabet, c)
	}

	return &ShortenerUC{
		baseURL:       strings.TrimRight(cfg.BaseURL, "/") + "/",
		defaultLength: cfg.DefaultLength,
		alphabet:      alphabet,
		repo:          repo,
		rng:           rand.New(rand.NewSource(time.Now().Unix())),
	}
}

func (uc *ShortenerUC) Ping(ctx context.Context) error {
	err := uc.repo.Ping(ctx)
	if err != nil {
		log.Printf("error pinging repo: %s", err)
		return ErrDbUnavailable
	}
	return nil
}

func (uc *ShortenerUC) CreateShortlink(ctx context.Context, userID string, length int, longURL string) (*entity.Shortlink, error) {
	uri, err := url.Parse(longURL)
	if err != nil {
		log.Printf("Error parsing URL: %s", err)
		return nil, ErrInvalidURL
	}
	if uri.Scheme == "" || uri.Host == "" {
		log.Printf("Provided URL is incomplete (%s)", longURL)
		return nil, ErrIncompleteURL
	}

	if length <= 0 {
		length = uc.defaultLength
	}

	var linkID string

	for tries := 0; ; tries++ {
		linkID = uc.generateLinkID(length)
		exists, err := uc.repo.FindShortlink(ctx, userID, linkID)
		if err != nil {
			return nil, err
		}
		if exists == nil {
			break
		}
		if tries > 2 {
			return nil, ErrIDConflict
		}
	}

	link := &entity.Shortlink{
		ID:     linkID,
		UserID: userID,
		Long:   longURL,
		Short:  uc.baseURL + linkID,
	}

	log.Printf("URL shortened: %s -> %s", link.Long, link.Short)

	return link, uc.repo.SaveShortlink(ctx, link)
}

func (uc *ShortenerUC) generateLinkID(length int) string {
	var id string

	for i := 0; i < length; i++ {
		random := uc.rng.Intn(len(uc.alphabet))
		id += string(uc.alphabet[random])
	}

	return id
}

func (uc *ShortenerUC) GetShortlink(ctx context.Context, linkID string) (*entity.Shortlink, error) {
	return uc.repo.FindShortlink(ctx, "", linkID)
}

func (uc *ShortenerUC) GetUserShortlink(ctx context.Context, userID, linkID string) (*entity.Shortlink, error) {
	return uc.repo.FindShortlink(ctx, userID, linkID)
}

func (uc *ShortenerUC) ListUserShortlinks(ctx context.Context, userID string) ([]*entity.Shortlink, error) {
	return uc.repo.GetShortlinks(ctx, userID)
}
