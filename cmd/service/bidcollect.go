package service

/**
 * https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md
 */

import (
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/services/bidcollect"
	"github.com/flashbots/relayscan/services/bidcollect/website"
	"github.com/flashbots/relayscan/vars"
	"github.com/spf13/cobra"
)

var (
	collectUltrasoundStream bool
	collectGetHeader        bool
	collectDataAPI          bool
	useAllRelays            bool

	outDir    string
	outputTSV bool // by default: CSV, but can be changed to TSV with this setting

	runDevServerOnly    bool // used to play with file listing website
	devServerListenAddr = ":8095"

	buildWebsite       bool
	buildWebsiteUpload bool
	buildWebsiteOutDir string
)

func init() {
	bidCollectCmd.Flags().BoolVar(&collectUltrasoundStream, "ultrasound-stream", false, "use ultrasound top-bid stream")
	bidCollectCmd.Flags().BoolVar(&collectGetHeader, "get-header", false, "use getHeader API")
	bidCollectCmd.Flags().BoolVar(&collectDataAPI, "data-api", false, "use data API")
	bidCollectCmd.Flags().BoolVar(&useAllRelays, "all-relays", false, "use all relays")

	// for getHeader
	bidCollectCmd.Flags().StringVar(&beaconNodeURI, "beacon-uri", vars.DefaultBeaconURI, "beacon endpoint")

	// for saving to file
	bidCollectCmd.Flags().StringVar(&outDir, "out", "csv", "output directory for CSV/TSV")
	bidCollectCmd.Flags().BoolVar(&outputTSV, "out-tsv", false, "output as TSV (instead of CSV)")

	// for dev purposes
	bidCollectCmd.Flags().BoolVar(&runDevServerOnly, "devserver", false, "only run devserver to play with file listing website")

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
			log.Infof("Bidcollect devserver starting (%s) ...", vars.Version)
			fileListingDevServer()
			return
		}

		if buildWebsite {
			log.Infof("Bidcollect %s building website (output: %s) ...", vars.Version, buildWebsiteOutDir)
			website.BuildProdWebsite(log, buildWebsiteOutDir, buildWebsiteUpload)
			return
		}

		log.Infof("Bidcollect starting (%s) ...", vars.Version)

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

		opts := bidcollect.BidCollectorOpts{
			Log:                     log,
			Relays:                  relays,
			CollectUltrasoundStream: collectUltrasoundStream,
			CollectGetHeader:        collectGetHeader,
			CollectDataAPI:          collectDataAPI,
			BeaconNodeURI:           beaconNodeURI,
			OutDir:                  outDir,
			OutputTSV:               outputTSV,
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
