package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mrbanja/watchparty-vlc-client/pkg/web"

	"go.uber.org/zap"

	"github.com/mrbanja/watchparty-vlc-client/api"
	"github.com/mrbanja/watchparty-vlc-client/app"
	"github.com/mrbanja/watchparty-vlc-client/pkg/controller"
	"github.com/mrbanja/watchparty-vlc-client/pkg/torrents"
	"github.com/mrbanja/watchparty-vlc-client/pkg/vlc"
	"github.com/mrbanja/watchparty-vlc-client/worker"
)

type ProgressBar struct {
	percent int
}

func (p *ProgressBar) Set(percent int) {
	p.percent = percent
}

func (p *ProgressBar) Get() int {
	return p.percent
}

var progress = &ProgressBar{percent: 0}

func mustSetupLogs() *zap.Logger {
	zc := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      false,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := zc.Build()
	if err != nil {
		panic(err)
	}
	return logger
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
			time.Sleep(10 * time.Minute)
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := mustSetupLogs()

	options := app.MustLoadOptions(logger)

	wCtx, wCancel := context.WithCancel(ctx)
	defer wCancel()
	webClient := web.New(options.ServerAddress, logger)
	webClient.MustJoinRoom(wCtx)
	vlcClient := vlc.MustNew(ctx, logger)

	contr := controller.New(
		vlcClient,
		webClient,
		torrents.New(logger, progress),
		logger,
	)

	srv := api.MustNew(worker.New(contr, progress), webClient, options.ServerAddress, progress, logger)

	if err := app.Run(srv, options, logger); err != nil {
		logger.Error("Error while listen", zap.Error(err))
	}
}
