package cmd

import (
	"sort"
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

func timeToSlot(t time.Time) uint64 {
	genesis := 1_606_824_023
	return uint64((t.Unix() - int64(genesis)) / 12)
}

func slotToTime(slot uint64) time.Time {
	genesis := 1_606_824_023
	timestamp := (slot * 12) + uint64(genesis)
	return time.Unix(int64(timestamp), 0).UTC()
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

		slotStart := timeToSlot(timeStart)
		slotEnd := timeToSlot(timeEnd)
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

		hourBucket := make(map[string]map[string]*database.BuilderStatsEntry)
		for _, entry := range entriesBySlot {
			builderID := entry.ExtraData

			t := slotToTime(entry.Slot)
			hour := t.Format("2006-01-02 15")
			// log.Infof("slot: %d, hour: %s", entry.Slot, hour)
			if _, ok := hourBucket[hour]; !ok {
				hourBucket[hour] = make(map[string]*database.BuilderStatsEntry)
			}

			if _, ok := hourBucket[hour][builderID]; !ok {
				bucketStartTime, err := time.Parse("2006-01-02 15", hour)
				check(err)
				bucketEndTime := bucketStartTime.Add(time.Hour)
				hourBucket[hour][builderID] = &database.BuilderStatsEntry{
					Hours:     1,
					TimeStart: bucketStartTime,
					TimeEnd:   bucketEndTime,
				}
			}
			hourBucket[hour][builderID].BlocksIncluded += 1
		}

		// sort hourBucket keys alphabetically
		var hours []string
		for hour := range hourBucket {
			hours = append(hours, hour)
		}
		sort.Strings(hours)

		// print
		for _, hour := range hours {
			builderStats := hourBucket[hour]
			log.Infof("hour: %s", hour)
			for builderID, stats := range builderStats {
				log.Infof("- %34s: %4d blocks", builderID, stats.BlocksIncluded)
			}
		}
	},
}
