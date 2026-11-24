//go:build !server

package cli

import (
	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/zajca/zfaktury/internal/desktop"
	"github.com/zajca/zfaktury/internal/epo"
	"github.com/zajca/zfaktury/internal/server"
)

var desktopPort int

func init() {
	rootCmd.Flags().IntVar(&desktopPort, "port", 0, "Accepted but unused in desktop mode (the window serves the router directly); use 'serve --port' for the HTTP server")
}

func runDesktop(cmd *cobra.Command, args []string) error {
	srvApp, err := server.New(server.Options{
		ConfigFile: configFile,
		InitConfig: true,
		Port:       portOverride(cmd, desktopPort),
	})
	if err != nil {
		return err
	}

	downloadSvc := &desktop.DownloadService{Handler: srvApp.Router()}
	// The client resolves the configured allowed host against the package default.
	browserSvc := &desktop.BrowserService{
		AllowedHost: epo.NewClient(epo.WithAllowedHost(srvApp.Config().EPO.AllowedHost)).AllowedHost(),
	}

	// Wails serves requests through AssetOptions.Handler directly,
	// no separate HTTP server needed.
	app := application.New(application.Options{
		Name: "ZFaktury",
		Assets: application.AssetOptions{
			Handler: srvApp.Router(),
		},
		Services: []application.Service{
			application.NewService(downloadSvc),
			application.NewService(browserSvc),
		},
		OnShutdown: func() {
			srvApp.Close()
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "ZFaktury",
		Width:  1280,
		Height: 800,
		URL:    "/",
	})

	return app.Run()
}
