package website

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/database"
	"github.com/sirupsen/logrus"
)

func (srv *Webserver) startRootHTMLUpdateLoops() {
	// kick off latest slot update
	go srv.latestSlotUpdateLoop()

	// kick off 24h update
	go srv.rootDataUpdateLoop(24)

	// kick off 12h update
	go srv.rootDataUpdateLoop(12)

	// kick off 1h update
	go srv.rootDataUpdateLoop(1)

	// kick off 7d update
	if envSkip7dStats {
		srv.log.Info("skipping 7d stats")
	} else {
		go srv.rootDataUpdateLoop(7 * 24)
	}
}

func (srv *Webserver) latestSlotUpdateLoop() {
	for {
		lastPayload, err := srv.db.GetLatestDeliveredPayload()
		if errors.Is(err, sql.ErrNoRows) {
			srv.log.Info("No last delivered payload entry found")
			time.Sleep(1 * time.Minute)
			continue
		} else if err != nil {
			srv.log.WithError(err).Error("Failed to get last delivered payload entry")
			time.Sleep(1 * time.Minute)
			continue
		}

		// Process the latest entry
		srv.latestSlot.Store(lastPayload.Slot)
		srv.log.WithFields(logrus.Fields{
			"slot": lastPayload.Slot,
		}).Infof("Latest database entry found for slot %d", lastPayload.Slot)

		// Wait a bit before checking again
		time.Sleep(1 * time.Minute)
	}
}

func (srv *Webserver) rootDataUpdateLoop(hours int) {
	for {
		startTime := time.Now()
		srv.log.Infof("updating %dh stats...", hours)

		// Get data from database
		stats, err := srv.getStatsForHours(time.Duration(hours) * time.Hour)
		if err != nil {
			srv.log.WithError(err).Errorf("Failed to get stats for %dh", hours)
			continue
		}

		srv.log.WithField("duration", time.Since(startTime).String()).Infof("updated %dh stats", hours)

		// Generate HTML
		overviewBytes, profitBytes, err := srv._renderRootHTML(stats)
		if err != nil {
			srv.log.WithError(err).Error("Failed to render root HTML")
			continue
		}

		// Save the HTML
		htmlKeyOverview := fmt.Sprintf("%s-overview", stats.TimeStr)
		htmlKeyProfit := fmt.Sprintf("%s-builder-profit", stats.TimeStr)

		srv.dataLock.Lock()
		srv.stats[stats.TimeStr] = stats
		srv.html[htmlKeyOverview] = &overviewBytes
		srv.html[htmlKeyProfit] = &profitBytes
		srv.dataLock.Unlock()

		// Wait a bit and then continue
		time.Sleep(1 * time.Minute)
	}
}

func (srv *Webserver) getStatsForHours(duration time.Duration) (stats *Stats, err error) {
	hours := int(duration.Hours())

	timeStr := fmt.Sprintf("%dh", hours)
	if hours == 168 {
		timeStr = "7d"
	}

	until := time.Now().UTC()
	since := until.Add(-1 * duration.Abs())
	log := srv.log.WithFields(logrus.Fields{
		"since": since,
		"until": until,
		"hours": hours,
	})

	log.Debug("- loading top relays...")
	startTime := time.Now()
	topRelays, err := srv.db.GetTopRelays(since, until)
	if err != nil {
		return nil, err
	}
	log.WithField("duration", time.Since(startTime).String()).Debug("- got top relays")

	log.Debug("- loading top builders...")
	startTime = time.Now()
	topBuilders, err := srv.db.GetTopBuilders(since, until, "")
	if err != nil {
		return nil, err
	}
	log.WithField("duration", time.Since(startTime).String()).Debug("- got top builders")

	log.Debug("- loading builder profits...")
	startTime = time.Now()
	builderProfits, err := srv.db.GetBuilderProfits(since, until)
	if err != nil {
		return nil, err
	}
	log.WithField("duration", time.Since(startTime).String()).Debug("- got builder profits")

	stats = &Stats{
		Since:   since,
		Until:   until,
		TimeStr: timeStr,

		TopRelays:          prepareRelaysEntries(topRelays),
		TopBuilders:        consolidateBuilderEntries(topBuilders),
		BuilderProfits:     consolidateBuilderProfitEntries(builderProfits),
		TopBuildersByRelay: make(map[string][]*TopBuilderDisplayEntry),
	}

	// Query builders for each relay
	log.Debug("- loading builders per relay...")
	startTime = time.Now()
	for _, relay := range topRelays {
		topBuildersForRelay, err := srv.db.GetTopBuilders(since, until, relay.Relay)
		if err != nil {
			return nil, err
		}
		stats.TopBuildersByRelay[relay.Relay] = consolidateBuilderEntries(topBuildersForRelay)
	}
	log.WithField("duration", time.Since(startTime).String()).Debug("- got builders per relay")
	return stats, nil
}

func (srv *Webserver) _renderRootHTML(stats *Stats) (overviewBytes, profitBytes []byte, err error) {
	latestSlotInDB := srv.latestSlot.Load()
	latestSlotInDBTime := common.SlotToTime(latestSlotInDB)

	// Render the HTML for overview
	htmlBuf := bytes.Buffer{}
	htmlData := &HTMLData{
		Title:             "MEV-Boost Relay & Builder Stats",
		View:              "overview",
		TimeSpans:         timespans,
		TimeSpan:          stats.TimeStr,
		Stats:             stats,
		LastUpdateSlot:    latestSlotInDB,
		LastUpdateTime:    latestSlotInDBTime,
		LastUpdateTimeStr: latestSlotInDBTime.Format("2006-01-02 15:04"),
	}

	// Render the template & minify
	if err := srv.templateIndex.ExecuteTemplate(&htmlBuf, "base", htmlData); err != nil {
		srv.log.WithError(err).Error("error rendering template")
		return nil, nil, err
	}
	overviewBytes, err = srv.minifier.Bytes("text/html", htmlBuf.Bytes())
	if err != nil {
		srv.log.WithError(err).Error("error minifying html")
		return nil, nil, err
	}

	// Render HTML for builder profit
	htmlBuf = bytes.Buffer{}
	htmlData.Title = "MEV-Boost Builder Profitability"
	htmlData.View = "builder-profit"
	if err := srv.templateIndex.ExecuteTemplate(&htmlBuf, "base", htmlData); err != nil {
		srv.log.WithError(err).Error("error rendering template")
		return nil, nil, err
	}
	profitBytes, err = srv.minifier.Bytes("text/html", htmlBuf.Bytes())
	if err != nil {
		srv.log.WithError(err).Error("error minifying html")
		return nil, nil, err
	}

	return overviewBytes, profitBytes, nil
}

func (srv *Webserver) _getDailyStats(t time.Time) (since, until, minDate time.Time, relays []*database.TopRelayEntry, builders []*database.TopBuilderEntry, builderProfits []*database.BuilderProfitEntry, err error) {
	now := time.Now().UTC()
	minDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour).UTC()
	if t.UTC().After(minDate.UTC()) {
		return now, now, minDate, nil, nil, nil, fmt.Errorf("date is too recent") //nolint:goerr113
	}

	since = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	until = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
	relays, builders, builderProfits, err = srv.db.GetStatsForTimerange(since, until, "")
	return since, until, minDate, relays, builders, builderProfits, err
}
