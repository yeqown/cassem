package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/yeqown/log"

	apikv "github.com/yeqown/cassem/api/kv"
	"github.com/yeqown/cassem/pkg/conf"
)

const clusterMemberKeyPrefix = "cassem/cluster/members"

type clusterMemberRecord struct {
	NodeID       uint64 `json:"node_id"`
	RaftAddr     string `json:"raft_addr"`
	GRPCEndpoint string `json:"grpc_endpoint"`
}

func clusterMemberKey(nodeID uint64) string {
	return fmt.Sprintf("%s/%d", clusterMemberKeyPrefix, nodeID)
}

func encodeClusterMemberRecord(record clusterMemberRecord) ([]byte, error) {
	if record.NodeID == 0 {
		return nil, fmt.Errorf("node id is required")
	}
	if strings.TrimSpace(record.RaftAddr) == "" {
		return nil, fmt.Errorf("raft addr is required")
	}
	if strings.TrimSpace(record.GRPCEndpoint) == "" {
		return nil, fmt.Errorf("grpc endpoint is required")
	}

	data, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshal cluster member record: %w", err)
	}
	return data, nil
}

func decodeClusterMemberRecord(data []byte) (clusterMemberRecord, error) {
	var record clusterMemberRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return clusterMemberRecord{}, fmt.Errorf("unmarshal cluster member record: %w", err)
	}
	if record.NodeID == 0 {
		return clusterMemberRecord{}, fmt.Errorf("node id is required")
	}
	if strings.TrimSpace(record.RaftAddr) == "" {
		return clusterMemberRecord{}, fmt.Errorf("raft addr is required")
	}
	if strings.TrimSpace(record.GRPCEndpoint) == "" {
		return clusterMemberRecord{}, fmt.Errorf("grpc endpoint is required")
	}
	return record, nil
}

func advertiseAddr(c *conf.CassemdbConfig) string {
	if c == nil {
		return ""
	}
	return c.AdvertiseAddr
}

func (d *app) registerCurrentMember() error {
	record := clusterMemberRecord{
		NodeID:       d.raft.NodeID(),
		RaftAddr:     d.raft.RaftAddr(),
		GRPCEndpoint: advertiseAddr(d.config),
	}
	return d.upsertClusterMember(context.Background(), record)
}

func (d *app) upsertClusterMember(ctx context.Context, record clusterMemberRecord) error {
	data, err := encodeClusterMemberRecord(record)
	if err != nil {
		return err
	}
	return d.raft.SetKV(ctx, &apikv.SetKVReq{
		Key:       clusterMemberKey(record.NodeID),
		Val:       data,
		Overwrite: true,
	})
}

func (d *app) deleteClusterMember(ctx context.Context, nodeID uint64) error {
	return d.raft.UnsetKV(ctx, &apikv.UnsetKVReq{Key: clusterMemberKey(nodeID)})
}

func (d *app) getClusterMember(nodeID uint64) (clusterMemberRecord, error) {
	entity, err := d.raft.GetKV(&apikv.GetKVReq{Key: clusterMemberKey(nodeID)})
	if err != nil {
		return clusterMemberRecord{}, fmt.Errorf("get cluster member %d: %w", nodeID, err)
	}
	return decodeClusterMemberRecord(entity.GetVal())
}

func (d *app) listMembers() ([]*apikv.ClusterMember, error) {
	resp, err := d.raft.Range(&apikv.RangeReq{Key: clusterMemberKeyPrefix, Limit: 100})
	if err != nil {
		return nil, fmt.Errorf("range cluster members: %w", err)
	}

	liveRaftAddrs := make(map[string]struct{})
	for _, peer := range d.raft.Peers() {
		liveRaftAddrs[peer] = struct{}{}
	}
	leaderID := d.raft.LeaderID()
	members := make([]*apikv.ClusterMember, 0, len(resp.GetEntities()))
	for _, entity := range resp.GetEntities() {
		record, err := decodeClusterMemberRecord(entity.GetVal())
		if err != nil {
			log.WithFields(log.Fields{"key": entity.GetKey(), "error": err}).Warn("skip invalid cluster member record")
			continue
		}
		if len(liveRaftAddrs) > 0 {
			if _, ok := liveRaftAddrs[record.RaftAddr]; !ok {
				log.WithFields(log.Fields{"nodeID": record.NodeID, "raftAddr": record.RaftAddr}).Warn("skip stale cluster member record")
				continue
			}
		}
		members = append(members, &apikv.ClusterMember{
			NodeId:       record.NodeID,
			RaftAddr:     record.RaftAddr,
			GrpcEndpoint: record.GRPCEndpoint,
			Leader:       leaderID != 0 && record.NodeID == leaderID,
		})
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].GetNodeId() < members[j].GetNodeId()
	})
	return members, nil
}
