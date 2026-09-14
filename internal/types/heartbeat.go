package types

type Heartbeat struct {
	InstanceID  string `json:"instance_id"`
	PlayerCount int    `json:"current_players"`
	MaxPlayer   int    `json:"max_players"`
	Status      string `json:"status"`
	MapName     string
}
