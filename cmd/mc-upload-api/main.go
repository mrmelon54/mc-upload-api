package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	exitReload "github.com/MrMelon54/exit-reload"
	"github.com/julienschmidt/httprouter"
	"github.com/mrmelon54/mc-upload-api/uploader"
	"gopkg.in/yaml.v3"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

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
		err := os.Mkdir(buildDir, os.ModeDir)
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

	mrUpld := uploader.NewModrinthUploader(configYml.Load().Modrinth, http.DefaultClient)
	cfUpld := uploader.NewCurseforgeUploader(configYml.Load().Curseforge, http.DefaultClient)
	_ = cfUpld

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
		err := mrUpld.UploadVersion("XRsldNHQ", "1.0.0", "alpha", []string{"1.20", "1.20.1"}, []string{"fabric", "forge"}, true, "my-test-file.jar", bytes.NewReader([]byte{0x54, 0x54}))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Error(rw, "OK", http.StatusOK)
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
