package main

import (
	lbuild "github.com/buildpack/libbuildpack/build"
	"github.com/buildpack/libbuildpack/logger"
	"github.com/orange-cloudfoundry/sidecars-buildpack/src/sidecars/build"
	"os"
)

func main() {
	log := logger.NewLogger(os.Stdout, os.Stdout)
	b, err := lbuild.DefaultBuild()
	if err != nil {
		log.Info(err.Error())
		os.Exit(9)
	}
	builder := build.Builder{
		Build: b,
	}

	err = builder.Run()
	if err != nil {
		log.Info("Error: %s", err)
		os.Exit(15)
	}
}
