package adm

import (
	"net/http"
	"time"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"
)

func (d app) GetAppsHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(pagingAppsReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := d.aggregate.GetApps(r.Context(), req.Seek, req.Limit, req.Query)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, out)
}

func (d app) GetAppHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(getAppReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := d.aggregate.GetApp(r.Context(), req.App)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, out)
}

func (d app) CreateAppHTTP(w http.ResponseWriter, r *http.Request) {
	req := struct {
		createAppUriReq
		createAppReq
	}{}
	if err := bindRequest(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	operator := concept.OperatorFromContext(r.Context())
	md := &concept.AppMetadata{
		Id:          req.App,
		Description: req.Description,
		CreatedAt:   time.Now().Unix(),
		Creator:     operator,
		Owner:       operator,
	}
	if err := d.aggregate.CreateApp(r.Context(), md); err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, nil)
}

func (d app) DeleteAppHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(deleteAppReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := d.aggregate.DeleteApp(r.Context(), req.App); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}

func (d app) GetAppEnvironmentsHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(getAppEnvsReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := d.aggregate.GetEnvironments(r.Context(), req.App, req.Seek, req.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, out)
}

func (d app) CreateAppEnvironmentHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(createAppEnvReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := d.aggregate.CreateEnvironment(r.Context(), req.AppId, req.Env); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}

func (d app) DeleteAppEnvironmentHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(deleteAppEnvReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := d.aggregate.DeleteEnvironment(r.Context(), req.AppId, req.Env); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}

func (d app) GetAppEnvElementsHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(getAppEnvElementsReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	var out any
	var err error
	if len(req.ElementKeys) != 0 {
		out, err = d.aggregate.GetElementsByKeys(r.Context(), req.AppId, req.Env, req.ElementKeys)
	} else {
		out, err = d.aggregate.GetElements(r.Context(), req.AppId, req.Env, req.Seek, req.Limit, req.Query)
	}
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, out)
}

func (d app) GetAppEnvElementHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(getAppEnvElementReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := d.aggregate.GetElementWithVersion(r.Context(), req.AppId, req.Env, req.ElementKey, int(req.Version))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, out)
}

func (d app) CreateAppEnvElementHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(createAppEnvElementReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := d.aggregate.CreateElement(r.Context(),
		req.AppId, req.Env, req.ElementKey, runtime.ToBytes(req.Raw), req.ContentType.concept()); err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, nil)
}

func (d app) UpdateAppEnvElementHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(updateAppEnvElementReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := d.aggregate.UpdateElement(r.Context(), req.AppId, req.Env, req.ElementKey, runtime.ToBytes(req.Raw)); err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, nil)
}

func (d app) DeleteAppEnvElementHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(deleteAppEnvElementsReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := d.aggregate.DeleteElement(r.Context(), req.AppId, req.Env, req.ElementKey); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}

func (d app) GetAppEnvElementAllVersionsHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(getAppEnvElementVersionsReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := d.aggregate.GetElementVersions(r.Context(), req.AppId, req.Env, req.ElementKey, req.Seek, req.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, out)
}

func (d app) GetAppEnvElementOperationsHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(getAppEnvElementOperationsReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := d.aggregate.GetElementOperations(r.Context(), req.AppId, req.Env, req.ElementKey, req.Seek, req.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, out)
}

func (d app) RollbackAppEnvElementHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(rollbackAppEnvElementReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := d.aggregate.RollbackElementVersion(r.Context(), req.AppId, req.Env, req.ElementKey, req.RollbackTo); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}

func (d app) PublishAppEnvElementHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(publishAppEnvElementReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	elem, err := d.aggregate.PublishElementVersion(r.Context(), req.AppId, req.Env, req.ElementKey, req.Publish)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if elem == nil {
		httpx.WriteJSON(w, nil)
		return
	}
	switch req.PublishMode {
	case concept.PublishMode_FULL:
		err = d.ap.notifyAll(elem)
	case concept.PublishMode_GRAY:
		if len(req.AgentIds) == 0 && len(req.InstanceIds) == 0 {
			httpx.WriteError(w, concept.Err_INVALID_ARGUMENT)
			return
		}
		if len(req.AgentIds) == 0 {
			err = d.ap.notifyAgentInstances(elem, req.InstanceIds, d.ap.getAgentIdKeys()...)
		} else {
			err = d.ap.notifyAgentInstances(elem, req.InstanceIds, req.AgentIds...)
		}
	}
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}
