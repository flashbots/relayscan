package service

/**
 * https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md
 */

import (
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/services/bidcollect"
	"github.com/flashbots/relayscan/services/bidcollect/website"
	"github.com/flashbots/relayscan/vars"
	"github.com/lithammer/shortuuid"
	"github.com/spf13/cobra"
)

var (
	collectTopBidWebsocketStream bool
	collectGetHeader             bool
	collectDataAPI               bool
	useAllRelays                 bool

	outDir    string
	outputTSV bool   // by default: CSV, but can be changed to TSV with this setting
	uid       string // used in output filenames, to avoid collissions between multiple collector instances

	runDevServerOnly    bool // used to play with file listing website
	devServerListenAddr string

	buildWebsite       bool
	buildWebsiteUpload bool
	buildWebsiteOutDir string
)

func init() {
	bidCollectCmd.Flags().BoolVar(&collectTopBidWebsocketStream, "top-bid-ws-stream", false, "use top-bid websocket streams")
	bidCollectCmd.Flags().BoolVar(&collectGetHeader, "get-header", false, "use getHeader API")
	bidCollectCmd.Flags().BoolVar(&collectDataAPI, "data-api", false, "use data API")
	bidCollectCmd.Flags().BoolVar(&useAllRelays, "all-relays", false, "use all relays")

	// for getHeader
	bidCollectCmd.Flags().StringVar(&beaconNodeURI, "beacon-uri", vars.DefaultBeaconURI, "beacon endpoint")

	// for saving to file
	bidCollectCmd.Flags().StringVar(&outDir, "out", "csv", "output directory for CSV/TSV")
	bidCollectCmd.Flags().BoolVar(&outputTSV, "out-tsv", false, "output as TSV (instead of CSV)")

	// utils
	bidCollectCmd.Flags().StringVar(&uid, "uid", "", "unique identifier for output files (to avoid collisions)")

	// for dev purposes
	bidCollectCmd.Flags().BoolVar(&runDevServerOnly, "devserver", false, "only run devserver to play with file listing website")
	bidCollectCmd.Flags().StringVar(&devServerListenAddr, "devserver-addr", "localhost:8095", "listen address for devserver")

	// building the S3 website
	bidCollectCmd.Flags().BoolVar(&buildWebsite, "build-website", false, "build file listing website")
	bidCollectCmd.Flags().BoolVar(&buildWebsiteUpload, "build-website-upload", false, "upload after building")
	bidCollectCmd.Flags().StringVar(&buildWebsiteOutDir, "build-website-out", "build", "output directory for website")
}

var bidCollectCmd = &cobra.Command{
	Use:   "bidcollect",
	Short: "Collect bids",
	Run: func(cmd *cobra.Command, args []string) {
		if runDevServerOnly {
			log.Infof("Bidcollect %s devserver starting on %s ...", vars.Version, devServerListenAddr)
			fileListingDevServer()
			return
		}

		if buildWebsite {
			log.Infof("Bidcollect %s building website (output: %s) ...", vars.Version, buildWebsiteOutDir)
			website.BuildProdWebsite(log, buildWebsiteOutDir, buildWebsiteUpload)
			return
		}

		if uid == "" {
			uid = shortuuid.New()[:6]
		}

		log.WithField("uid", uid).Infof("Bidcollect %s starting ...", vars.Version)

		// Prepare relays
		relays := []common.RelayEntry{
			common.MustNewRelayEntry(vars.RelayFlashbots, false),
			common.MustNewRelayEntry(vars.RelayUltrasound, false),
		}
		if useAllRelays {
			relays = common.MustGetRelays()
		}

		log.Infof("Using %d relays", len(relays))
		for index, relay := range relays {
			log.Infof("- relay #%d: %s", index+1, relay.Hostname())
		}

		topBidRelays := []common.RelayEntry{}
		for _, url := range vars.TopBidStreamURLs {
			topBidRelays = append(topBidRelays, common.MustNewRelayEntry(url, false))
		}

		opts := bidcollect.BidCollectorOpts{
			Log:                          log,
			UID:                          uid,
			Relays:                       relays,
			CollectTopBidWebsocketStream: collectTopBidWebsocketStream,
			TopBidWebsocketRelays:        topBidRelays,
			CollectGetHeader:             collectGetHeader,
			CollectDataAPI:               collectDataAPI,
			BeaconNodeURI:                beaconNodeURI,
			OutDir:                       outDir,
			OutputTSV:                    outputTSV,
		}

		bidCollector := bidcollect.NewBidCollector(&opts)
		bidCollector.MustStart()
	},
}

func fileListingDevServer() {
	webserver, err := website.NewDevWebserver(&website.DevWebserverOpts{ //nolint:exhaustruct
		ListenAddress: devServerListenAddr,
		Log:           log,
	})
	if err != nil {
		log.Fatal(err)
	}
	err = webserver.StartServer()
	if err != nil {
		log.Fatal(err)
	}
}
