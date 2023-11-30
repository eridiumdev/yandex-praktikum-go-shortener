package batch

import (
	"context"
	"time"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
)

type (
	Processor struct {
		repo repository.ShortlinkRepo
		log  *logger.Logger

		deleteShortlinksChan chan shortlinksBatch
	}
	shortlinksBatch struct {
		UserUID  string
		LinkUIDs []string
	}
)

func NewProcessor(ctx context.Context, repo repository.ShortlinkRepo, log *logger.Logger) *Processor {
	p := &Processor{
		repo: repo,
		log:  log,

		deleteShortlinksChan: make(chan shortlinksBatch),
	}
	go p.bufferShortlinksForDelete(ctx)

	return p
}

func (p *Processor) BatchDeleteShortlinks(ctx context.Context, userUID string, linkUIDs []string) {
	p.deleteShortlinksChan <- shortlinksBatch{UserUID: userUID, LinkUIDs: linkUIDs}
}

func (p *Processor) bufferShortlinksForDelete(ctx context.Context) {
	buffer := make(map[string][]string)
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case batch := <-p.deleteShortlinksChan:
			if _, ok := buffer[batch.UserUID]; !ok {
				buffer[batch.UserUID] = batch.LinkUIDs
			} else {
				buffer[batch.UserUID] = append(buffer[batch.UserUID], batch.LinkUIDs...)
			}
		case <-ticker.C:
			for userUID, linksUIDs := range buffer {
				err := p.repo.DeleteShortlinks(ctx, userUID, linksUIDs)
				if err != nil {
					p.log.Error(ctx, err).Msg("delete shortlinks")
				} else {
					p.log.Info(ctx).Msgf("delete %d shortlinks for user %s", len(linksUIDs), userUID)
				}
				delete(buffer, userUID)
			}
		}
	}
}
