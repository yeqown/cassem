package core

import "encoding/json"

type _action uint8

const (
	logActionSyncCache _action = iota + 1
	logActionSetLeaderAddr
	logActionChangesNotify
)

type coreFSMLog struct {
	Action _action
	Data   []byte
}

func newFsmLog(action _action, cmd command) (*coreFSMLog, error) {
	data, err := cmd.serialize()
	if err != nil {
		return nil, err
	}

	return &coreFSMLog{
		Action: action,
		Data:   data,
	}, nil
}

func (l coreFSMLog) serialize() ([]byte, error)     { return json.Marshal(l) }
func (l *coreFSMLog) deserialize(data []byte) error { return json.Unmarshal(data, l) }

type command interface {
	serialize() ([]byte, error)
	deserialize(data []byte) error
}

type setLeaderAddrCommand struct {
	LeaderAddr string
}

func (cc setLeaderAddrCommand) serialize() ([]byte, error)     { return json.Marshal(cc) }
func (cc *setLeaderAddrCommand) deserialize(data []byte) error { return json.Unmarshal(data, cc) }

type setCacheCommand struct {
	NeedDeleteKey string
	NeedSetKey    string
	NeedSetData   []byte
}

func (cc setCacheCommand) serialize() ([]byte, error)     { return json.Marshal(cc) }
func (cc *setCacheCommand) deserialize(data []byte) error { return json.Unmarshal(data, cc) }

// changesNotifyCommand for changes notify.
// TODO(@yeqown): fill this and test logic.
type changesNotifyCommand struct {
}

func (cc changesNotifyCommand) serialize() ([]byte, error)     { return json.Marshal(cc) }
func (cc *changesNotifyCommand) deserialize(data []byte) error { return json.Unmarshal(data, cc) }
