package cmd

import (
	"time"

	"github.com/flashbots/relayscan/database"
	"github.com/spf13/cobra"
)

var (
	// cliRelay   string
	// initCursor uint64
	// minSlot    uint64
	// bidsOnly   bool
	builderStatsDateStart string
	builderStatsDateEnd   string
)

func init() {
	rootCmd.AddCommand(updateBuilderStatsCmd)
	updateBuilderStatsCmd.Flags().StringVar(&builderStatsDateStart, "start", "", "yyyy-mm-dd hh:mm")
	updateBuilderStatsCmd.Flags().StringVar(&builderStatsDateEnd, "end", "", "yyyy-mm-dd hh:mm")
}

var updateBuilderStatsCmd = &cobra.Command{
	Use:   "update-builder-stats",
	Short: "Update builder stats",
	Run: func(cmd *cobra.Command, args []string) {
		if builderStatsDateStart == "" {
			log.Fatal("start date is required")
		} else if builderStatsDateEnd == "" {
			log.Fatal("end date is required")
		}

		layout1 := "2006-01-02"
		layout2 := "2006-01-02 15:04"
		timeStart, err := time.Parse(layout1, builderStatsDateStart)
		if err != nil {
			timeStart, err = time.Parse(layout2, builderStatsDateStart)
			check(err)
		}
		timeEnd, err := time.Parse(layout1, builderStatsDateEnd)
		if err != nil {
			timeEnd, err = time.Parse(layout2, builderStatsDateEnd)
			check(err)
		}

		log.Infof("update builder stats: %s -> %s ", timeStart.String(), timeEnd.String())

		genesis := 1_606_824_023
		slotStart := (timeStart.Unix() - int64(genesis)) / 12
		slotEnd := (timeEnd.Unix() - int64(genesis)) / 12
		log.Infof("slots: %d -> %d (%d total)", slotStart, slotEnd, slotEnd-slotStart)

		db := mustConnectPostgres(defaultPostgresDSN)
		log.Info("Connected to database")

		entries, err := db.GetDeliveredPayloadsForSlots(uint64(slotStart), uint64(slotEnd))
		check(err)
		log.Infof("Found %d delivered-payload entries in database", len(entries))

		entriesBySlot := make(map[uint64]*database.DataAPIPayloadDeliveredEntry)
		for _, entry := range entries {
			entriesBySlot[entry.Slot] = entry
		}
		log.Infof("Unique slots with payloads: %d", len(entriesBySlot))
	},
}
