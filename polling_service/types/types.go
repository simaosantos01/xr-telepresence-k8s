package types

type PollingData struct {
	Clients      int16  `json:"clients"`
	LatestChange string `json:"latestChange"`
}