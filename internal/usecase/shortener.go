package usecase

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
)

type ShortenerUC struct {
	baseUrl       string
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
		baseUrl:       cfg.BaseUrl,
		defaultLength: cfg.DefaultLength,
		alphabet:      alphabet,
		repo:          repo,
		rng:           rand.New(rand.NewSource(time.Now().Unix())),
	}
}

func (uc *ShortenerUC) CreateShortlink(ctx context.Context, length int, url string) (*entity.Shortlink, error) {
	if length <= 0 {
		length = uc.defaultLength
	}

	var id string

	for tries := 0; ; tries++ {
		id = uc.generateID(length)
		exists, err := uc.repo.FindShortlink(ctx, id)
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
		ID:    id,
		Long:  url,
		Short: uc.baseUrl + id,
	}

	log.Printf("URL shortened: %s -> %s", link.Long, link.Short)

	return link, uc.repo.SaveShortlink(ctx, link)
}

func (uc *ShortenerUC) generateID(length int) string {
	var id string

	for i := 0; i < length; i++ {
		random := uc.rng.Intn(len(uc.alphabet))
		id += string(uc.alphabet[random])
	}

	return id
}

func (uc *ShortenerUC) GetShortlink(ctx context.Context, id string) (*entity.Shortlink, error) {
	return uc.repo.FindShortlink(ctx, id)
}
