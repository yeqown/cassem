package http

import (
	"fmt"

	coord "github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

type getContainerReq struct {
	Key       string `uri:"key" binding:"required"`
	Namespace string `uri:"ns" binding:"required"`
}

type fieldVO struct {
	Key       string             `json:"key" binding:"required"`
	FieldType datatypes.FieldTyp `json:"fieldType" binding:"required,oneof:1 2 3"`
	// NOTE: Value would be nil while paging container rather than get detail.
	Value interface{} `json:"value,omitempty" binding:"required"`
}

// DONE(@yeqown): fill this part
type containerVO struct {
	Key       string    `json:"key"`
	Namespace string    `json:"namespace"`
	CheckSum  string    `json:"checkSum"`
	Fields    []fieldVO `json:"fields"`
}

func fieldPairs(fld datatypes.IField, showDetail bool) interface{} {
	log.
		WithFields(log.Fields{
			"fld":        fld,
			"showDetail": showDetail,
		}).
		Debug("fieldPairs called")

	if !showDetail {
		// if no need to show detail of container, so there's no need to display any pairs in fields,
		// then just return a nil value which is omitted by fields' json tag.
		return nil
	}

	pairV := fld.Value()
	if pairV == nil {
		log.
			WithFields(log.Fields{
				"fld.NeedSetKey": fld.Name(),
				"fld.Type":       fld.Type(),
			}).
			Warnf("fieldPairs could not handle field with nil pair")

		return nil
	}

	switch fld.Type() {
	case datatypes.KV_FIELD_:
		return toPairVO(pairV.(datatypes.IPair))
	case datatypes.LIST_FIELD_:
		out := make([]*pairVO, 0, 8)
		for _, v := range pairV.([]datatypes.IPair) {
			out = append(out, toPairVO(v))
		}
		return out
	case datatypes.DICT_FIELD_:
		out := make(map[string]*pairVO, 8)
		for k, pair := range pairV.(map[string]datatypes.IPair) {
			out[k] = toPairVO(pair)
		}
		return out
	}

	log.
		WithField("field", fld).
		Warnf("unknown field type %d", fld.Type())

	return nil
}

func toFieldVO(fld datatypes.IField, showDetail bool) fieldVO {
	return fieldVO{
		Key:       fld.Name(),
		FieldType: fld.Type(),
		Value:     fieldPairs(fld, showDetail), // DONE(@yeqown) convert to pairVO
	}
}

// fromFieldVO to construct datatypes.IField should only contains relationship between field and key.
// Such as: kv, list, dict.
func fromFieldVO(ns string, fld fieldVO) (f datatypes.IField, err error) {
	var nonData = datatypes.WithEmpty()

	switch fld.FieldType {
	case datatypes.KV_FIELD_:
		pairKey, ok := fld.Value.(string)
		if !ok {
			err = errors.New("invalid KV_FIELD_ value: " + fld.Key + ", string is expected")
		}
		f = datatypes.NewKVField(fld.Key, datatypes.NewPair(ns, pairKey, nonData))
	case datatypes.LIST_FIELD_:
		v, ok := fld.Value.([]string)
		if !ok {
			err = errors.New("invalid LIST_FIELD_ value: " + fld.Key + ", string list is expected")
			break
		}
		pairs := make([]datatypes.IPair, 0, len(v))
		for _, pairKey := range v {
			pairs = append(pairs, datatypes.NewPair(ns, pairKey, nonData))
		}
		f = datatypes.NewListField(fld.Key, pairs)
	case datatypes.DICT_FIELD_:
		v, ok := fld.Value.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("invalid DICT_FIELD_ value: %s, "+
				"dict[string]string is expected, but got: %T", fld.Key, fld.Value)
			break
		}
		pairs := make(map[string]datatypes.IPair, len(v))
		for dictKey, pairKeyV := range v {
			pairKey, ok := pairKeyV.(string)
			if !ok {
				err = fmt.Errorf("invalid DICT_FIELD_ value: %s.%s, "+
					"string is expected, but got: %T", fld.Key, dictKey, fld.Value)
				goto END
			}

			pairs[dictKey] = datatypes.NewPair(ns, pairKey, nonData)
		}
		f = datatypes.NewDictField(fld.Key, pairs)
	default:
		err = errors.New("unknown field type")
	}

END:
	return
}

func toContainerVO(c datatypes.IContainer, showDetail bool) containerVO {
	f := c.Fields()
	fields := make([]fieldVO, 0, len(f))
	for _, v := range f {
		fields = append(fields, toFieldVO(v, showDetail))
	}

	return containerVO{
		Key:       c.Key(),
		Namespace: c.NS(),
		CheckSum:  c.CheckSum(""),
		Fields:    fields,
	}
}

func (srv *Server) GetContainer(c *gin.Context) {
	req := new(getContainerReq)
	if err := c.ShouldBindUri(req); err != nil {
		responseError(c, err)
		return
	}

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	container, err := srv.coordinator.GetContainer(req.Key, req.Namespace)
	if err != nil {
		responseError(c, err)
		return
	}

	vo := toContainerVO(container, true)
	responseJSON(c, vo)
}

type commonNSAndKeyReq struct {
	Key       string `uri:"key" binding:"required"`
	Namespace string `uri:"ns" binding:"required"`
}

type containerDownloadReq struct {
	commonNSAndKeyReq

	Format   string `form:"format,default=json" binding:"required,oneof=json toml"`
	Filename string `form:"filename"`
}

// ContainerDownload could download container into one file as you want (format, filename).
// DONE(@yeqown): get container to file
func (srv *Server) ContainerDownload(c *gin.Context) {
	req := new(containerDownloadReq)
	if err := c.ShouldBindUri(&(req.commonNSAndKeyReq)); err != nil {
		responseError(c, err)
		return
	}

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	data, err := srv.coordinator.DownloadContainer(req.Key, req.Namespace, req.Format)
	if err != nil {
		responseError(c, err)
		return
	}

	// generate filename
	if req.Filename == "" {
		req.Filename = req.Key + "." + req.Format
	}

	// decide which content type should be passed to responseFile
	var (
		ct contentType
	)
	switch req.Format {
	case "json":
		ct = jsonContentType
	case "toml":
		ct = tomlContentType
	}

	// DONE(@yeqown): response a file
	responseFile(c, req.Filename, ct, data)
}

type pagingContainersReq struct {
	Limit      int    `form:"limit,default=10"`
	Offset     int    `form:"offset,default=0"`
	KeyPattern string `form:"key"`
	Namespace  string `uri:"ns"`
}

func (srv *Server) PagingContainers(c *gin.Context) {
	req := new(pagingContainersReq)
	if err := c.ShouldBindUri(req); err != nil {
		responseError(c, err)
		return
	}

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	out, count, err := srv.coordinator.PagingContainers(&coord.FilterContainersOption{
		Limit:      req.Limit,
		Offset:     req.Offset,
		KeyPattern: req.KeyPattern,
		Namespace:  req.Namespace,
	})
	if err != nil {
		responseError(c, err)
		return
	}

	containers := make([]containerVO, len(out))
	for idx, v := range out {
		containers[idx] = toContainerVO(v, false)
	}

	r := struct {
		Containers []containerVO `json:"containers"`
		Total      int           `json:"total"`
	}{
		Containers: containers,
		Total:      count,
	}

	responseJSON(c, r)
}

type upsertContainerReq struct {
	commonNSAndKeyReq

	Fields []fieldVO `json:"fields" binding:"required"`
}

func (srv *Server) UpsertContainer(c *gin.Context) {
	req := new(upsertContainerReq)
	if err := c.ShouldBindUri(&(req.commonNSAndKeyReq)); err != nil {
		responseError(c, err)
		return
	}

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	// DONE(@yeqown): construct a datatypes.Container from request
	var container = datatypes.NewContainer(req.Namespace, req.Key)
	for _, vo := range req.Fields {
		fld, err := fromFieldVO(req.Namespace, vo)
		if err != nil {
			err = errors.Wrap(err, "at: "+vo.Key)
			responseError(c, err)
			return
		}

		if fld == nil {
			responseError(c, errors.New("server error: could not construct IField at: "+vo.Key))
			return
		}

		evicted, err := container.SetField(fld)
		if err != nil {
			err = errors.Wrap(err, "at: "+vo.Key)
			responseError(c, err)
			return
		}

		if evicted {
			responseError(c, errors.New("duplicated field name in container at: "+vo.Key))
			return
		}
	}

	err := srv.coordinator.SaveContainer(container)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}

type removeContainerReq struct {
	commonNSAndKeyReq
}

func (srv *Server) RemoveContainer(c *gin.Context) {
	req := new(removeContainerReq)
	if err := c.ShouldBindUri(&(req.commonNSAndKeyReq)); err != nil {
		responseError(c, err)
		return
	}

	err := srv.coordinator.RemoveContainer(req.Key, req.Namespace)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}
