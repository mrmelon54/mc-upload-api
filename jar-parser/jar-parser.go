package jar_parser

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver/v3"
	"github.com/wreulicke/classfile-parser"
	"io"
	"strings"
)

type ModMetadata struct {
	VersionNumber  string
	ReleaseChannel string
	GameVersions   []*semver.Constraints
	Loaders        []string
	Environment    string
}

func JarParser(r io.ReaderAt, size int64) (ModMetadata, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return ModMetadata{}, err
	}

	meta := ModMetadata{
		ReleaseChannel: "release",
	}

	// try loading fabric
	openFabricModJson, err := zr.Open("fabric.mod.json")
	if err == nil {
		var fabricJson FabricJson
		if err := json.NewDecoder(openFabricModJson).Decode(&fabricJson); err != nil {
			return ModMetadata{}, fmt.Errorf("failed to decode fabric.mod.json: %w", err)
		}
		_ = openFabricModJson.Close()
		for _, entrypoint := range fabricJson.Entrypoints.Main {
			entrypoint = strings.ReplaceAll(entrypoint, ".", "/")
			class, err := loadClass(zr, entrypoint)
			if err != nil {
				return ModMetadata{}, err
			}
			if implementsInterface(class, "net/fabricmc/api/ModInitializer") {
				meta.VersionNumber = fabricJson.Version
				meta.Loaders = append(meta.Loaders, "fabric")
				meta.Environment = fabricJson.Environment
				meta.GameVersions = append(meta.GameVersions, fabricJson.Depends.Minecraft.C)
			}
		}
	}

	// try loading quilt
	openQuiltModJson, err := zr.Open("quilt.mod.json")
	if err == nil {
		var quiltJson QuiltJson
		if err := json.NewDecoder(openQuiltModJson).Decode(&quiltJson); err != nil {
			return ModMetadata{}, err
		}
		_ = openQuiltModJson.Close()
		for _, entrypoint := range quiltJson.QuiltLoader.Entrypoints.Init {
			entrypoint = strings.ReplaceAll(entrypoint, ".", "/")
			class, err := loadClass(zr, entrypoint)
			if err != nil {
				return ModMetadata{}, err
			}
			if implementsInterface(class, "org/quiltmc/qsl/base/api/entrypoint/ModInitializer") {
				meta.VersionNumber = quiltJson.QuiltLoader.Version
				meta.Loaders = append(meta.Loaders, "quilt")
				meta.Environment = quiltJson.Minecraft.Environment
				for _, j := range quiltJson.QuiltLoader.Depends {
					if j.Id == "minecraft" {
						meta.GameVersions = append(meta.GameVersions, j.Version.C)
					}
				}
			}
		}
	}

	// try loading forge/neoforge
	openModsToml, err := zr.Open("META-INF/mods.toml")
	if err == nil {
		var forgeToml ForgeToml
		if _, err := toml.NewDecoder(openModsToml).Decode(&forgeToml); err != nil {
			return ModMetadata{}, err
		}
		_ = openModsToml.Close()

		var entrypointLoader string
	entrypointFinder:
		for _, i := range zr.File {
			if !strings.HasSuffix(i.Name, ".class") {
				continue
			}
			class, err := loadClass(zr, strings.TrimSuffix(i.Name, ".class"))
			if err != nil {
				return ModMetadata{}, err
			}
			visibleAnnotations := class.RuntimeVisibleAnnotations()
			if visibleAnnotations == nil {
				continue
			}
			for _, j := range visibleAnnotations.Annotations {
				annotationClass, _ := j.Type(class.ConstantPool)
				switch annotationClass {
				case "Lnet/minecraftforge/fml/common/Mod;":
					entrypointLoader = "forge"
					break entrypointFinder
				case "Lnet/neoforged/fml/common/Mod;":
					entrypointLoader = "neoforge"
					break entrypointFinder
				}
			}
		}
		if entrypointLoader != "forge" && entrypointLoader != "neoforge" {
			return ModMetadata{}, fmt.Errorf("invalid forge-like loader: %s", entrypointLoader)
		}

		modId := forgeToml.Mods[0].ModID
		meta.VersionNumber = forgeToml.Mods[0].Version
		meta.Loaders = append(meta.Loaders, entrypointLoader)
		for k, v := range forgeToml.Dependencies {
			if k == modId {
				for _, j := range v {
					if j.ModID == "minecraft" {
						versionRange, err := ForgeVersionRange(j.VersionRange)
						if err != nil {
							return ModMetadata{}, fmt.Errorf("failed to parse forge version range '%s': %w\n", j.VersionRange, err)
						}
						meta.GameVersions = append(meta.GameVersions, versionRange)
					}
				}
			}
		}
	}
	return meta, nil
}

func loadClass(zr *zip.Reader, className string) (*parser.Classfile, error) {
	classFile, err := zr.Open(className + ".class")
	if err != nil {
		return nil, fmt.Errorf("failed to open classfile: %w", err)
	}
	defer classFile.Close()
	p := parser.New(classFile)
	class, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse class: %w", err)
	}
	thisClassName, _ := class.ThisClassName()
	if thisClassName != className {
		return nil, fmt.Errorf("class name: %s doesn't match requested: %s", thisClassName, className)
	}
	return class, nil
}

func implementsInterface(class *parser.Classfile, interfaceName string) bool {
	for _, i := range class.Interfaces {
		name, _ := class.ConstantPool.GetClassName(i)
		if name == interfaceName {
			return true
		}
	}
	return false
}
