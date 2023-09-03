package api

import (
	"net/http"
	"os"
	"sync"

	"github.com/mrbanja/watchparty-vlc-client/tools/renderer"

	"github.com/mrbanja/watchparty-vlc-client/pkg/web"

	"github.com/mrbanja/watchparty-vlc-client/worker"

	"go.uber.org/zap"

	"github.com/mrbanja/watchparty-vlc-client/ui"
)

type API struct {
	worker     *worker.Worker
	progress   ui.Progress
	serverAddr string
	wsClient   *web.Client

	renderer *renderer.Engine

	mu               sync.Mutex
	isServiceRunning bool

	logger *zap.Logger
}

func MustNew(
	worker *worker.Worker,
	wsClient *web.Client,
	serverAddr string,
	progress ui.Progress,
	logger *zap.Logger,
) *API {
	ren := renderer.New("./static", ".html")
	_ = ren.Load()

	return &API{
		worker:     worker,
		logger:     logger,
		wsClient:   wsClient,
		serverAddr: serverAddr,
		progress:   progress,
		renderer:   ren,
	}
}

type m map[string]interface{}

func (a *API) Index(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isServiceRunning {
		_ = a.renderer.Render(w, "index", m{"IsServerRunning": true, "PartID": a.wsClient.ID, "ID": "party", "ServerAddr": a.serverAddr})
		return
	}

	// err := a.template.Lookup("index").Execute(w, m{"ToIgnoreOutput": true, "PartID": a.wsClient.ID, "ID": "party", "ServerAddr": a.serverAddr})
	err := a.renderer.Render(w, "index", m{"ToIgnoreOutput": true, "PartID": a.wsClient.ID, "ID": "party", "ServerAddr": a.serverAddr})
	if err != nil {
		a.logger.Warn("ERR EX", zap.Error(err))
	}
}

func (a *API) Begin(w http.ResponseWriter, r *http.Request) {
	path := r.FormValue("path")
	stat, err := os.Stat(path)
	if err != nil {
		_ = a.renderer.Render(w, "form", m{"State": "", "PathValue": "", "Error": "Folder not found"})
		return
	}
	if !stat.IsDir() {
		_ = a.renderer.Render(w, "form", m{"State": "", "PathValue": "", "Error": "Is not a folder"})
		return
	}

	go func() {
		a.mu.Lock()
		if a.isServiceRunning {
			a.mu.Unlock()
			return
		}
		a.isServiceRunning = true
		a.mu.Unlock()
		if err := a.worker.Run(path); err != nil {
			a.logger.Panic("Service error", zap.Error(err))
		}
		a.mu.Lock()
		a.isServiceRunning = false
		a.mu.Unlock()
	}()
	_ = a.renderer.Render(w, "form", m{"Percent": 0, "State": "disabled", "PathValue": path})
}

func (a *API) DownloadProgress(w http.ResponseWriter, r *http.Request) {
	if a.progress.Get() >= 100 {
		w.Header().Set("HX-Trigger", "DownloadDone")
		_ = a.renderer.Render(w, "progress_bar", m{"Percent": 100})
		return
	}
	_ = a.renderer.Render(w, "progress_bar", m{"Percent": a.progress.Get()})
}

func (a *API) DownloadDone(w http.ResponseWriter, r *http.Request) {
	_ = a.renderer.Render(w, "download_done", nil)
}
