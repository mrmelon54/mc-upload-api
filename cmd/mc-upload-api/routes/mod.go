package routes

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/mrmelon54/mc-upload-api/database"
	"log"
	"net/http"
)

func (r routeCtx) modGet(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
	slug := params.ByName("slug")
	project, ok := (*r.projectsYml.Load())[slug]
	if !ok {
		http.Error(rw, "404 Not Found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(rw).Encode(project.ProjectDetails)
}

func (r routeCtx) modVersionsGet(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
	slug := params.ByName("slug")
	_, ok := (*r.projectsYml.Load())[slug]
	if !ok {
		http.Error(rw, "404 Not Found", http.StatusNotFound)
		return
	}
	rows, err := r.db.ListBuilds(req.Context(), slug)
	if err != nil {
		log.Println("Database Error:", err)
		http.Error(rw, "Database Error", http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []database.ListBuildsRow{}
	}
	_ = json.NewEncoder(rw).Encode(rows)
}
