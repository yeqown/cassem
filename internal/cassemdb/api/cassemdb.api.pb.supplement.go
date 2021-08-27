package api

import "github.com/yeqown/cassem/pkg/watcher"

var _ watcher.IChange = &Change{}

func (m *Change) Topic() string          { return m.GetKey() }
func (*Change) Type() watcher.ChangeType { return watcher.ChangeType_KV }

func (m *ParentDirectoryChange) Topic() string          { return m.GetSpecificTopic() }
func (*ParentDirectoryChange) Type() watcher.ChangeType { return watcher.ChangeType_DIR }
