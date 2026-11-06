package concept

import clientpb "github.com/yeqown/cassem/sdk/go/protogen"

// Reuse the public wire types so the server and SDK register each descriptor once.
type (
	Element           = clientpb.Element
	ElementMetadata   = clientpb.ElementMetadata
	ContentType       = clientpb.ContentType
	Instance          = clientpb.Instance
	Instance_Watching = clientpb.Instance_Watching
)

const (
	ContentType_UNKNOWN   = clientpb.ContentType_UNKNOWN
	ContentType_JSON      = clientpb.ContentType_JSON
	ContentType_TOML      = clientpb.ContentType_TOML
	ContentType_INI       = clientpb.ContentType_INI
	ContentType_PLAINTEXT = clientpb.ContentType_PLAINTEXT
)

var (
	ContentType_name  = clientpb.ContentType_name
	ContentType_value = clientpb.ContentType_value
)
