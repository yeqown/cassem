package core

import (
	"encoding/json"

	"github.com/yeqown/cassem/internal/watcher"
)

type _fsmLogAction uint8

const (
	logActionSyncCache _fsmLogAction = iota + 1
	logActionSetLeaderAddr
	logActionChangesNotify
)

type coreFSMLog struct {
	Action _fsmLogAction
	Data   []byte
	// CreatedAt timestamp records when the log was created at.
	// Oo need to set the value manually, Core.propagateToSlaves will set this.
	//
	// CreatedAt helps fsm to recognize logs those SHOULD better not be executed again.
	// time.Now().Unix() - CreatedAt > 10 * time.Second.
	CreatedAt int64
}

func newFsmLog(action _fsmLogAction, cmd command) (*coreFSMLog, error) {
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

type serializer interface {
	serialize() ([]byte, error)
	deserialize(data []byte) error
}

// TODO(@yeqown): use proto rather than json with benchmark tests.
type command interface {
	serializer
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
type changesNotifyCommand struct {
	watcher.Changes
}

func (cc changesNotifyCommand) serialize() ([]byte, error)     { return json.Marshal(cc) }
func (cc *changesNotifyCommand) deserialize(data []byte) error { return json.Unmarshal(data, cc) }
