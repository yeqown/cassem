package conf

// CassemdbConfig contains all config to cassemdb.
type CassemdbConfig struct {
	Bolt          *Bolt  `toml:"bolt"`
	Addr          string `toml:"addr"`
	Raft          *Raft  `toml:"raft"`
	HeartbeatTick uint   `toml:"heartbeatTick"`
}
