package coord

import (
	"context"
	"errors"
	"fmt"
	apikv "github.com/yeqown/cassem/api/kv"
	"io"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/concept"
)

const (
	_AGENT_PREFIX = "cassem/agents"
)

type agentInsHybrid struct {
	cassemdb apikv.KVClient
}

// func NewAgentInstanceHybrid(endpoints []string) (AgentHybrid, error) {
//	cc, err := apikv.DialWithMode(endpoints, apikv.Mode_X)
//	if err != nil {
//		return nil, errors.Wrap(err, "NewInstanceHybrid")
//	}
//	return &agentInsHybrid{cassemdb: apikv.NewKVClient(cc)}, nil
// }

func (_h agentInsHybrid) Watch(ctx context.Context, ch chan<- *concept.AgentInstanceChange) error {
	stream, err := _h.cassemdb.Watch(ctx, &apikv.WatchReq{
		Keys: []string{_AGENT_PREFIX},
	})
	if err != nil {
		log.
			WithField("error", err).
			Error("cassem.concept.agentInsHybrid failed to watch")
		return err
	}

	change := new(apikv.Change)
	ctx2, cancel := context.WithCancel(stream.Context())
	defer cancel()
loop:
	for {
		select {
		case <-ctx2.Done():
			err = ctx2.Err()
		default:
			if err = stream.RecvMsg(change); err != nil {
				log.
					WithFields(log.Fields{"error": err}).
					Warn("cassem.concept.agentInsHybrid failed to receive message")
				if errors.Is(err, io.EOF) {
					break loop
				}
				continue
			}

			log.
				WithFields(log.Fields{"change": change}).
				Debug("cassem.concept.agentInsHybrid received message")
			c, ok := ConvertChangeToChange(change)
			if !ok {
				continue
			}

			// send to channel
			select {
			case ch <- c:
			default:
				log.
					WithFields(log.Fields{
						"change": change,
						"error":  "agent changes channel is full",
						"len":    len(ch),
						"cap":    cap(ch),
					}).
					Warn("cassem.concept.agentInsHybrid skip push change to channel")
			}
		}
	}

	log.
		WithFields(log.Fields{
			"error": err,
		}).
		Debug("cassem.concept.agentInsHybrid.Watch quit")
	return err
}

func (_h agentInsHybrid) Register(ctx context.Context, ins *concept.AgentInstance, ttl int32) error {
	bytes, err := concept.MarshalProto(ins)
	if err != nil {
		return fmt.Errorf("cassem.concept.agentInsHybrid.Register: %w", err)
	}

	_, err = _h.cassemdb.SetKV(ctx, &apikv.SetKVReq{
		Key:       concept.WithAgentPrefix(ins.AgentId),
		IsDir:     false,
		Ttl:       ttl,
		Val:       bytes,
		Overwrite: false,
	})

	return err
}

func (_h agentInsHybrid) Renew(ctx context.Context, ins *concept.AgentInstance, ttl int32) error {
	bytes, err := concept.MarshalProto(ins)
	if err != nil {
		return fmt.Errorf("cassem.concept.agentInsHybrid.Renew: %w", err)
	}

	_, err = _h.cassemdb.SetKV(ctx, &apikv.SetKVReq{
		Key:       concept.WithAgentPrefix(ins.AgentId),
		IsDir:     false,
		Ttl:       ttl,
		Val:       bytes,
		Overwrite: true,
	})

	return err
}

func (_h agentInsHybrid) Unregister(ctx context.Context, agentId string) error {
	_, err := _h.cassemdb.UnsetKV(ctx, &apikv.UnsetKVReq{
		Key:   concept.WithAgentPrefix(agentId),
		IsDir: false,
	})

	return err
}

func (_h agentInsHybrid) GetAgents(ctx context.Context, seek string, limit int) (*concept.GetAgentsResult, error) {
	r, err := _h.cassemdb.Range(ctx, &apikv.RangeReq{
		Key:   _AGENT_PREFIX,
		Seek:  seek,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("agentInsHybrid.GetAgents: %w", err)
	}

	result := &concept.GetAgentsResult{
		CommonPager: concept.CommonPager{
			HasMore:  r.GetHasMore(),
			NextSeek: r.GetNextSeekKey(),
		},
		Agents: make([]*concept.AgentInstance, 0, len(r.GetEntities())),
	}
	for _, v := range r.GetEntities() {
		agent := new(concept.AgentInstance)
		if err2 := concept.UnmarshalProto(v.GetVal(), agent); err2 != nil {
			log.
				WithFields(log.Fields{
					"error":  err,
					"entity": v,
				}).
				Error("cassem.concept.agentInsHybrid.GetAgents failed unmarshal proto")
			continue
		}

		result.Agents = append(result.Agents, agent)
	}

	return result, nil
}
