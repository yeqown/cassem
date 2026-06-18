package conf

// CassemdbConfig contains all config to cassemdb.
type CassemdbConfig struct {
	Bolt          *Bolt  `toml:"bolt"`
	ListenAddr    string `toml:"listenAddr"`
	AdvertiseAddr string `toml:"advertiseAddr"`
	Raft          *Raft  `toml:"raft"`
	HeartbeatTick uint   `toml:"heartbeatTick"`
}
