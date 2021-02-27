package core

import "encoding/json"

type _action uint8

const (
	logActionSyncCache _action = iota + 1
	logActionSetLeaderAddr
)

type coreFSMLog struct {
	Action _action
	Data   []byte
}

func newFsmLog(action _action, v serializer) (data []byte, err error) {
	if data, err = v.serialize(); err != nil {
		return
	}

	return coreFSMLog{
		Action: action,
		Data:   data,
	}.serialize()
}

func (l coreFSMLog) serialize() ([]byte, error) {
	return json.Marshal(l)
}

func (l *coreFSMLog) deserialize(data []byte) error {
	return json.Unmarshal(data, l)
}

type serializer interface {
	serialize() ([]byte, error)
}

type setLeaderAddr struct {
	LeaderAddr string
}

func (cla setLeaderAddr) serialize() ([]byte, error) {
	return json.Marshal(cla)
}

func (cla *setLeaderAddr) deserialize(data []byte) error {
	return json.Unmarshal(data, cla)
}

type coreSetCache struct {
	NeedDeleteKey string
	NeedSetKey    string
	NeedSetData   []byte
}

func (cc coreSetCache) serialize() ([]byte, error) {
	return json.Marshal(cc)
}

func (cc *coreSetCache) deserialize(data []byte) error {
	return json.Unmarshal(data, cc)
}
