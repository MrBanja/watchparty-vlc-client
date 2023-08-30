package torrents

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent"
	"go.uber.org/zap"

	"github.com/mrbanja/watchparty-vlc-client/ui"
)

type Client struct {
	logger *zap.Logger
	setter ui.Progress
}

func New(logger *zap.Logger, setter ui.Progress) *Client {
	return &Client{
		logger: logger.Named("TORRENT"),
		setter: setter,
	}
}

func (c *Client) EnforceLogger(logger *zap.Logger) {
	c.logger = logger.Named("TORRENT")
}

func (c *Client) Download(
	ctx context.Context,
	magnet string,
	dataDir string,
) (string, error) {
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = dataDir

	trrCli, err := torrent.NewClient(cfg)
	if err != nil {
		c.logger.Error("Client creation error", zap.Error(err))
		return "", err
	}
	defer func() {
		errs := trrCli.Close()
		if errs != nil {
			c.logger.Error("Client closed: Errors: ", zap.Error(errors.Join(errs...)))
		}
	}()

	t, err := trrCli.AddMagnet(magnet)
	if err != nil {
		c.logger.Error("Add magnet err", zap.Error(err))
		return "", err
	}

	c.logger.Info("Getting torrent info")
	<-t.GotInfo()
	c.logger.Info("Got torrent info")
	fp := filepath.Join(cfg.DataDir, t.Files()[0].Path())

	if t.Complete.Bool() {
		c.logger.Info("Torrent already downloaded")
		c.setter.Set(100)
		return fp, nil
	}

	c.logger.Info("Begin downloading")
	t.DownloadAll()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				percent := countPercent(t)
				c.logger.Info(fmt.Sprintf(
					"%s, %vMB / %vMB [%v%%]",
					t.Name(),
					t.BytesCompleted()/1024/1024,
					(t.BytesMissing()+t.BytesCompleted())/1024/1024,
					percent,
				))

				c.setter.Set(percent)
			case <-ctx.Done():
				return
			}
		}
	}()

	trrCli.WaitAll()
	c.logger.Info("Torrent downloaded")
	c.setter.Set(100)
	cancel()
	return fp, nil
}

func countPercent(t *torrent.Torrent) int {
	m := t.BytesMissing()
	c := t.BytesCompleted()
	total := m + c
	return int((100 * c) / total)
}
