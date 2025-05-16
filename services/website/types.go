package website

import "github.com/flashbots/relayscan/database"

type HTTPErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type TopBuilderDisplayEntry struct {
	Info     *database.TopBuilderEntry   `json:"info"`
	Children []*database.TopBuilderEntry `json:"children"`
}
