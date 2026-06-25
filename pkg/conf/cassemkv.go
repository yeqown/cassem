package conf

// CassemKVConfig contains all config to cassemkv.
type CassemKVConfig struct {
	Bolt          *Bolt  `toml:"bolt"`
	ListenAddr    string `toml:"listenAddr"`
	AdvertiseAddr string `toml:"advertiseAddr"`
	Raft          *Raft  `toml:"raft"`
	HeartbeatTick uint   `toml:"heartbeatTick"`
}
