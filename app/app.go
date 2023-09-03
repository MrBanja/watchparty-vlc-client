package app

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/mrbanja/watchparty-vlc-client/logging"

	"github.com/gorilla/mux"
	"github.com/mrbanja/watchparty-vlc-client/tools/http_server"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/mrbanja/watchparty-vlc-client/api"
)

type Options struct {
	ServerAddress string `yaml:"server_address"`
	LocalAddress  string `yaml:"local_address"`
}

func MustLoadOptions(logger *zap.Logger) Options {
	f, err := os.ReadFile("config.yaml")
	if err != nil {
		logger.Panic("Config reading error", zap.Error(err))
	}
	o := Options{}
	if err := yaml.Unmarshal(f, &o); err != nil {
		logger.Panic("Error while parse env", zap.Error(err))
	}
	return o
}

func Run(srv *api.API, opt Options, logger *zap.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := mux.NewRouter()
	handler.HandleFunc("/", srv.Index).Methods(http.MethodGet)
	handler.HandleFunc("/download/progress", srv.DownloadProgress).Methods(http.MethodGet)
	handler.HandleFunc("/download/done", srv.DownloadDone).Methods(http.MethodGet)
	handler.HandleFunc("/begin", srv.Begin).Methods(http.MethodPost)

	handler.Use(logging.Middleware(logger))

	server := &http.Server{
		Addr: opt.LocalAddress,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Handler: handler,
	}

	logger.Info("[*] Http server started", zap.String("Pub Addr", opt.LocalAddress), zap.String("Local Addr", opt.LocalAddress))
	if err := http_server.Serve(ctx, server, 10*time.Second, func(server *http.Server) error {
		return server.ListenAndServe()
	}); err != nil {
		return err
	}

	return nil
}
