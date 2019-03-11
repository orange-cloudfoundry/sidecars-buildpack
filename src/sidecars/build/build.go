package build

import (
	"encoding/json"
	"fmt"
	"github.com/buildpack/libbuildpack/build"
	"github.com/buildpack/libbuildpack/layers"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	cloudSidecarsRepo = "orange-cloudfoundry/cloud-sidecars"
	configFileName    = "sidecars-config.yml"
	pathSidecarsWd    = ".sidecars"
	bpIoPathEnvVarKey = "BUILDPACKS_IO_LAUNCHER_PATH"
)

type Builder struct {
	Build build.Build
}

func (b *Builder) Run() error {
	layer := b.Build.Layers.Layer("sidecars")
	layerBin := filepath.Join(layer.Root, "bin")

	b.Build.Logger.Info("Downloading cloud sidecars ...")
	err := downloadCloudSidecar(layerBin)
	if err != nil {
		return err
	}
	b.Build.Logger.Info("Finished downloading cloud sidecars.")

	b.Build.Logger.Info("Running cloud-sidecars setup ...")
	logLevel := "info"
	if os.Getenv("BP_DEBUG") != "" {
		logLevel = "debug"
	}

	appDir := b.Build.Application.Root
	os.MkdirAll(filepath.Join(appDir, pathSidecarsWd), 0755)
	confPath := filepath.Join(appDir, pathSidecarsWd, configFileName)
	userWdConfPath := filepath.Join(appDir, configFileName)
	if _, err := os.Stat(userWdConfPath); err == nil {
		b.Build.Logger.Info("Move %s to %s ...", userWdConfPath, confPath)
		err := os.Rename(userWdConfPath, confPath)
		if err != nil {
			return err
		}
	}

	cmd := exec.Command("cloud-sidecars",
		"--log-level", logLevel,
		"--dir", appDir,
		"--config-path", confPath,
		"--profile-dir", filepath.Join(layer.Root, "profile.d"),
		"setup",
	)
	launcherPath, err := findLifecycleLauncher()
	if err != nil {
		return err
	}
	env := os.Environ()
	env = append(env, fmt.Sprintf("%s='%s'", bpIoPathEnvVarKey, launcherPath))
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	if err != nil {
		return err
	}
	err = cmd.Run()
	if err != nil {
		return err
	}
	err = layer.WriteMetadata(map[string]string{"name": "sidecars"}, layers.Build, layers.Cache, layers.Launch)
	if err != nil {
		return err
	}

	b.Build.Logger.Info("Finished running cloud-sidecars setup.")

	return layer.WriteProfile("cloud-sidecars.sh",
		`export PATH=$PATH:"$HOME/bin":%s`,
		layerBin,
	)
}

func findLifecycleLauncher() (string, error) {
	if _, err := os.Stat(filepath.Join(string(os.PathSeparator), "lifecycle")); err == nil {
		return filepath.Join(string(os.PathSeparator), "lifecycle", "launcher"), nil
	}
	return "", fmt.Errorf("could not found lifecycle launcher")
}

func downloadCloudSidecar(folder string) error {
	binName := "cloud-sidecars_linux_amd64"
	if runtime.GOOS == "windows" {
		binName = "cloud-sidecars_windows_amd64.exe"
	}
	version, err := cloudSidecarsVersion()
	if err != nil {
		return fmt.Errorf("Error when retrieving cloud-sidecars version: %s", err.Error())
	}

	dlUrl := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", cloudSidecarsRepo, version, binName)
	resp, err := http.Get(dlUrl)
	if err != nil {
		return fmt.Errorf("Error when downloading %s version: %s", dlUrl, err.Error())
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(filepath.Join(folder, "cloud-sidecars"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("Error when downloading %s version: %s", dlUrl, err.Error())
	}

	io.Copy(f, resp.Body)
	return nil
}

func cloudSidecarsVersion() (string, error) {
	version, ok := os.LookupEnv("BP_SIDECARS_VERSION")
	if ok {
		return version, nil
	}
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", cloudSidecarsRepo))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tagStruct := struct {
		TagName string `json:"tag_name"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&tagStruct)
	if err != nil {
		return "", err
	}
	return tagStruct.TagName, nil
}
