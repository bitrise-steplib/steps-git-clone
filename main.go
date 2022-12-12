package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-steplib/steps-git-clone/gitclone"
)

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func main() {
	var cfg gitclone.Config
	envRepo := env.NewRepository()
	if err := stepconf.NewInputParser(envRepo).Parse(&cfg); err != nil {
		failf("Error: %s\n", err)
	}
	stepconf.Print(cfg)

	if err := gitclone.Execute(cfg); err != nil {
		failf("Error: %v", err)
	}

	fmt.Println()
	log.Donef("Success")
}
