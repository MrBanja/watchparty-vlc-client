package controller

import (
	"context"
	"fmt"

	"github.com/mrbanja/watchparty-vlc-client/pkg/web"

	"github.com/mrbanja/watchparty-vlc-client/pkg/torrents"
	"github.com/mrbanja/watchparty-vlc-client/pkg/vlc"
	"go.uber.org/zap"
)

type Config struct {
	DownloadDir string
}

type Controller struct {
	vlc           *vlc.VLC
	webClient     *web.Client
	torrentClient *torrents.Client
	logger        *zap.Logger
}

func New(
	vlc *vlc.VLC,
	webClient *web.Client,
	torrentClient *torrents.Client,
	logger *zap.Logger,
) *Controller {
	return &Controller{
		vlc:           vlc,
		webClient:     webClient,
		logger:        logger.Named("CONTROLLER"),
		torrentClient: torrentClient,
	}
}

func (c *Controller) EnforceLogger(logger *zap.Logger) {
	logger = logger.With(zap.String("ClientID", c.webClient.ID))
	c.webClient.EnforceLogger(logger)
	c.torrentClient.EnforceLogger(logger)
	c.vlc.EnforceLogger(logger)
	c.logger = logger.Named("CONTROLLER")
}

func (c *Controller) Run(ctx context.Context, cfg Config) error {
	magnet, err := c.webClient.GetMagnet(ctx)
	if err != nil {
		return err
	}

	filepath, err := c.torrentClient.Download(ctx, magnet, cfg.DownloadDir)
	if err != nil {
		return err
	}
	if err := c.vlc.Add(filepath); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	return c.Serve(ctx)
}

func (c *Controller) Serve(ctx context.Context) error {
	resp, err := c.webClient.Listen(ctx)
	if err != nil {
		return err
	}
	if err := c.vlc.PlayBy(vlc.BySystem); err != nil {
		return err
	}
	if err := c.vlc.PauseBy(vlc.BySystem); err != nil {
		return err
	}
	if err := c.vlc.Seek(0); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Context DONE received. Stopping controller")
			return nil
		case r, ok := <-resp:
			if !ok {
				c.logger.Warn("Channel closed. Stopping controller")
				return fmt.Errorf("web closed the channel")
			}
			c.logger.Info("Received status change", zap.Any("New status", r))
			switch r.Status {
			case web.Play:
				if err := c.vlc.PlayBy(vlc.ByNet); err != nil {
					return err
				}
			case web.Pause:
				if err := c.vlc.PauseBy(vlc.ByNet); err != nil {
					return err
				}
			}
			if err := c.vlc.Seek(r.Time); err != nil {
				return err
			}
		case s := <-c.vlc.StatusCh():
			c.logger.Info("Status changed", zap.String("To", s.State), zap.Object("Caller", s.By))
			if s.By != vlc.ByUser {
				continue
			}
			switch vlc.GetPlayingState(s.State) {
			case vlc.StateStopped:
				return c.webClient.Send(web.Message{Time: 0, Status: web.Pause})
			case vlc.StatePlaying:
				if err := c.webClient.Send(web.Message{Time: int(s.Time), Status: web.Play}); err != nil {
					return err
				}
				continue
			case vlc.StatePaused:
				if err := c.webClient.Send(web.Message{Time: int(s.Time), Status: web.Pause}); err != nil {
					return err
				}
				continue
			}
		}
	}
}
