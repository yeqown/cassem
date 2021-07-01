package conf

// CassemdbConfig contains all config to cassemdb.
type CassemdbConfig struct {
	Persistence struct {
		BBolt *BBolt `toml:"bbolt"`
	} `toml:"persistence"`

	Server struct {
		HTTP *HTTP `toml:"http"`
		Raft *Raft `toml:"raft"`
	} `toml:"server"`
}
