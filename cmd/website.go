package cmd

import (
	"net/url"
	"os"

	"github.com/flashbots/mev-boost-relay/common"
	"github.com/flashbots/relayscan/database"
	"github.com/flashbots/relayscan/services/website"
	"github.com/spf13/cobra"
)

var (
	websiteDefaultListenAddr = common.GetEnv("LISTEN_ADDR", "localhost:9060")
	websiteListenAddr        string
	websiteDev               = os.Getenv("DEV") == "1"
)

func init() {
	rootCmd.AddCommand(websiteCmd)
	websiteCmd.Flags().StringVar(&websiteListenAddr, "listen-addr", websiteDefaultListenAddr, "listen address for webserver")
	websiteCmd.Flags().BoolVar(&websiteDev, "dev", websiteDev, "development mode")
}

var websiteCmd = &cobra.Command{
	Use:   "website",
	Short: "Start the website server",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		// Connect to Postgres
		dbURL, err := url.Parse(defaultPostgresDSN)
		if err != nil {
			log.WithError(err).Fatalf("couldn't read db URL")
		}
		log.Infof("Connecting to Postgres database at %s%s ...", dbURL.Host, dbURL.Path)
		db, err := database.NewDatabaseService(defaultPostgresDSN)
		if err != nil {
			log.WithError(err).Fatalf("Failed to connect to Postgres database at %s%s", dbURL.Host, dbURL.Path)
		}

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
