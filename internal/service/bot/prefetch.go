package bot

import (
	"context"
	"log"
	"time"

	"ps4rpc/internal/source/ps4"
)

const (
	defaultRefresh = 30 * time.Minute
	iconMaxAge     = 24 * time.Hour
	probeEvery     = 20 * time.Second
)

func (b *Bot) refreshEvery() time.Duration {
	if b.cache.Refresh <= 0 {
		return defaultRefresh
	}
	return time.Duration(b.cache.Refresh) * time.Minute
}

func (b *Bot) warmLoop(ctx context.Context) {
	for {
		wait := b.refreshEvery()
		if !b.warm(ctx) {
			wait = probeEvery
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

func (b *Bot) warm(ctx context.Context) bool {
	if err := b.info.Check(); err != nil {
		return false
	}
	started := time.Now()
	before, _ := b.info.Cache().Stats()

	if _, _, err := b.info.Username(); err != nil {
		log.Printf("cache: username: %v", err)
	}
	if _, _, err := b.info.Avatar(); err != nil {
		log.Printf("cache: avatar: %v", err)
	}

	games, _, err := b.info.Games()
	if err != nil {
		log.Printf("cache: library: %v", err)
		return false
	}

	titles, _, err := b.info.TrophyTitles()
	if err != nil {
		log.Printf("cache: trophies: %v", err)
	}
	for _, t := range titles {
		if ctx.Err() != nil {
			return false
		}
		if _, _, err := b.info.Trophies(t.CommID); err != nil {
			log.Printf("cache: trophies %s: %v", t.CommID, err)
			break
		}
	}

	for _, g := range games {
		if ctx.Err() != nil {
			return false
		}
		b.info.Playtime(g.TitleID)
	}

	if b.cache.Icons {
		for _, g := range games {
			if ctx.Err() != nil {
				return false
			}
			if !b.iconStale(g.TitleID) {
				continue
			}
			if _, _, err := b.info.Icon(g.TitleID); err != nil {
				continue
			}
		}
	}

	if ctx.Err() != nil {
		return false
	}
	if _, _, _, err := b.info.LatestScreenshot(); err != nil {
		log.Printf("cache: screenshot: %v", err)
	}

	after, bytes := b.info.Cache().Stats()
	log.Printf("cache: warmed %d new files (%d total, %s) in %s",
		after-before, after, humanBytes(bytes), time.Since(started).Round(time.Second))
	return true
}

func (b *Bot) iconStale(titleID string) bool {
	age, ok := b.info.Cache().Age(ps4.IconPath(titleID))
	return !ok || age > iconMaxAge
}
