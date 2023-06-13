package core

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/database"
	"github.com/flashbots/relayscan/vars"
	"github.com/spf13/cobra"
)

var (
	builderStatsDateStart  string
	builderStatsDateEnd    string
	builderStatsSaveDaily  bool
	builderStatsSaveHourly bool
	builderStatsVerbose    bool
	builderStatsBackfill   bool
)

func init() {
	// rootCmd.AddCommand(updateBuilderStatsCmd)
	updateBuilderStatsCmd.Flags().StringVar(&builderStatsDateStart, "start", "", "yyyy-mm-dd hh:mm")
	updateBuilderStatsCmd.Flags().StringVar(&builderStatsDateEnd, "end", "", "yyyy-mm-dd hh:mm")
	updateBuilderStatsCmd.Flags().BoolVar(&builderStatsSaveDaily, "daily", false, "save daily stats")
	updateBuilderStatsCmd.Flags().BoolVar(&builderStatsSaveHourly, "hourly", false, "save hourly stats")
	updateBuilderStatsCmd.Flags().BoolVar(&builderStatsVerbose, "verbose", false, "verbose output")
	updateBuilderStatsCmd.Flags().BoolVar(&builderStatsBackfill, "backfill", false, "backfill hourly stats since last saved")
}

var updateBuilderStatsCmd = &cobra.Command{
	Use:   "update-builder-stats",
	Short: "Update builder stats",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		// args check
		if builderStatsBackfill {
			if builderStatsSaveHourly {
				log.Fatal("--backfill cannot be used with --hourly (for now)")
			}
		} else {
			if builderStatsDateStart == "" {
				log.Fatal("start date is required")
			}
			if builderStatsDateEnd == "" {
				builderStatsDateEnd = time.Now().UTC().Format("2006-01-02")
				log.Infof("Using today as end date: %s", builderStatsDateEnd)
			}
		}
		if !builderStatsSaveDaily && !builderStatsSaveHourly {
			log.Fatal("at least one of --daily or --hourly is required")
		}

		// let's go
		db := database.MustConnectPostgres(log, vars.DefaultPostgresDSN)

		var timeStart, timeEnd time.Time
		var entries []*database.DataAPIPayloadDeliveredEntry
		if builderStatsBackfill {
			// backfill -- between latest daily entry and now
			lastEntry, err := db.GetLastDailyBuilderStatsEntry(database.BuilderStatsEntryTypeExtraData)
			if errors.Is(err, sql.ErrNoRows) {
				log.Fatal("No daily entries found in database. Please run without --backfill first.")
			}
			check(err)
			log.Infof("Last daily entry: %s %s - %s", lastEntry.BuilderName, lastEntry.TimeStart.String(), lastEntry.TimeEnd.String())
			timeStart = lastEntry.TimeStart
			timeEnd = common.BeginningOfDay(time.Now().UTC())
			// return
		} else {
			// query date range
			timeStart = common.MustParseDateTimeStr(builderStatsDateStart)
			timeEnd = common.MustParseDateTimeStr(builderStatsDateEnd)
		}

		log.Infof("Updating builder stats: %s -> %s ", timeStart.String(), timeEnd.String())
		slotStart := common.TimeToSlot(timeStart)
		slotEnd := common.TimeToSlot(timeEnd)

		// make sure slotStart is at the beginning of the day and not before
		timeSlotStart := common.SlotToTime(slotStart)
		if timeSlotStart.Before(timeStart) {
			slotStart++
		}
		timeSlotEnd := common.SlotToTime(slotEnd)
		if timeSlotEnd.After(timeEnd) {
			slotEnd++
		}

		log.Infof("Slots: %d -> %d (%d total)", slotStart, slotEnd, slotEnd-slotStart)

		log.Info("Querying payloads...")
		entries, err = db.GetDeliveredPayloadsForSlots(slotStart, slotEnd)
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
		builderID := vars.BuilderNameFromExtraData(entry.ExtraData)

		t := common.SlotToTime(entry.Slot)
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
		if !strings.Contains(hourBucket[hour][builderID].ExtraData, entry.ExtraData+"\n") {
			hourBucket[hour][builderID].ExtraData += entry.ExtraData + "\n"
		}
		if !strings.Contains(hourBucket[hour][builderID].BuilderPubkeys, entry.BuilderPubkey+"\n") {
			hourBucket[hour][builderID].BuilderPubkeys += entry.BuilderPubkey + "\n"
		}
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
		log.Infof("- updating hour: %s", hour)
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
	bucketByExtraData := make(map[string]map[string]*database.BuilderStatsEntry)
	bucketByPubkey := make(map[string]map[string]*database.BuilderStatsEntry)

	for _, entry := range entriesBySlot {
		builderID := vars.BuilderNameFromExtraData(entry.ExtraData)

		t := common.SlotToTime(entry.Slot)
		day := t.Format("2006-01-02")
		bucketStartTime, err := time.Parse("2006-01-02", day)
		check(err)
		bucketEndTime := bucketStartTime.Add(24 * time.Hour)

		if _, ok := bucketByExtraData[day]; !ok {
			bucketByExtraData[day] = make(map[string]*database.BuilderStatsEntry)
			bucketByPubkey[day] = make(map[string]*database.BuilderStatsEntry)
		}
		if _, ok := bucketByExtraData[day][builderID]; !ok {
			bucketByExtraData[day][builderID] = &database.BuilderStatsEntry{
				Type:        database.BuilderStatsEntryTypeExtraData,
				Hours:       24,
				TimeStart:   bucketStartTime,
				TimeEnd:     bucketEndTime,
				BuilderName: builderID,
			}
		}
		if _, ok := bucketByPubkey[day][entry.BuilderPubkey]; !ok {
			bucketByPubkey[day][entry.BuilderPubkey] = &database.BuilderStatsEntry{
				Type:           database.BuilderStatsEntryTypeBuilderPubkey,
				Hours:          24,
				TimeStart:      bucketStartTime,
				TimeEnd:        bucketEndTime,
				BuilderName:    entry.BuilderPubkey,
				BuilderPubkeys: entry.BuilderPubkey + "\n",
			}
		}

		// Update by extra data
		bucketByExtraData[day][builderID].BlocksIncluded += 1
		if !strings.Contains(bucketByExtraData[day][builderID].ExtraData, entry.ExtraData+"\n") {
			bucketByExtraData[day][builderID].ExtraData += entry.ExtraData + "\n"
		}
		if !strings.Contains(bucketByExtraData[day][builderID].BuilderPubkeys, entry.BuilderPubkey+"\n") {
			bucketByExtraData[day][builderID].BuilderPubkeys += entry.BuilderPubkey + "\n"
		}

		// Update by pubkey
		bucketByPubkey[day][entry.BuilderPubkey].BlocksIncluded += 1
		if !strings.Contains(bucketByPubkey[day][entry.BuilderPubkey].ExtraData, entry.ExtraData+"\n") {
			bucketByPubkey[day][entry.BuilderPubkey].ExtraData += entry.ExtraData + "\n"
		}
	}

	// sort hourBucket keys alphabetically
	var days []string //nolint:prealloc
	for day := range bucketByExtraData {
		days = append(days, day)
	}
	sort.Strings(days)

	// print
	for _, day := range days {
		log.Infof("- updating day: %s", day)
		entries := make([]*database.BuilderStatsEntry, 0)
		for builderID, stats := range bucketByExtraData[day] {
			if builderStatsVerbose {
				log.Infof("- [extra_data] %34s: %4d blocks", builderID, stats.BlocksIncluded)
			}
			entries = append(entries, stats)
		}
		for builderPubkey, stats := range bucketByPubkey[day] {
			if builderStatsVerbose {
				log.Infof("- [pubkey] %34s: %4d blocks", builderPubkey, stats.BlocksIncluded)
			}
			entries = append(entries, stats)
		}
		err := db.SaveBuilderStats(entries)
		check(err)
	}
}
