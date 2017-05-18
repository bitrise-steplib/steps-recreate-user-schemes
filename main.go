package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-recreate-user-schemes/schemes"
	"github.com/bitrise-tools/go-steputils/input"
	"github.com/bitrise-tools/go-xcode/xcodeproj"
)

// ConfigsModel ...
type ConfigsModel struct {
	ProjectPath string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		ProjectPath: os.Getenv("project_path"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")
	log.Printf("- ProjectPath: %s", configs.ProjectPath)
}

func (configs ConfigsModel) validate() error {
	if err := input.ValidateIfPathExists(configs.ProjectPath); err != nil {
		return fmt.Errorf("ProjectPath - %s", err)
	}

	return nil
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func main() {
	configs := createConfigsModelFromEnvs()
	configs.print()

	fmt.Println()

	if err := configs.validate(); err != nil {
		failf("Issue with input: %s", err)
	}

	// Shared schemes
	isWorkspace := xcodeproj.IsXCWorkspace(configs.ProjectPath)
	sharedSchemes := []xcodeproj.SchemeModel{}
	var err error

	if isWorkspace {
		log.Infof("Searching for workspace shared schemes...")
		sharedSchemes, err = xcodeproj.WorkspaceSharedSchemes(configs.ProjectPath)
	} else {
		log.Infof("Searching for project shared schemes...")
		sharedSchemes, err = xcodeproj.ProjectSharedSchemes(configs.ProjectPath)
	}

	if err != nil {
		failf("Failed to list shared schemes, error: %s", err)
	}

	log.Printf("shared scheme count: %d", len(sharedSchemes))

	if len(sharedSchemes) > 0 {
		log.Donef("Shared schemes:")
		for _, scheme := range sharedSchemes {
			log.Donef("- %s", scheme.Name)
		}

		os.Exit(0)
	}

	// Generate schemes
	fmt.Println("")
	log.Errorf("No shared schemes found, generating default user schemes...")
	log.Errorf("The newly generated schemes, may differs from the ones in your project.")
	log.Errorf("Make sure to share your schemes, to have the expected behaviour.")
	fmt.Println("")

	if isWorkspace {
		err = schemes.ReCreateWorkspaceUserSchemes(configs.ProjectPath)
	} else {
		err = schemes.ReCreateProjectUserSchemes(configs.ProjectPath)
	}

	if err != nil {
		failf("Failed to recreate user schemes, error: %s", err)
	}

	// Ensure user schemes
	log.Infof("Ensure generated schemes")

	schemes := []xcodeproj.SchemeModel{}

	if isWorkspace {
		schemes, err = xcodeproj.WorkspaceSharedSchemes(configs.ProjectPath)
	} else {
		schemes, err = xcodeproj.ProjectSharedSchemes(configs.ProjectPath)
	}

	if err != nil {
		failf("Failed to list shared schemes, error: %s", err)
	}

	log.Printf("generated scheme count: %d", len(schemes))

	if len(schemes) == 0 {
		failf("No schemes generated")
	}

	fmt.Println("")
	log.Donef("Generated schemes:")
	for _, scheme := range schemes {
		log.Infof("- %s", scheme.Name)
	}
}
