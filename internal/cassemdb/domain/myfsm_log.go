package domain

import (
	"encoding/json"
	"time"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/internal/cassemdb/infras/repository"
)

// action indicates the operation of fsmLog
//go:generate stringer -type=action
type action uint8

const (
	actionSetKV action = iota + 1
	// actionSetLeader
	actionChange

	// _LOG_EXPIRED_TS means 10s
	_LOG_EXPIRED_TS = int64(10)
)

type fsmLog struct {
	// Action indicates which operation is propagated
	Action action

	// Data delivery the raw data that is related to action.
	Data []byte

	// CreatedAt timestamp records when the log was created at.
	// Oo need to set the value manually, Core.propagateToSlaves will set this.
	//
	// CreatedAt helps fsm to recognize logs those SHOULD better not be executed again.
	// time.Now().Unix() - CreatedAt > 10 * time.Second.
	CreatedAt int64
}

func newLog(action action, c command) (*fsmLog, error) {
	data, err := c.Serialize()
	if err != nil {
		return nil, err
	}

	return &fsmLog{
		Action: action,
		Data:   data,
	}, nil
}

func (l fsmLog) Serialize() ([]byte, error)     { return json.Marshal(l) }
func (l *fsmLog) Deserialize(data []byte) error { return json.Unmarshal(data, l) }

// TODO(@yeqown): use proto rather than json with benchmark tests.
type command interface {
	action() action

	Serialize() ([]byte, error)
	Deserialize(data []byte) error
}

type actionApplyFunc func(f *fsm, log *fsmLog) error

type setKVCommand struct {
	DeleteKey repository.StoreKey
	IsDir     bool
	SetKey    repository.StoreKey
	Data      *repository.StoreValue
}

func (cc setKVCommand) action() action                 { return actionSetKV }
func (cc setKVCommand) Serialize() ([]byte, error)     { return json.Marshal(cc) }
func (cc *setKVCommand) Deserialize(data []byte) error { return json.Unmarshal(data, cc) }

func applyActionSetKV(f *fsm, l *fsmLog) (err error) {
	cc := new(setKVCommand)
	if err = cc.Deserialize(l.Data); err != nil {
		panic("could not unmarshal: " + err.Error())
	}

	log.
		WithFields(log.Fields{"command": cc}).
		Debug("applyActionSetKV called")

	if cc.SetKey != "" {
		err = f.repo.SetKV(cc.SetKey, cc.Data, cc.IsDir)
	}
	if cc.DeleteKey != "" {
		err = f.repo.UnsetKV(cc.DeleteKey, cc.IsDir)
	}

	if err != nil {
		log.Error(err)
	}

	return err
}

type changeCommand struct {
	*repository.Change
}

func (cc changeCommand) action() action                 { return actionChange }
func (cc changeCommand) Serialize() ([]byte, error)     { return json.Marshal(cc) }
func (cc *changeCommand) Deserialize(data []byte) error { return json.Unmarshal(data, cc) }

func applyActionChange(f *fsm, l *fsmLog) error {
	if now := time.Now().Unix(); now-l.CreatedAt > _LOG_EXPIRED_TS {
		log.
			WithFields(log.Fields{
				"log":    l,
				"reason": "log.CreatedAt is elder 10 than now",
			}).
			Debug("applyActionChange skip one changeCommand")
		return nil
	}

	cc := new(changeCommand)
	if err := cc.Deserialize(l.Data); err != nil {
		panic("could not unmarshal: " + err.Error())
	}

	log.
		WithFields(log.Fields{
			"command": cc,
		}).
		Debug("applyActionChange called")

	select {
	case f.ch <- cc.Change:
	default:
		log.
			WithFields(log.Fields{
				"reason":       "fsmLog is old",
				"change":       cc.Change,
				"logCreatedAt": l.CreatedAt,
			}).
			Warn("applyActionChange skip one change")
	}

	return nil
}
