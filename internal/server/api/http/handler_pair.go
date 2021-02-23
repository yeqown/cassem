package http

import (
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

	pairs := make([]pairResponse, len(out))
	for idx, v := range out {
		pairs[idx] = toPairResponse(v)
	}

	r := struct {
		Pairs []pairResponse `json:"pairs"`
		Total int            `json:"total"`
	}{
		Pairs: pairs,
		Total: count,
	}

	responseData(c, r)
}

type getPairReq struct {
	Namespace string `uri:"ns"`
	Key       string `uri:"key"`
}

type pairResponse struct {
	Key      string             `json:"key"`
	Value    interface{}        `json:"value"`
	Datatype datatypes.Datatype `json:"datatype"`
}

func toPairResponse(p datatypes.IPair) pairResponse {
	return pairResponse{
		Key:      p.Key(),
		Value:    p.Value(),
		Datatype: p.Value().Datatype(),
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

	responseData(c, toPairResponse(pair))
}
