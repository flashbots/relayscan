package website

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/flashbots/relayscan/database"
	"github.com/stretchr/testify/require"
)

// TestXSSBuilderExtraData ensures attacker-controlled builder extra_data is
// HTML-escaped (html/template contextual auto-escaping) and cannot break out
// of the HTML/attribute/JS contexts it is rendered into.
func TestXSSBuilderExtraData(t *testing.T) {
	// ParseIndexTemplate uses paths relative to the repo root.
	require.NoError(t, os.Chdir("../.."))
	defer func() { _ = os.Chdir("services/website") }()

	tpl, err := ParseIndexTemplate()
	require.NoError(t, err)

	payload := `</td></tr></table><script>alert('xss')</script>`

	stats := NewStats()
	stats.TopBuilders = []*TopBuilderDisplayEntry{
		{Info: &database.TopBuilderEntry{ExtraData: payload, NumBlocks: 1, Percent: "100.00"}},
	}

	htmlData := &HTMLData{
		Title:          "test",
		TimeSpans:      timespans,
		TimeSpan:       "24h",
		View:           "overview",
		Stats:          stats,
		LastUpdateSlot: 1,
		LastUpdateTime: time.Now().UTC(),
	}

	var buf bytes.Buffer
	require.NoError(t, tpl.ExecuteTemplate(&buf, "base", htmlData))
	out := buf.String()

	// The raw <script> payload must not appear verbatim in the output.
	require.NotContains(t, out, "<script>alert('xss')</script>")
	// It must appear in escaped form instead.
	require.Contains(t, out, "&lt;script&gt;")
	// The onclick attribute must not be broken out of by the payload.
	require.NotContains(t, out, "');\"></td>")
}
