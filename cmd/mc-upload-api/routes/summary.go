package routes

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	mc_upload_api "github.com/mrmelon54/mc-upload-api"
	"net/http"
)

func (r routeCtx) summaryGet(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
	projects := *r.projectsYml.Load()
	a := make(map[string]mc_upload_api.ProjectDetails)
	for k, v := range projects {
		a[k] = v.ProjectDetails
	}
	_ = json.NewEncoder(rw).Encode(a)
}
