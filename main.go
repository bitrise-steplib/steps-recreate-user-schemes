package main

import (
	"fmt"
	"os"
	"path/filepath"

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
		return fmt.Errorf("ProjectPath %s", err)
	}

	return nil
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func main() {
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		failf("Issue with input: %s", err)
	}

	isWorkspace := xcodeproj.IsXCWorkspace(configs.ProjectPath)
	if isWorkspace {
		log.Infof("Analyzing workspace: %s", configs.ProjectPath)
	} else {
		log.Infof("Analyzing project: %s", configs.ProjectPath)
	}

	projectOrWorkspaceName := filepath.Base(configs.ProjectPath)

	// Shared schemes
	log.Infof("Searching for shared schemes...")

	sharedSchemes := []xcodeproj.SchemeModel{}

	if isWorkspace {
		workspaceSharedSchemes, err := xcodeproj.WorkspaceSharedSchemes(configs.ProjectPath)
		if err != nil {
			failf("Failed to list workspace (%s) shared schemes, error: %s", configs.ProjectPath, err)
		}

		sharedSchemes = workspaceSharedSchemes
	} else {
		projectSchemesMap, err := xcodeproj.ProjectSharedSchemes(configs.ProjectPath)
		if err != nil {
			failf("Failed to list project (%s) shared schemes, error: %s", configs.ProjectPath, err)
		}

		sharedSchemes = projectSchemesMap
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
		if err := schemes.ReCreateWorkspaceUserSchemes(configs.ProjectPath); err != nil {
			failf("Failed to recreate workspace (%s) user schemes, error: %s", projectOrWorkspaceName, err)
		}
	} else {
		if err := schemes.ReCreateProjectUserSchemes(configs.ProjectPath); err != nil {
			failf("Failed to recreate project (%s) user schemes, error: %s", configs.ProjectPath, err)
		}
	}

	// Ensure user schemes
	log.Infof("Ensure generated schemes")

	schemes := []xcodeproj.SchemeModel{}

	if isWorkspace {
		workspaceSchemes, err := xcodeproj.WorkspaceSharedSchemes(configs.ProjectPath)
		if err != nil {
			failf("Failed to list workspace (%s) shared schemes, error: %s", configs.ProjectPath, err)
		}

		schemes = workspaceSchemes
	} else {
		projectSchemes, err := xcodeproj.ProjectSharedSchemes(configs.ProjectPath)
		if err != nil {
			failf("Failed to list project (%s) shared schemes, error: %s", configs.ProjectPath, err)
		}

		schemes = projectSchemes
	}

	log.Printf("generated scheme count: %d", len(schemes))

	if len(schemes) == 0 {
		failf("No schemes generated")
	}

	fmt.Println("")
	log.Donef("Generated schemes:")
	for _, scheme := range schemes {
		log.Donef("- %s", scheme.Name)
	}
}
