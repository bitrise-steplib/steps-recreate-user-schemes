package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
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
		fmt.Println()
		log.Donef("Shared schemes:")
		for _, scheme := range sharedSchemes {
			log.Printf("- %s", scheme.Name)
		}

		os.Exit(0)
	}

	// Generate schemes
	fmt.Println()
	log.Errorf("No shared schemes found...")
	log.Errorf("The newly generated schemes, may differs from the ones in your project.")
	log.Errorf("Make sure to share your schemes, to have the expected behaviour.")

	fmt.Println()
	log.Infof("Generating user schemes")

	if isWorkspace {
		projects, err := xcodeproj.WorkspaceProjectReferences(configs.ProjectPath)
		if err != nil {
			failf("Failed to get workspace referred projects, error: %s", err)
		}

		for _, project := range projects {
			if exist, err := pathutil.IsPathExists(project); err != nil {
				failf("Failed to check if path (%s) exist, error: %s", project, err)
			} else if !exist {
				log.Warnf("skip recreating user schemes for: %s, issue: file not exists", project)
				continue
			}

			log.Printf("recreating user schemes for: %s", project)

			if err := schemes.ReCreateProjectUserSchemes(project); err != nil {
				failf("Failed to recreate user schemes for project (%s), error: %s", project, err)
			}
		}

	} else {
		log.Printf("recreating user schemes for: %s", configs.ProjectPath)
		err = schemes.ReCreateProjectUserSchemes(configs.ProjectPath)
	}

	if err != nil {
		failf("Failed to recreate user schemes, error: %s", err)
	}

	// Ensure user schemes
	fmt.Println()
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

	fmt.Println()
	log.Donef("Generated schemes:")
	for _, scheme := range schemes {
		log.Printf("- %s", scheme.Name)
	}
}
