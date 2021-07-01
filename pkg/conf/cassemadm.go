package conf

// CassemAdminConfig contains all config to cassemadm.
type CassemAdminConfig struct {
	// CassemDBCluster in format like: cassemdb://172.16.2.25:2021,172.16.2.26:2021,172.16.2.27:2021
	CassemDBCluster string `toml:"cassemdb_cluster"`

	// HTTP indicates which port is cassemadm serving on.
	HTTP *HTTP `toml:"http"`
}
