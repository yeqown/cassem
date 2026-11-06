package raftimpl

import (
	"context"

	pb "github.com/yeqown/cassem/internal/protogen"
)

// proposal carries local coordination state alongside a serialized Raft entry.
type proposal struct {
	Ctx   context.Context
	Entry *pb.LogEntry
	ErrC  chan<- error
}

func newProposal(ctx context.Context, entry *pb.LogEntry, errC chan<- error) *proposal {
	if ctx == nil || entry == nil || errC == nil {
		panic("invalid parameters for commit")
	}
	return &proposal{Ctx: ctx, Entry: entry, ErrC: errC}
}
