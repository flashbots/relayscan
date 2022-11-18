package website

import (
	"fmt"
	"sort"
	"strings"

	"github.com/metachris/relayscan/database"
)

type statsResp struct {
	GeneratedAt uint64                      `json:"generated_at"`
	DataStartAt uint64                      `json:"data_start_at"`
	TopRelays   []*database.TopRelayEntry   `json:"top_relays"`
	TopBuilders []*database.TopBuilderEntry `json:"top_builders"`
}

type HTTPErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func consolidateBuilderEntries(builders []*database.TopBuilderEntry) []*database.TopBuilderEntry {
	// Get total builder payloads, and build consolidated builder list
	buildersMap := make(map[string]*database.TopBuilderEntry)
	buildersNumPayloads := uint64(0)
	for _, entry := range builders {
		buildersNumPayloads += entry.NumBlocks
		if strings.Contains(entry.ExtraData, "builder0x69") {
			topBuilderEntry, isKnown := buildersMap["builder0x69"]
			if isKnown {
				topBuilderEntry.NumBlocks += entry.NumBlocks
				topBuilderEntry.Aliases = append(topBuilderEntry.Aliases, entry.ExtraData)
			} else {
				buildersMap["builder0x69"] = &database.TopBuilderEntry{
					ExtraData: "builder0x69",
					NumBlocks: entry.NumBlocks,
					Aliases:   []string{entry.ExtraData},
				}
			}
		} else {
			buildersMap[entry.ExtraData] = entry
		}
	}

	// Prepare top builders by extra stats
	for _, entry := range builders {
		p := float64(entry.NumBlocks) / float64(buildersNumPayloads) * 100
		entry.Percent = fmt.Sprintf("%.2f", p)
	}

	// Prepare top builders by summary stats
	resp := []*database.TopBuilderEntry{}
	for _, entry := range buildersMap {
		p := float64(entry.NumBlocks) / float64(buildersNumPayloads) * 100
		entry.Percent = fmt.Sprintf("%.2f", p)
		resp = append(resp, entry)
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].NumBlocks > resp[j].NumBlocks
	})
	return resp
}

func prepareRelaysEntries(relays []*database.TopRelayEntry) []*database.TopRelayEntry {
	topRelaysNumPayloads := uint64(0)
	for _, entry := range relays {
		topRelaysNumPayloads += entry.NumPayloads
	}
	for i, entry := range relays {
		p := float64(entry.NumPayloads) / float64(topRelaysNumPayloads) * 100
		relays[i].Percent = fmt.Sprintf("%.2f", p)
	}
	return relays
}
