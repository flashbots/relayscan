package service

import (
	"os"

	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/database"
	"github.com/flashbots/relayscan/services/website"
	"github.com/spf13/cobra"
)

var (
	websiteDefaultListenAddr = relaycommon.GetEnv("LISTEN_ADDR", "localhost:9060")
	websiteListenAddr        string
	websiteDev               = os.Getenv("DEV") == "1"
)

func init() {
	// rootCmd.AddCommand(websiteCmd)
	websiteCmd.Flags().StringVar(&websiteListenAddr, "listen-addr", websiteDefaultListenAddr, "listen address for webserver")
	websiteCmd.Flags().BoolVar(&websiteDev, "dev", websiteDev, "development mode")
}

var websiteCmd = &cobra.Command{
	Use:   "website",
	Short: "Start the website server",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		// Connect to Postgres
		db := database.MustConnectPostgres(log, common.DefaultPostgresDSN)

		// Create the website service
		opts := &website.WebserverOpts{
			ListenAddress: websiteListenAddr,
			DB:            db,
			Log:           log,
			Dev:           websiteDev,
		}

		srv, err := website.NewWebserver(opts)
		if err != nil {
			log.WithError(err).Fatal("failed to create service")
		}

		// Start the server
		log.Infof("Webserver starting on %s ...", websiteListenAddr)
		log.Fatal(srv.StartServer())
	},
}
