package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zajca/zfaktury/internal/server"
)

var (
	servePort       int
	serveDev        bool
	serveInitConfig bool
)

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "HTTP server port")
	serveCmd.Flags().BoolVar(&serveDev, "dev", false, "Enable development mode (proxy frontend to Vite)")
	serveCmd.Flags().BoolVar(&serveInitConfig, "init-config", false, "Create a default config file if it does not exist")
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long:  "Start the ZFaktury HTTP server serving both the API and the frontend.",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := server.New(server.Options{
			ConfigFile: configFile,
			InitConfig: serveInitConfig,
			Port:       servePort,
			DevMode:    serveDev,
		})
		if err != nil {
			return err
		}
		defer func() { _ = app.Close() }()

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		return app.ListenAndServe(ctx)
	},
}
