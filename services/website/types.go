package website

// type statsResp struct {
// 	GeneratedAt uint64                      `json:"generated_at"`
// 	DataStartAt uint64                      `json:"data_start_at"`
// 	TopRelays   []*database.TopRelayEntry   `json:"top_relays"`
// 	TopBuilders []*database.TopBuilderEntry `json:"top_builders"`
// }

type HTTPErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
