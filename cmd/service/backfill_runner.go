package service

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/flashbots/relayscan/cmd/core"
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/database"
	"github.com/flashbots/relayscan/vars"
	"github.com/spf13/cobra"
)

var (
	runnerInterval       time.Duration
	runnerEthNodeURI     string
	runnerEthBackupURI   string
	runnerLimit          uint64
	runnerNumThreads     uint64
	runnerRunOnce        bool
	runnerSkipBackfill   bool
	runnerSkipCheckValue bool
	runnerRelay          string
	runnerMinSlot        int64
)

func init() {
	backfillRunnerCmd.Flags().DurationVar(&runnerInterval, "interval", 5*time.Minute, "interval between runs")
	backfillRunnerCmd.Flags().StringVar(&runnerEthNodeURI, "eth-node", vars.DefaultEthNodeURI, "eth node URI")
	backfillRunnerCmd.Flags().StringVar(&runnerEthBackupURI, "eth-node-backup", vars.DefaultEthBackupNodeURI, "eth backup node URI")
	backfillRunnerCmd.Flags().Uint64Var(&runnerLimit, "limit", 1000, "limit for check-payload-value")
	backfillRunnerCmd.Flags().Uint64Var(&runnerNumThreads, "threads", 10, "number of threads for check-payload-value")
	backfillRunnerCmd.Flags().BoolVar(&runnerRunOnce, "once", false, "run once and exit")
	backfillRunnerCmd.Flags().BoolVar(&runnerSkipBackfill, "skip-backfill", false, "skip data-api-backfill step")
	backfillRunnerCmd.Flags().BoolVar(&runnerSkipCheckValue, "skip-check-value", false, "skip check-payload-value step")
	backfillRunnerCmd.Flags().StringVar(&runnerRelay, "relay", "", "specific relay only (e.g. 'fb', 'us', or full URL)")
	backfillRunnerCmd.Flags().Int64Var(&runnerMinSlot, "min-slot", 0, "minimum slot (negative for offset from latest)")
}

var backfillRunnerCmd = &cobra.Command{
	Use:   "backfill-runner",
	Short: "Continuously run data-api-backfill and check-payload-value",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var relays []common.RelayEntry

		log.Infof("Relayscan backfill-runner %s starting...", vars.Version)
		log.Infof("Interval: %s", runnerInterval)

		// Get relays
		if runnerRelay != "" {
			var relayEntry common.RelayEntry
			switch runnerRelay {
			case "fb":
				relayEntry, err = common.NewRelayEntry(vars.RelayURLs[0], false)
			case "us":
				relayEntry, err = common.NewRelayEntry(vars.RelayURLs[1], false)
			default:
				relayEntry, err = common.NewRelayEntry(runnerRelay, false)
			}
			if err != nil {
				log.WithField("relay", runnerRelay).WithError(err).Fatal("failed to decode relay")
			}
			relays = []common.RelayEntry{relayEntry}
		} else {
			relays, err = common.GetRelays()
			if err != nil {
				log.WithError(err).Fatal("failed to get relays")
			}
		}
		log.Infof("Using %d relays", len(relays))
		for i, relay := range relays {
			log.Infof("- relay #%d: %s", i+1, relay.Hostname())
		}

		if runnerMinSlot != 0 {
			log.Infof("Using min-slot: %d", runnerMinSlot)
		}

		// Connect to Postgres
		db := database.MustConnectPostgres(log, vars.DefaultPostgresDSN)

		// Connect to eth nodes
		var ethClient, ethClient2 *ethclient.Client
		if !runnerSkipCheckValue {
			if runnerEthNodeURI == "" {
				log.Fatal("eth-node is required for check-payload-value")
			}
			ethClient, err = ethclient.Dial(runnerEthNodeURI)
			if err != nil {
				log.WithError(err).Fatalf("failed to connect to eth node: %s", runnerEthNodeURI)
			}
			log.Infof("Connected to eth node: %s", runnerEthNodeURI)

			ethClient2 = ethClient
			if runnerEthBackupURI != "" {
				ethClient2, err = ethclient.Dial(runnerEthBackupURI)
				if err != nil {
					log.WithError(err).Fatalf("failed to connect to backup eth node: %s", runnerEthBackupURI)
				}
				log.Infof("Connected to backup eth node: %s", runnerEthBackupURI)
			}
		}

		// Prepare check-payload-value options
		checkOpts := core.CheckPayloadValueOpts{
			Limit:      runnerLimit,
			NumThreads: runnerNumThreads,
		}

		// Run function
		runBackfillCycle := func() {
			log.Info("Starting backfill cycle...")

			// Step 1: data-api-backfill
			if !runnerSkipBackfill {
				log.Info("Running data-api-backfill...")
				err := core.RunBackfill(db, relays, 0, runnerMinSlot)
				if err != nil {
					log.WithError(err).Error("data-api-backfill failed")
				}
			}

			// Step 2: check-payload-value
			if !runnerSkipCheckValue {
				log.Info("Running check-payload-value...")
				err := core.RunCheckPayloadValue(db, ethClient, ethClient2, checkOpts)
				if err != nil {
					log.WithError(err).Error("check-payload-value failed")
				}
			}

			log.Info("Backfill cycle complete")
		}

		// Run once immediately
		runBackfillCycle()

		if runnerRunOnce {
			log.Info("Run once mode, exiting")
			return
		}

		// Set up signal handling
		sigC := make(chan os.Signal, 1)
		signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)

		// Run on interval
		ticker := time.NewTicker(runnerInterval)
		defer ticker.Stop()

		log.Infof("Waiting for next run in %s...", runnerInterval)

		for {
			select {
			case <-ticker.C:
				runBackfillCycle()
				log.Infof("Waiting for next run in %s...", runnerInterval)
			case sig := <-sigC:
				log.Infof("Received signal %s, shutting down...", sig)
				return
			}
		}
	},
}
