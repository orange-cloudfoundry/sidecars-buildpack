package installer

import (
	"fmt"
	"github.com/cloudfoundry/libbuildpack"
	"os"
	"path/filepath"
	"runtime"
)

type Manifest interface {
	// TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/manifest.go
	AllDependencyVersions(string) []string
	DefaultVersion(string) (libbuildpack.Dependency, error)
}

type DepInstaller interface {
	// TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/installer.go
	InstallDependency(libbuildpack.Dependency, string) error
	InstallOnlyVersion(string, string) error
}

type Installer struct {
	depInstaller DepInstaller
	manifest     Manifest
}

func NewInstaller(depInstaller DepInstaller, manifest Manifest) *Installer {
	return &Installer{
		depInstaller: depInstaller,
		manifest:     manifest,
	}
}

func (s *Installer) InstallCloudSidecars(depDir, tempDir string) error {
	gServerName := "cloud-sidecars"
	stack := "cflinuxfs2"
	if runtime.GOOS == "windows" {
		gServerName += ".exe"
		stack = "windows"
	}
	os.Setenv("CF_STACK", stack)
	dep, err := s.manifest.DefaultVersion("cloud-sidecars")
	if err != nil {
		return err
	}

	installDir := filepath.Join(filepath.Join(depDir, "bin", gServerName))

	if err := s.depInstaller.InstallDependency(dep, tempDir); err != nil {
		return err
	}

	binName := "cloud-sidecars_linux_amd64"
	if runtime.GOOS == "windows" {
		binName = "cloud-sidecars_windows_amd64.exe"
	}

	if err := os.Rename(filepath.Join(tempDir, binName), installDir); err != nil {
		return err
	}

	return os.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), filepath.Join(depDir, "bin")))
}
