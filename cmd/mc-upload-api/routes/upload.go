package routes

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/mrmelon54/mc-upload-api/database"
	"github.com/mrmelon54/mc-upload-api/database/types"
	jarparser "github.com/mrmelon54/mc-upload-api/jar-parser"
	resolveversions "github.com/mrmelon54/mc-upload-api/resolve-versions"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const MaxFilesize = 5 << 20 // 5 MiB

func (r routeCtx) uploadPost(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
	slug := params.ByName("slug")
	project, ok := (*r.projectsYml.Load())[slug]
	if !ok {
		http.Error(rw, "404 Not Found", http.StatusNotFound)
		return
	}
	bearer, ok := getBearer(req)
	if !ok {
		http.Error(rw, "403 Forbidden", http.StatusForbidden)
		return
	}
	if bytes.Compare([]byte(project.Token), []byte(bearer)) != 0 {
		http.Error(rw, "403 Forbidden", http.StatusForbidden)
		return
	}
	mpFile, mpFileHeader, err := req.FormFile("upload")
	if err != nil {
		http.Error(rw, "Invalid file", http.StatusInternalServerError)
		return
	}
	if mpFileHeader.Size > MaxFilesize {
		http.Error(rw, "File too big", http.StatusRequestEntityTooLarge)
		return
	}

	fileBuffer := new(bytes.Buffer)
	_, err = io.CopyN(fileBuffer, mpFile, MaxFilesize)
	if err != nil && !errors.Is(err, io.EOF) {
		http.Error(rw, "Failed to transfer file", http.StatusInternalServerError)
		return
	}

	// calculate file hash
	h512 := sha512.New()
	h512.Write(fileBuffer.Bytes())
	h512hex := hex.EncodeToString(h512.Sum(nil))

	modMeta, err := jarparser.JarParser(bytes.NewReader(fileBuffer.Bytes()), int64(fileBuffer.Len()))
	if err != nil {
		log.Println("Failed to parse JAR:", err)
		http.Error(rw, "Failed to parse JAR", http.StatusInternalServerError)
		return
	}

	gameVersions, err := resolveversions.ResolveGameVersions(modMeta.GameVersions, r.mcVersions)
	if err != nil {
		log.Println("Failed to resolve game versions:", err)
		http.Error(rw, "Failed to resolve game versions", http.StatusInternalServerError)
		return
	}

	lastId, err := r.db.CreateBuild(req.Context(), database.CreateBuildParams{
		Project: slug,
		Meta: &types.BuildMeta{
			VersionNumber:  modMeta.VersionNumber,
			ReleaseChannel: modMeta.ReleaseChannel,
			GameVersions:   gameVersions,
			Loaders:        modMeta.Loaders,
			Environment:    modMeta.Environment,
		},
		Filename: mpFileHeader.Filename,
		Sha512:   h512hex,
	})
	if err != nil {
		log.Println("Database Error:", err)
		http.Error(rw, "Database Error", http.StatusInternalServerError)
		return
	}

	datCreate, err := os.Create(filepath.Join(r.buildDir, h512hex+".jar"))
	if err != nil {
		log.Println("Failed file saving:", err)
		http.Error(rw, "Failed file saving", http.StatusInternalServerError)
		return
	}
	_, err = datCreate.Write(fileBuffer.Bytes())
	if err != nil {
		log.Println("Failed file saving:", err)
		http.Error(rw, "Failed file saving", http.StatusInternalServerError)
		return
	}

	if project.Modrinth.Enabled() {
		log.Printf("[Upload] Updating project %s (%s) on Modrinth\n", project.Name, project.Modrinth.Id)
		mrId, err := r.mrUpld.UploadVersion(project.Modrinth.Id, modMeta, gameVersions, mpFileHeader.Filename, bytes.NewReader(fileBuffer.Bytes()))
		if err != nil {
			http.Error(rw, fmt.Errorf("upload modrinth: %w", err).Error(), http.StatusInternalServerError)
			return
		}
		err = r.db.UpdateModrinthFile(req.Context(), database.UpdateModrinthFileParams{
			ModrinthID: mrId,
			ID:         lastId,
		})
		if err != nil {
			log.Println("Database Error:", err)
			http.Error(rw, "Database Error", http.StatusInternalServerError)
			return
		}
	}
	if project.Curseforge.Enabled() {
		log.Printf("[Upload] Updating project %s (%s) on Curseforge\n", project.Name, project.Curseforge.Id)
		cfId, err := r.cfUpld.UploadVersion(project.Curseforge.Id, modMeta, gameVersions, mpFileHeader.Filename, bytes.NewReader(fileBuffer.Bytes()))
		if err != nil {
			http.Error(rw, fmt.Errorf("upload curseforge: %w", err).Error(), http.StatusInternalServerError)
			return
		}
		err = r.db.UpdateCurseforgeFile(req.Context(), database.UpdateCurseforgeFileParams{
			CurseforgeID: cfId,
			ID:           lastId,
		})
		if err != nil {
			log.Println("Database Error:", err)
			http.Error(rw, "Database Error", http.StatusInternalServerError)
			return
		}
	}
	http.Error(rw, "OK", http.StatusOK)
}
