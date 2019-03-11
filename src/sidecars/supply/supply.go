package supply

import (
	"fmt"
	"github.com/cloudfoundry/libbuildpack"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	configFileName = "sidecars-config.yml"
	pathSidecarsWd = ".sidecars"
)

type Stager interface {
	// TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/stager.go
	BuildDir() string
	DepDir() string
	DepsIdx() string
	DepsDir() string
	WriteProfileD(scriptName, scriptContents string) error
}

type Installer interface {
	InstallCloudSidecars(depDir, tempDir string) error
}

type Command interface {
	// TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/command.go
	Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(dir string, program string, args ...string) (string, error)
}

type Supplier struct {
	Stager    Stager
	Command   Command
	Log       *libbuildpack.Logger
	Installer Installer
}

func (s *Supplier) Run() error {
	s.Log.BeginStep("Staging cloud-sidecars")
	s.Log.Info("Installing cloud-sidecars ...")
	err := s.Installer.InstallCloudSidecars(s.Stager.DepDir(), "/tmp/cloud-sidecars")
	if err != nil {
		return err
	}
	s.Log.Info("Finished installing cloud-sidecars ...")

	s.Log.Info("Running cloud-sidecars setup ...")
	logLevel := "info"
	if os.Getenv("BP_DEBUG") != "" {
		logLevel = "debug"
	}
	buildDir := s.Stager.BuildDir()
	os.MkdirAll(filepath.Join(buildDir, pathSidecarsWd), 0755)
	confPath := filepath.Join(buildDir, pathSidecarsWd, configFileName)
	userWdConfPath := filepath.Join(buildDir, configFileName)
	if _, err := os.Stat(userWdConfPath); err == nil {
		s.Log.Info("Move %s to %s ...", userWdConfPath, confPath)
		err := os.Rename(userWdConfPath, confPath)
		if err != nil {
			return err
		}
	}
	cmd := exec.Command("cloud-sidecars",
		"--log-level", logLevel,
		"--dir", buildDir,
		"--config-path", confPath,
		"--profile-dir", filepath.Join(s.Stager.DepDir(), "profile.d"),
		"setup",
	)
	cmd.Env = os.Environ()
	cmd.Stdout = s.Log.Output()
	cmd.Stderr = s.Log.Output()
	if err != nil {
		return err
	}
	err = cmd.Run()
	if err != nil {
		return err
	}

	s.Log.Info("Finished running cloud-sidecars setup.")

	return s.Stager.WriteProfileD("cloud-sidecars.sh",
		fmt.Sprintf(
			`export PATH=$PATH:"$HOME/bin":%s`,
			filepath.Join("$DEPS_DIR", s.Stager.DepsIdx(), "bin")),
	)
}
