package myraft

import (
	"encoding/json"
)

type _fsmLogAction uint8

const (
	ActionSet _fsmLogAction = iota + 1
	ActionSetLeaderAddr
)

type CoreFSMLog struct {
	Action _fsmLogAction
	Data   []byte
	// CreatedAt timestamp records when the log was created at.
	// Oo need to set the value manually, Core.propagateToSlaves will set this.
	//
	// CreatedAt helps fsm to recognize logs those SHOULD better not be executed again.
	// time.Now().Unix() - CreatedAt > 10 * time.Second.
	CreatedAt int64
}

func NewFsmLog(action _fsmLogAction, cmd command) (*CoreFSMLog, error) {
	data, err := cmd.Serialize()
	if err != nil {
		return nil, err
	}

	return &CoreFSMLog{
		Action: action,
		Data:   data,
	}, nil
}

func (l CoreFSMLog) Serialize() ([]byte, error)     { return json.Marshal(l) }
func (l *CoreFSMLog) Deserialize(data []byte) error { return json.Unmarshal(data, l) }

type Serializer interface {
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
}

// TODO(@yeqown): use proto rather than json with benchmark tests.
type command interface {
	Serializer
}

type SetLeaderAddrCommand struct {
	LeaderAddr string
}

func (cc SetLeaderAddrCommand) Serialize() ([]byte, error)     { return json.Marshal(cc) }
func (cc *SetLeaderAddrCommand) Deserialize(data []byte) error { return json.Unmarshal(data, cc) }

type SetCommand struct {
	DeleteKey   string
	SetKey      string
	NeedSetData []byte
}

func (cc SetCommand) Serialize() ([]byte, error)     { return json.Marshal(cc) }
func (cc *SetCommand) Deserialize(data []byte) error { return json.Unmarshal(data, cc) }

//// ChangesNotifyCommand for changes notify.
//type ChangesNotifyCommand struct {
//	watcher.Changes
//}
//
//func (cc ChangesNotifyCommand) Serialize() ([]byte, error)     { return json.Marshal(cc) }
//func (cc *ChangesNotifyCommand) Deserialize(data []byte) error { return json.Unmarshal(data, cc) }
