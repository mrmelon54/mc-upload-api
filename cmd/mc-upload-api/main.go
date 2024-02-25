package main

import (
	"errors"
	"flag"
	exitReload "github.com/MrMelon54/exit-reload"
	mcuploadapi "github.com/mrmelon54/mc-upload-api"
	"github.com/mrmelon54/mc-upload-api/cmd/mc-upload-api/routes"
	resolveversions "github.com/mrmelon54/mc-upload-api/resolve-versions"
	"github.com/mrmelon54/mc-upload-api/uploader"
	"gopkg.in/yaml.v3"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	var configYml = new(atomic.Pointer[mcuploadapi.Config])
	var projectsYml = new(atomic.Pointer[mcuploadapi.ProjectsConfig])

	if err := loadConfig[mcuploadapi.Config](configYml, configYmlPath); err != nil {
		log.Fatalln("Failed to load config:", err)
	}
	if err := loadConfig[mcuploadapi.ProjectsConfig](projectsYml, projectsYmlPath); err != nil {
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

	db, err := mcuploadapi.InitDB(filepath.Join(wd, "builds.sqlite3.db"))
	if err != nil {
		log.Fatalln("[DatabaseError] ", err)
	}

	mrUpld := uploader.NewModrinthUploader(configYml.Load().Modrinth, http.DefaultClient)
	cfUpld := uploader.NewCurseforgeUploader(configYml.Load().Curseforge, http.DefaultClient)
	mcVersions := resolveversions.NewMcVersionCache(http.DefaultClient)

	srv := &http.Server{
		Addr:              configYml.Load().Listen,
		Handler:           routes.Router(db, projectsYml, buildDir, mrUpld, cfUpld, mcVersions),
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
		if err := loadConfig[mcuploadapi.Config](configYml, configYmlPath); err != nil {
			log.Println("Failed to load config:", err)
			return
		}
		if err := loadConfig[mcuploadapi.ProjectsConfig](projectsYml, projectsYmlPath); err != nil {
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
