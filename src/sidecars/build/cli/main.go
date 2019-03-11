package main

import (
	lbuild "github.com/buildpack/libbuildpack/build"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/orange-cloudfoundry/sidecars-buildpack/src/sidecars/build"
	"github.com/orange-cloudfoundry/sidecars-buildpack/src/sidecars/installer"
	"os"
	"time"
)

func main() {
	log := libbuildpack.NewLogger(os.Stdout)
	b, err := lbuild.DefaultBuild()
	if err != nil {
		log.Info(err.Error())
		os.Exit(9)
	}

	manifest, err := libbuildpack.NewManifest(b.Buildpack.Root, log, time.Now())
	if err != nil {
		log.Error("Unable to load buildpack manifest: %s", err.Error())
		os.Exit(10)
	}
	lInstaller := libbuildpack.NewInstaller(manifest)

	if err = lInstaller.SetAppCacheDir(b.Buildpack.Root); err != nil {
		log.Error("Unable to setup app cache dir: %s", err)
		os.Exit(18)
	}

	builder := build.Builder{
		Build:     b,
		Installer: installer.NewInstaller(lInstaller, manifest),
	}
	err = builder.Run()
	if err != nil {
		log.Error("Error: %s", err)
		os.Exit(15)
	}

	if err = lInstaller.CleanupAppCache(); err != nil {
		log.Error("Unable clean up app cache: %s", err)
		os.Exit(19)
	}
}
