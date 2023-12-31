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

//go:embed create-tables.sql
var createTablesSql string

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

	db, err := sql.Open("sqlite3", filepath.Join(wd, "builds.sqlite3.db"))
	if err != nil {
		log.Fatalln("Failed to open database:", err)
	}
	_, err = db.Exec(createTablesSql)
	if err != nil {
		log.Fatalln("Failed to initialise database:", err)
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

		exec, err := db.Exec(`INSERT INTO builds (project, meta, filename, sha512) VALUES (?, ?, ?, ?)`, slug, string(buildMetaJson), mpFileHeader.Filename, h512hex)
		if err != nil {
			log.Println("Database Error:", err)
			http.Error(rw, "Database Error", http.StatusInternalServerError)
			return
		}
		autoIncr, err := exec.LastInsertId()
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
			_, err = db.Exec(`UPDATE builds SET mrId = ? WHERE id = ?`, mrId, autoIncr)
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
			_, err = db.Exec(`UPDATE builds SET cfId = ? WHERE id = ?`, cfId, autoIncr)
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
		_ = json.NewEncoder(rw).Encode(projects)
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
		type VersionData struct {
			Meta     json.RawMessage `json:"meta"`
			Filename string          `json:"filename"`
			Sha512   string          `json:"sha512"`
			MrId     *string         `json:"modrinth_id,omitempty"`
			CfId     *string         `json:"curseforge_id,omitempty"`
		}
		versionBlob := make([]VersionData, 0)
		query, err := db.Query(`SELECT meta, filename, sha512, mrId, cfId FROM builds WHERE project = ? ORDER BY id`, slug)
		if err != nil {
			return
		}
		for query.Next() {
			var version VersionData
			var meta string
			err := query.Scan(&meta, &version.Filename, &version.Sha512, &version.MrId, &version.CfId)
			if err != nil {
				log.Println("Database Error:", err)
				http.Error(rw, "Database Error", http.StatusInternalServerError)
				return
			}
			version.Meta = json.RawMessage(meta)
			versionBlob = append(versionBlob, version)
		}
		if query.Err() != nil {
			log.Println("Database Error:", err)
			http.Error(rw, "Database Error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(rw).Encode(versionBlob)
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
