package supply

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

const (
	configFileName         = "sidecars-config.yml"
	pathSidecarsWd         = ".sidecars"
	profiledUserPath       = ".profile.d"
	profiledUserFilename   = "sidecars_buildpack.sh"
	sidecarsConfPathEnvVar = "SIDECARS_BP_CONFIG_PATH"
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
	confPath, err := s.retrieveConfigPath()
	if err != nil {
		return err
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

	s.Log.Info("Adding cloud-sidecars.sh as deps.")
	err = s.Stager.WriteProfileD("cloud-sidecars.sh",
		fmt.Sprintf(
			`export PATH=$PATH:"$HOME/bin":%s`,
			filepath.Join("$DEPS_DIR", s.Stager.DepsIdx(), "bin")),
	)

	if err != nil {
		return err
	}
	s.Log.Info("Finished adding cloud-sidecars.sh as deps.")

	s.Log.Info("Setting %s/%s.", profiledUserPath, profiledUserFilename)
	err = s.writeProfileDUser()
	if err != nil {
		return err
	}
	s.Log.Info("Finished setting %s/%s.", profiledUserPath, profiledUserFilename)
	return nil
}

func (s *Supplier) writeProfileDUser() error {
	idx, err := strconv.ParseInt(s.Stager.DepsIdx(), 10, 64)
	if err != nil {
		return err
	}
	script := ""
	for i := int64(0); i < idx+1; i++ {
		script += fmt.Sprintf(`

if [ -n "$(ls $DEPS_DIR/%d/profile.d/* 2> /dev/null)" ]; then
  for env_file in $DEPS_DIR/%d/profile.d/*; do
    source $env_file
  done
fi

`, i, i)
	}
	os.MkdirAll(filepath.Join(s.Stager.BuildDir(), profiledUserPath), 0755)
	return ioutil.WriteFile(filepath.Join(s.Stager.BuildDir(), profiledUserPath, profiledUserFilename), []byte(script), 0775)
}

func (s *Supplier) retrieveConfigPath() (string, error) {
	buildDir := s.Stager.BuildDir()
	userConfPath := strings.TrimSpace(os.Getenv(sidecarsConfPathEnvVar))
	if userConfPath != "" {
		if strings.HasPrefix(userConfPath, "/") {
			return "", fmt.Errorf("env var %s could not be an absolute path", sidecarsConfPathEnvVar)
		}
		return userConfPath, nil
	}
	os.MkdirAll(filepath.Join(buildDir, pathSidecarsWd), 0755)
	confPath := filepath.Join(buildDir, pathSidecarsWd, configFileName)
	userWdConfPath := filepath.Join(buildDir, configFileName)
	if _, err := os.Stat(userWdConfPath); err == nil {
		s.Log.Info("Move %s to %s ...", userWdConfPath, confPath)
		err := os.Rename(userWdConfPath, confPath)
		if err != nil {
			return "", err
		}
	}
	return confPath, nil
}
