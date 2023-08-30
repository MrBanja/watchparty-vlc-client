package app

import (
	"os"

	"github.com/gofiber/contrib/fiberzap"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
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

func Run(srv *api.API, o Options, logger *zap.Logger) error {
	engine := html.New("./static", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})
	app.Use(fiberzap.New(fiberzap.Config{
		Logger: logger,
	}))

	app.Get("/", srv.Index)
	app.Get("/download/progress", srv.DownloadProgress)
	app.Get("/download/done", srv.DownloadDone)
	app.Post("/begin", srv.Begin)

	return app.Listen(o.LocalAddress)
}
