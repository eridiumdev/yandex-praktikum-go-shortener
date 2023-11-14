package usecase

import (
	"context"
	"log"
	"math/rand"
	neturl "net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

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
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (uc *ShortenerUC) Ping(ctx context.Context) error {
	err := uc.repo.Ping(ctx)
	if err != nil {
		log.Printf("error pinging repo: %s", err)
		return ErrDBUnavailable
	}
	return nil
}

func (uc *ShortenerUC) CreateShortlink(ctx context.Context, userUID string, length int, longURL string) (*entity.Shortlink, error) {
	err := uc.validateURL(longURL)
	if err != nil {
		return nil, err
	}

	if length <= 0 {
		length = uc.defaultLength
	}

	var linkUID string

	for tries := 0; ; tries++ {
		linkUID = uc.generateLinkUID(length)
		exists, err := uc.repo.FindShortlink(ctx, userUID, linkUID)
		if err != nil {
			return nil, err
		}
		if exists == nil {
			break
		}
		if tries > 2 {
			return nil, ErrUIDConflict
		}
	}

	link := &entity.Shortlink{
		UID:     linkUID,
		UserUID: userUID,
		Long:    longURL,
		Short:   uc.baseURL + linkUID,
	}

	log.Printf("URL shortened: %s -> %s", link.Long, link.Short)

	link, err = uc.repo.SaveShortlink(ctx, link)
	if err != nil {
		if errors.Is(err, repository.ErrURLConflict) {
			return link, err
		}
		return nil, errors.Wrapf(err, "[UC] save shortlink")
	}

	return link, nil
}

func (uc *ShortenerUC) CreateShortlinks(ctx context.Context, data CreateShortlinksIn) ([]*entity.Shortlink, error) {
	if data.Length <= 0 {
		data.Length = uc.defaultLength
	}

	linkMap := make(map[string]*entity.Shortlink, len(data.Links))
	linkUIDs := make([]string, 0)

	for _, longLink := range data.Links {
		err := uc.validateURL(longLink.URL)
		if err != nil {
			return nil, errors.Wrapf(err, "URL = %s", longLink.URL)
		}

		link, err := uc.prepareShortlink(longLink.URL, data.Length, data.UserUID, longLink.CorrelationID)
		if err != nil {
			return nil, errors.Wrapf(err, "[UC] prepare shortlink")
		}
		linkMap[link.UID] = link
		linkUIDs = append(linkUIDs, link.UID)
	}

	for tries := 0; ; tries++ {
		if tries > 2 {
			return nil, ErrUIDConflict
		}

		duplicates, err := uc.repo.FindShortlinks(ctx, linkUIDs)
		if err != nil {
			return nil, errors.Wrapf(err, "[UC] find shortlinks")
		}
		if len(duplicates) == 0 {
			break
		}

		// Reset for next FindShortlinks (will only contain re-generated links)
		linkUIDs = linkUIDs[:0]

		for _, dup := range duplicates {
			link, err := uc.prepareShortlink(dup.Long, data.Length, data.UserUID, dup.CorrelationID)
			if err != nil {
				return nil, errors.Wrapf(err, "[UC] prepare shortlink (dup)")
			}
			linkUIDs = append(linkUIDs, link.UID)
			linkMap[dup.UID] = link
		}
	}

	var links []*entity.Shortlink

	for _, link := range linkMap {
		links = append(links, link)
		log.Printf("URL shortened: %s -> %s", link.Long, link.Short)
	}

	links, err := uc.repo.SaveShortlinks(ctx, links)
	if err != nil {
		return nil, errors.Wrapf(err, "[UC] save shortlinks")
	}

	return links, nil
}

func (uc *ShortenerUC) prepareShortlink(longURL string, length int, userUID string, correlationID string) (*entity.Shortlink, error) {
	linkUID := uc.generateLinkUID(length)

	return &entity.Shortlink{
		UID:           linkUID,
		UserUID:       userUID,
		Long:          longURL,
		Short:         uc.baseURL + linkUID,
		CorrelationID: correlationID,
	}, nil
}

func (uc *ShortenerUC) validateURL(url string) error {
	uri, err := neturl.Parse(url)
	if err != nil {
		log.Printf("Error parsing URL: %s", err)
		return ErrInvalidURL
	}
	if uri.Scheme == "" || uri.Host == "" {
		log.Printf("Provided URL is incomplete (%s)", url)
		return ErrIncompleteURL
	}
	return nil
}

func (uc *ShortenerUC) generateLinkUID(length int) string {
	var id string

	for i := 0; i < length; i++ {
		random := uc.rng.Intn(len(uc.alphabet))
		id += string(uc.alphabet[random])
	}

	return id
}

func (uc *ShortenerUC) GetShortlink(ctx context.Context, linkUID string) (*entity.Shortlink, error) {
	return uc.repo.FindShortlink(ctx, "", linkUID)
}

func (uc *ShortenerUC) GetUserShortlink(ctx context.Context, userUID, linkUID string) (*entity.Shortlink, error) {
	return uc.repo.FindShortlink(ctx, userUID, linkUID)
}

func (uc *ShortenerUC) ListUserShortlinks(ctx context.Context, userUID string) ([]*entity.Shortlink, error) {
	return uc.repo.GetShortlinks(ctx, userUID)
}
