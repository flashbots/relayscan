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
	builderStatsDateStart  string
	builderStatsDateEnd    string
	builderStatsSaveDaily  bool
	builderStatsSaveHourly bool
	builderStatsVerbose    bool
)

func init() {
	rootCmd.AddCommand(updateBuilderStatsCmd)
	updateBuilderStatsCmd.Flags().StringVar(&builderStatsDateStart, "start", "", "yyyy-mm-dd hh:mm")
	updateBuilderStatsCmd.Flags().StringVar(&builderStatsDateEnd, "end", "", "yyyy-mm-dd hh:mm")
	updateBuilderStatsCmd.Flags().BoolVar(&builderStatsSaveDaily, "daily", false, "save daily stats")
	updateBuilderStatsCmd.Flags().BoolVar(&builderStatsSaveHourly, "hourly", false, "save hourly stats")
	updateBuilderStatsCmd.Flags().BoolVar(&builderStatsVerbose, "verbose", false, "verbose output")
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
		if !builderStatsSaveDaily && !builderStatsSaveHourly {
			log.Fatal("at least one of --daily or --hourly is required")
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

		entries, err := db.GetDeliveredPayloadsForSlots(slotStart, slotEnd)
		check(err)
		log.Infof("Found %d delivered-payload entries in database", len(entries))

		entriesBySlot := make(map[uint64]*database.DataAPIPayloadDeliveredEntry)
		for _, entry := range entries {
			entriesBySlot[entry.Slot] = entry
		}
		log.Infof("Unique slots with payloads: %d", len(entriesBySlot))

		if builderStatsSaveDaily {
			saveDayBucket(db, entriesBySlot)
		}
		if builderStatsSaveHourly {
			saveHourBucket(db, entriesBySlot)
		}
	},
}

func saveHourBucket(db *database.DatabaseService, entriesBySlot map[uint64]*database.DataAPIPayloadDeliveredEntry) {
	hourBucket := make(map[string]map[string]*database.BuilderStatsEntry)
	for _, entry := range entriesBySlot {
		builderID := entry.ExtraData

		t := slotToTime(entry.Slot)
		hour := t.Format("2006-01-02 15")
		if _, ok := hourBucket[hour]; !ok {
			hourBucket[hour] = make(map[string]*database.BuilderStatsEntry)
		}

		if _, ok := hourBucket[hour][builderID]; !ok {
			bucketStartTime, err := time.Parse("2006-01-02 15", hour)
			check(err)
			bucketEndTime := bucketStartTime.Add(time.Hour)
			hourBucket[hour][builderID] = &database.BuilderStatsEntry{
				Hours:       1,
				TimeStart:   bucketStartTime,
				TimeEnd:     bucketEndTime,
				BuilderName: builderID,
			}
		}
		hourBucket[hour][builderID].BlocksIncluded += 1
	}

	// sort hourBucket keys alphabetically
	var hours []string //nolint:prealloc
	for hour := range hourBucket {
		hours = append(hours, hour)
	}
	sort.Strings(hours)

	// print
	for _, hour := range hours {
		builderStats := hourBucket[hour]
		log.Infof("hour: %s", hour)
		entries := make([]*database.BuilderStatsEntry, 0, len(builderStats))
		for builderID, stats := range builderStats {
			if builderStatsVerbose {
				log.Infof("- %34s: %4d blocks", builderID, stats.BlocksIncluded)
			}
			entries = append(entries, stats)
		}
		err := db.SaveBuilderStats(entries)
		check(err)
	}
}

func saveDayBucket(db *database.DatabaseService, entriesBySlot map[uint64]*database.DataAPIPayloadDeliveredEntry) {
	dayBucket := make(map[string]map[string]*database.BuilderStatsEntry)
	for _, entry := range entriesBySlot {
		builderID := entry.ExtraData

		t := slotToTime(entry.Slot)
		day := t.Format("2006-01-02")
		if _, ok := dayBucket[day]; !ok {
			dayBucket[day] = make(map[string]*database.BuilderStatsEntry)
		}
		if _, ok := dayBucket[day][builderID]; !ok {
			bucketStartTime, err := time.Parse("2006-01-02", day)
			check(err)
			bucketEndTime := bucketStartTime.Add(24 * time.Hour)
			dayBucket[day][builderID] = &database.BuilderStatsEntry{
				Hours:       24,
				TimeStart:   bucketStartTime,
				TimeEnd:     bucketEndTime,
				BuilderName: builderID,
			}
		}
		dayBucket[day][builderID].BlocksIncluded += 1
	}

	// sort hourBucket keys alphabetically
	var days []string //nolint:prealloc
	for day := range dayBucket {
		days = append(days, day)
	}
	sort.Strings(days)

	// print
	for _, day := range days {
		builderStats := dayBucket[day]
		log.Infof("day: %s", day)
		entries := make([]*database.BuilderStatsEntry, 0, len(builderStats))
		for builderID, stats := range builderStats {
			if builderStatsVerbose {
				log.Infof("- %34s: %4d blocks", builderID, stats.BlocksIncluded)
			}
			entries = append(entries, stats)
		}
		err := db.SaveBuilderStats(entries)
		check(err)
	}
}
