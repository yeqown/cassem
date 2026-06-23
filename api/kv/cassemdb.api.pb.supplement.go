package kv

import "github.com/yeqown/cassem/api/concept"

var _ concept.Change = &Change{}

func (m *Change) Topic() string          { return m.GetKey() }
func (*Change) Type() concept.ChangeType { return concept.ChangeType_KV }

func (m *ParentDirectoryChange) Topic() string          { return m.GetSpecificTopic() }
func (*ParentDirectoryChange) Type() concept.ChangeType { return concept.ChangeType_DIR }
