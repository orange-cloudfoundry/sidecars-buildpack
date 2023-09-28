package build

import (
	"fmt"
	"github.com/buildpack/libbuildpack/build"
	"github.com/buildpack/libbuildpack/layers"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	configFileName    = "sidecars-config.yml"
	pathSidecarsWd    = ".sidecars"
	bpIoPathEnvVarKey = "BUILDPACKS_IO_LAUNCHER_PATH"
)

type Installer interface {
	InstallCloudSidecars(depDir, tempDir string) error
}

type Builder struct {
	Build     build.Build
	Installer Installer
}

func (b *Builder) Run() error {
	layer := b.Build.Layers.Layer("sidecars")
	layerBin := filepath.Join(layer.Root, "bin")
	if err := os.MkdirAll(layerBin, 0755); err != nil {
		return err
	}

	b.Build.Logger.Info("Installing cloud-sidecars ...")
	err := b.Installer.InstallCloudSidecars(layer.Root, layerBin)
	if err != nil {
		return err
	}
	b.Build.Logger.Info("Finished installing cloud-sidecars ...")

	b.Build.Logger.Info("Running cloud-sidecars setup ...")
	logLevel := "info"
	if os.Getenv("BP_DEBUG") != "" {
		logLevel = "debug"
	}

	appDir := b.Build.Application.Root
	if err := os.MkdirAll(filepath.Join(appDir, pathSidecarsWd), 0755); err != nil {
		return err
	}
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

	return layer.WriteProfile("000_cloud-sidecars.sh",
		`export PATH=$PATH:"$HOME/bin":%s
export %s='%s'
`,
		layerBin,
		bpIoPathEnvVarKey,
		launcherPath,
	)
}

func findLifecycleLauncher() (string, error) {
	if _, err := os.Stat(filepath.Join(string(os.PathSeparator), "lifecycle")); err == nil {
		return filepath.Join(string(os.PathSeparator), "lifecycle", "launcher"), nil
	}
	return "", fmt.Errorf("could not found lifecycle launcher")
}
