package website

import (
	"text/template"

	"github.com/flashbots/relayscan/common"
)

type HTMLData struct {
	Title string
	Path  string

	// Root page
	EthMainnetMonths []string

	// File-listing page
	CurrentNetwork string
	CurrentMonth   string
	Files          []FileEntry
}

type FileEntry struct {
	Filename string
	Size     uint64
	Modified string
}

func prettyInt(i uint64) string {
	return printer.Sprintf("%d", i)
}

func caseIt(s string) string {
	return caser.String(s)
}

func percent(cnt, total uint64) string {
	p := float64(cnt) / float64(total) * 100
	return printer.Sprintf("%.2f", p)
}

func substr10(s string) string {
	return s[:10]
}

var DummyHTMLData = &HTMLData{
	Title: "",
	Path:  "",

	EthMainnetMonths: []string{
		"2023-08",
		"2023-09",
	},

	CurrentNetwork: "Ethereum Mainnet",
	CurrentMonth:   "2023-08",
	Files: []FileEntry{
		{"2023-08-29_all.csv.zip", 97210118, "02:02:23 2023-09-02"},
		{"2023-08-29_top.csv.zip", 7210118, "02:02:23 2023-09-02"},

		{"2023-08-30_all.csv.zip", 97210118, "02:02:23 2023-09-02"},
		{"2023-08-30_top.csv.zip", 7210118, "02:02:23 2023-09-02"},

		{"2023-08-31_all.csv.zip", 97210118, "02:02:23 2023-09-02"},
		{"2023-08-31_top.csv.zip", 7210118, "02:02:23 2023-09-02"},
	},
}

var funcMap = template.FuncMap{
	"prettyInt":  prettyInt,
	"caseIt":     caseIt,
	"percent":    percent,
	"humanBytes": common.HumanBytes,
	"substr10":   substr10,
}

func ParseIndexTemplate() (*template.Template, error) {
	return template.New("index.html").Funcs(funcMap).ParseFiles("services/bidcollect/website/templates/index_root.html", "services/bidcollect/website/templates/base.html")
}

func ParseFilesTemplate() (*template.Template, error) {
	return template.New("index.html").Funcs(funcMap).ParseFiles("services/bidcollect/website/templates/index_files.html", "services/bidcollect/website/templates/base.html")
}
