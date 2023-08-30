package api

import (
	"os"
	"sync"
	"time"

	"github.com/mrbanja/watchparty-vlc-client/pkg/web"

	"github.com/mrbanja/watchparty-vlc-client/worker"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/mrbanja/watchparty-vlc-client/ui"
)

type API struct {
	worker     *worker.Worker
	progress   ui.Progress
	serverAddr string
	wsClient   *web.Client

	mu               sync.Mutex
	isServiceRunning bool

	logger *zap.Logger
}

func New(
	worker *worker.Worker,
	wsClient *web.Client,
	serverAddr string,
	progress ui.Progress,
	logger *zap.Logger,
) *API {
	return &API{
		worker:     worker,
		logger:     logger,
		wsClient:   wsClient,
		serverAddr: serverAddr,
		progress:   progress,
	}
}

func (a *API) Index(c *fiber.Ctx) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isServiceRunning {
		return c.Render("index", fiber.Map{"IsServerRunning": true, "PartID": a.wsClient.ID, "ID": "party", "ServerAddr": a.serverAddr})
	}
	return c.Render("index", fiber.Map{"ToIgnoreOutput": true, "PartID": a.wsClient.ID, "ID": "party", "ServerAddr": a.serverAddr})
}

func (a *API) Begin(c *fiber.Ctx) error {
	stat, err := os.Stat(c.FormValue("path"))
	if err != nil {
		return c.Render("form", fiber.Map{"State": "", "PathValue": "", "Error": "Folder not found"})
	}
	if !stat.IsDir() {
		return c.Render("form", fiber.Map{"State": "", "PathValue": "", "Error": "Is not a folder"})
	}

	go func() {
		a.mu.Lock()
		a.isServiceRunning = true
		a.mu.Unlock()
		if err := a.worker.Run(c.FormValue("path")); err != nil {
			a.logger.Panic("Service error", zap.Error(err))
		}
		if err := c.App().ShutdownWithTimeout(time.Second * 5); err != nil {
			a.logger.Panic("Shutdown error", zap.Error(err))
		}
	}()
	return c.Render("form", fiber.Map{"Percent": 0, "State": "disabled", "PathValue": c.FormValue("path")})
}

func (a *API) DownloadProgress(c *fiber.Ctx) error {
	if a.progress.Get() >= 100 {
		c.Append("HX-Trigger", "DownloadDone")
		return c.Render("progress_bar", fiber.Map{"Percent": a.progress.Get()})
	}
	return c.Render("progress_bar", fiber.Map{"Percent": a.progress.Get()})
}

func (a *API) DownloadDone(c *fiber.Ctx) error {
	return c.Render("download_done", nil)
}
