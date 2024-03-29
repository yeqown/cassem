package concept

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/yeqown/log"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
)

type agentInsHybrid struct {
	cassemdb apicassemdb.KVClient
}

//func NewAgentInstanceHybrid(endpoints []string) (AgentHybrid, error) {
//	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
//	if err != nil {
//		return nil, errors.Wrap(err, "NewInstanceHybrid")
//	}
//	return &agentInsHybrid{cassemdb: apicassemdb.NewKVClient(cc)}, nil
//}

func (_h agentInsHybrid) Watch(ctx context.Context, ch chan<- *AgentInstanceChange) error {
	stream, err := _h.cassemdb.Watch(ctx, &apicassemdb.WatchReq{
		Keys: []string{_AGENT_PREFIX},
	})
	if err != nil {
		log.
			WithField("error", err).
			Error("cassem.concept.agentInsHybrid failed to watch")
		return err
	}

	change := new(apicassemdb.Change)
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
			c, ok := convertChangeToChange(change)
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

func (_h agentInsHybrid) Register(ctx context.Context, ins *AgentInstance, ttl int32) error {
	bytes, err := MarshalProto(ins)
	if err != nil {
		return errors.Wrap(err, "cassem.concept.agentInsHybrid.Register")
	}

	_, err = _h.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
		Key:       withAgentPrefix(ins.AgentId),
		IsDir:     false,
		Ttl:       ttl,
		Val:       bytes,
		Overwrite: false,
	})

	return err
}

func (_h agentInsHybrid) Renew(ctx context.Context, ins *AgentInstance, ttl int32) error {
	bytes, err := MarshalProto(ins)
	if err != nil {
		return errors.Wrap(err, "cassem.concept.agentInsHybrid.Renew")
	}

	_, err = _h.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
		Key:       withAgentPrefix(ins.AgentId),
		IsDir:     false,
		Ttl:       ttl,
		Val:       bytes,
		Overwrite: true,
	})

	return err
}

func (_h agentInsHybrid) Unregister(ctx context.Context, agentId string) error {
	_, err := _h.cassemdb.UnsetKV(ctx, &apicassemdb.UnsetKVReq{
		Key:   withAgentPrefix(agentId),
		IsDir: false,
	})

	return err
}

func (_h agentInsHybrid) GetAgents(ctx context.Context, seek string, limit int) (*getAgentsResult, error) {
	r, err := _h.cassemdb.Range(ctx, &apicassemdb.RangeReq{
		Key:   _AGENT_PREFIX,
		Seek:  seek,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, errors.Wrap(err, "agentInsHybrid.GetAgents")
	}

	result := &getAgentsResult{
		commonPager: commonPager{
			HasMore:  r.GetHasMore(),
			NextSeek: r.GetNextSeekKey(),
		},
		Agents: make([]*AgentInstance, 0, len(r.GetEntities())),
	}
	for _, v := range r.GetEntities() {
		agent := new(AgentInstance)
		if err2 := UnmarshalProto(v.GetVal(), agent); err2 != nil {
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
