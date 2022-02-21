package main

import (
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/v2/env"
)

func main() {
	os.Exit(run())
}

func run() int {
	s := createStep()
	cfg, err := s.ProcessConfig()
	if err != nil {
		log.Errorf("Process config failed: %s", err)
		return 1

	}

	if err := s.Run(cfg); err != nil {
		log.Errorf("Run failed: %s", err)
		return 1
	}

	return 0
}

func createStep() SchemeGenerator {
	inputParser := stepconf.NewInputParser(env.NewRepository())
	return NewSchemeGenerator(inputParser)
}
