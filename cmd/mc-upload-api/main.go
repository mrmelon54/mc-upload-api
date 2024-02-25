package main

import (
	"bytes"
	"crypto/sha512"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	exitReload "github.com/MrMelon54/exit-reload"
	"github.com/julienschmidt/httprouter"
	_ "github.com/mattn/go-sqlite3"
	mc_upload_api "github.com/mrmelon54/mc-upload-api"
	"github.com/mrmelon54/mc-upload-api/database"
	jar_parser "github.com/mrmelon54/mc-upload-api/jar-parser"
	resolve_versions "github.com/mrmelon54/mc-upload-api/resolve-versions"
	"github.com/mrmelon54/mc-upload-api/uploader"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const MaxFilesize = 5 << 20 // 5 MiB

func main() {
	var configYmlPath string

	flag.StringVar(&configYmlPath, "conf", "", "Path to the config file")
	flag.Parse()

	log.Printf("[Main] Starting up MC Upload API\n")

	wd := filepath.Dir(configYmlPath)
	projectsYmlPath := filepath.Join(wd, "projects.yml")

	var configYml = new(atomic.Pointer[Config])
	var projectsYml = new(atomic.Pointer[ProjectsConfig])

	if err := loadConfig[Config](configYml, configYmlPath); err != nil {
		log.Fatalln("Failed to load config:", err)
	}
	if err := loadConfig[ProjectsConfig](projectsYml, projectsYmlPath); err != nil {
		log.Fatalln("Failed to load projects:", err)
	}

	buildDir := filepath.Join(wd, "builds")
	stat, err := os.Stat(buildDir)
	switch {
	case os.IsNotExist(err):
		err := os.Mkdir(buildDir, 0775)
		if err != nil {
			log.Fatalln("buildDir could not be created")
		}
	case err != nil:
		log.Fatalln("buildDir error:", err)
	default:
		if !stat.IsDir() {
			log.Fatalln("buildDir is not a directory")
		}
	}

	db, err := mc_upload_api.InitDB(filepath.Join(wd, "builds.sqlite3.db"))
	if err != nil {
		log.Fatalln("[DatabaseError] ", err)
	}

	mrUpld := uploader.NewModrinthUploader(configYml.Load().Modrinth, http.DefaultClient)
	cfUpld := uploader.NewCurseforgeUploader(configYml.Load().Curseforge, http.DefaultClient)
	mcVersions := resolve_versions.NewMcVersionCache(http.DefaultClient)

	r := httprouter.New()
	r.POST("/upload/:slug", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		slug := params.ByName("slug")
		project, ok := (*projectsYml.Load())[slug]
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

		modMeta, err := jar_parser.JarParser(bytes.NewReader(fileBuffer.Bytes()), int64(fileBuffer.Len()))
		if err != nil {
			log.Println("Failed to parse JAR:", err)
			http.Error(rw, "Failed to parse JAR", http.StatusInternalServerError)
			return
		}

		gameVersions, err := resolve_versions.ResolveGameVersions(modMeta.GameVersions, mcVersions)
		if err != nil {
			log.Println("Failed to resolve game versions:", err)
			http.Error(rw, "Failed to resolve game versions", http.StatusInternalServerError)
			return
		}

		type BuildMeta struct {
			VersionNumber  string   `json:"version"`
			ReleaseChannel string   `json:"channel"`
			GameVersions   []string `json:"game_versions"`
			Loaders        []string `json:"loaders"`
			Environment    string   `json:"environment"`
		}

		buildMetaJson, err := json.Marshal(BuildMeta{
			VersionNumber:  modMeta.VersionNumber,
			ReleaseChannel: modMeta.ReleaseChannel,
			GameVersions:   gameVersions,
			Loaders:        modMeta.Loaders,
			Environment:    modMeta.Environment,
		})
		if err != nil {
			log.Println("Failed to generate build meta:", err)
			http.Error(rw, "Failed to generate build meta", http.StatusInternalServerError)
			return
		}

		lastId, err := db.CreateBuild(req.Context(), database.CreateBuildParams{
			Project:  slug,
			Meta:     buildMetaJson,
			Filename: mpFileHeader.Filename,
			Sha512:   h512hex,
		})
		if err != nil {
			log.Println("Database Error:", err)
			http.Error(rw, "Database Error", http.StatusInternalServerError)
			return
		}

		datCreate, err := os.Create(filepath.Join(buildDir, h512hex+".jar"))
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
			mrId, err := mrUpld.UploadVersion(project.Modrinth.Id, modMeta, gameVersions, mpFileHeader.Filename, bytes.NewReader(fileBuffer.Bytes()))
			if err != nil {
				http.Error(rw, fmt.Errorf("upload modrinth: %w", err).Error(), http.StatusInternalServerError)
				return
			}
			err = db.UpdateModrinthFile(req.Context(), database.UpdateModrinthFileParams{
				Mrid: sql.NullString{String: mrId, Valid: true},
				ID:   lastId,
			})
			if err != nil {
				log.Println("Database Error:", err)
				http.Error(rw, "Database Error", http.StatusInternalServerError)
				return
			}
		}
		if project.Curseforge.Enabled() {
			log.Printf("[Upload] Updating project %s (%s) on Curseforge\n", project.Name, project.Curseforge.Id)
			cfId, err := cfUpld.UploadVersion(project.Curseforge.Id, modMeta, gameVersions, mpFileHeader.Filename, bytes.NewReader(fileBuffer.Bytes()))
			if err != nil {
				http.Error(rw, fmt.Errorf("upload curseforge: %w", err).Error(), http.StatusInternalServerError)
				return
			}
			err = db.UpdateCurseforgeFile(req.Context(), database.UpdateCurseforgeFileParams{
				Cfid: sql.NullString{String: cfId, Valid: true},
				ID:   lastId,
			})
			if err != nil {
				log.Println("Database Error:", err)
				http.Error(rw, "Database Error", http.StatusInternalServerError)
				return
			}
		}
		http.Error(rw, "OK", http.StatusOK)
	})
	r.GET("/summary", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		projects := *projectsYml.Load()
		a := make(map[string]ProjectDetails)
		for k, v := range projects {
			a[k] = v.ProjectDetails
		}
		_ = json.NewEncoder(rw).Encode(a)
	})
	r.GET("/mod/:slug", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		slug := params.ByName("slug")
		project, ok := (*projectsYml.Load())[slug]
		if !ok {
			http.Error(rw, "404 Not Found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(rw).Encode(project.ProjectDetails)
	})
	r.GET("/mod/:slug/versions", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		slug := params.ByName("slug")
		_, ok := (*projectsYml.Load())[slug]
		if !ok {
			http.Error(rw, "404 Not Found", http.StatusNotFound)
			return
		}
		rows, err := db.ListBuilds(req.Context(), slug)
		if err != nil {
			log.Println("Database Error:", err)
			http.Error(rw, "Database Error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(rw).Encode(rows)
	})
	srv := &http.Server{
		Addr:              configYml.Load().Listen,
		Handler:           r,
		ReadTimeout:       time.Minute,
		ReadHeaderTimeout: time.Minute,
		WriteTimeout:      time.Minute,
		IdleTimeout:       time.Minute,
		MaxHeaderBytes:    5000,
	}
	go func() {
		err = srv.ListenAndServe()
		if err != nil && !errors.Is(http.ErrServerClosed, err) {
			log.Println("Serve HTTP Error:", err)
		}
	}()

	exitReload.ExitReload("MC Upload API", func() {
		if err := loadConfig[Config](configYml, configYmlPath); err != nil {
			log.Println("Failed to load config:", err)
			return
		}
		if err := loadConfig[ProjectsConfig](projectsYml, projectsYmlPath); err != nil {
			log.Println("Failed to load projects:", err)
			return
		}
	}, func() {
		err := srv.Close()
		if err != nil {
			log.Println(err)
		}
	})
}

func loadConfig[T any](ptr *atomic.Pointer[T], p string) error {
	var c T
	file, err := os.Open(p)
	if err != nil {
		return err
	}
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&c)
	if err != nil {
		return err
	}
	ptr.Store(&c)
	return nil
}

func getBearer(req *http.Request) (string, bool) {
	auth := req.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", false
	}
	return auth[len("Bearer "):], true
}
