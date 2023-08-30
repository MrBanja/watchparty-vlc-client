package worker

import (
	"context"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/mrbanja/watchparty-vlc-client/pkg/controller"
	"github.com/mrbanja/watchparty-vlc-client/ui"
)

type Worker struct {
	controller *controller.Controller
	progress   ui.Progress

	logger *zap.Logger
}

func New(controller *controller.Controller, progress ui.Progress) *Worker {
	return &Worker{
		controller: controller,
		progress:   progress,
	}
}

func (w *Worker) Run(dirPath string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := w.init(dirPath); err != nil {
		w.logger.Error("Logging init error", zap.Error(err))
		return err
	}
	w.controller.EnforceLogger(w.logger)

	if err := w.controller.Run(ctx, controller.Config{DownloadDir: dirPath}); err != nil {
		w.logger.Error("Controller error", zap.Error(err))
		return err
	}

	return nil
}

func (w *Worker) init(dirPath string) error {
	zc := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      false,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr", filepath.Join(dirPath, "app.log")},
		ErrorOutputPaths: []string{"stderr", filepath.Join(dirPath, "app.log")},
	}

	logger, err := zc.Build()
	if err != nil {
		return err
	}

	logger.Warn("=====================START=====================\n\n")
	logger.Info("Logic startup", zap.String("path", dirPath))
	w.logger = logger
	return nil
}
