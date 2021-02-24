package http

import (
	"errors"

	coord "github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/gin-gonic/gin"
)

type pagingPairsReq struct {
	Limit      int    `form:"limit,default=10"`
	Offset     int    `form:"offset,default=0"`
	KeyPattern string `form:"key"`
	Namespace  string `uri:"ns"`
}

func (srv *Server) PagingPairs(c *gin.Context) {
	req := new(pagingPairsReq)
	if err := c.ShouldBindUri(req); err != nil {
		responseError(c, err)
		return
	}

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	out, count, err := srv.coordinator.PagingPairs(&coord.FilterPairsOption{
		Limit:      req.Limit,
		Offset:     req.Offset,
		KeyPattern: req.KeyPattern,
		Namespace:  req.Namespace,
	})
	if err != nil {
		responseError(c, err)
		return
	}

	pairs := make([]*pairVO, len(out))
	for idx, v := range out {
		pairs[idx] = toPairVO(v)
	}

	r := struct {
		Pairs []*pairVO `json:"pairs"`
		Total int       `json:"total"`
	}{
		Pairs: pairs,
		Total: count,
	}

	responseJSON(c, r)
}

type getPairReq struct {
	Namespace string `uri:"ns"`
	Key       string `uri:"key"`
}

type pairVO struct {
	Key       string             `json:"key" uri:"key"`
	Value     interface{}        `json:"value"`
	Datatype  datatypes.Datatype `json:"datatype"`
	Namespace string             `json:"namespace"`
}

func toPairVO(p datatypes.IPair) *pairVO {
	if p == nil {
		return nil
	}

	return &pairVO{
		Key:       p.Key(),
		Value:     p.Value(),
		Datatype:  p.Value().Datatype(),
		Namespace: p.NS(),
	}
}

func (srv *Server) GetPair(c *gin.Context) {
	req := new(getPairReq)
	if err := c.ShouldBindUri(req); err != nil {
		responseError(c, err)
		return
	}

	pair, err := srv.coordinator.GetPair(req.Key, req.Namespace)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, toPairVO(pair))
}

type upsertPairReq struct {
	pairVO

	Namespace string `uri:"ns"`
}

func (srv *Server) UpsertPair(c *gin.Context) {
	req := new(upsertPairReq)
	if err := c.ShouldBindUri(req); err != nil {
		responseError(c, err)
		return
	}

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	d := datatypes.ConstructIData(req.Value)
	if d.Datatype() != req.Datatype {
		responseError(c, errors.New("value and datatype unmatch"))
		return
	}

	if d.Data() == nil {
		responseError(c, errors.New("could not parse value to basic datatype"))
		return
	}

	pair := datatypes.NewPair(req.Namespace, req.Key, d)
	err := srv.coordinator.SavePair(pair)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}
