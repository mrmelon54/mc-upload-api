package routes

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mrmelon54/mc-upload-api"
	"github.com/mrmelon54/mc-upload-api/database"
	resolveversions "github.com/mrmelon54/mc-upload-api/resolve-versions"
	"github.com/mrmelon54/mc-upload-api/uploader"
	"net/http"
	"strings"
	"sync/atomic"
)

type routeCtx struct {
	db          *database.Queries
	projectsYml *atomic.Pointer[mc_upload_api.ProjectsConfig]
	buildDir    string
	mrUpld      uploader.Uploader
	cfUpld      uploader.Uploader
	mcVersions  *resolveversions.McVersions
}

func Router(db *database.Queries, projectsYml *atomic.Pointer[mc_upload_api.ProjectsConfig], buildDir string, mrUpld uploader.Uploader, cfUpld uploader.Uploader, mcVersions *resolveversions.McVersions) http.Handler {
	base := routeCtx{db, projectsYml, buildDir, mrUpld, cfUpld, mcVersions}

	r := httprouter.New()
	r.POST("/upload/:slug", base.uploadPost)
	r.GET("/summary", base.summaryGet)
	r.GET("/mod/:slug", base.modGet)
	r.GET("/mod/:slug/versions", base.modVersionsGet)
	return r
}

func getBearer(req *http.Request) (string, bool) {
	auth := req.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", false
	}
	return auth[len("Bearer "):], true
}
