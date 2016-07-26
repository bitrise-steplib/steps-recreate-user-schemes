package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/bitrise-io/generate-shared-scheme/logger"
	"github.com/bitrise-io/generate-shared-scheme/xcodeproject"
)

const (
	schemeFileExtension = ".xcscheme"
)

// -----------------------
// --- Functions
// -----------------------

func isWorkspace(pth string) bool {
	return strings.HasSuffix(pth, ".xcworkspace")
}

func validateRequiredInput(key, value string) {
	if value == "" {
		log.Fail("Missing required input: %s", key)
	}
}

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	envman := exec.Command("envman", "add", "--key", keyStr)
	envman.Stdin = strings.NewReader(valueStr)
	envman.Stdout = os.Stdout
	envman.Stderr = os.Stderr
	return envman.Run()
}

// -----------------------
// --- Main
// -----------------------

func main() {
	// Validate options
	projectOrWorkspacePth := os.Getenv("project_path")

	log.Configs(projectOrWorkspacePth)

	validateRequiredInput("project_path", projectOrWorkspacePth)

	isWorkspace := isWorkspace(projectOrWorkspacePth)
	if isWorkspace {
		log.Info("Analyzing workspace: %s", projectOrWorkspacePth)
	} else {
		log.Info("Analyzing project: %s", projectOrWorkspacePth)
	}

	// Shared schemes
	sharedSchemes := []string{}

	if isWorkspace {
		log.Info("Searching for workspace shared schemes...")

		workspaceSharedSchemes, err := xcodeproject.SharedSchemes(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace (%s) shared schemes, error: %s", projectOrWorkspacePth, err)
		}

		log.Details("workspace shared schemes: %v", workspaceSharedSchemes)

		projects, err := xcodeproject.WorkspaceProjectReferences(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace referred projects, error: %s", err)
		}

		for _, project := range projects {
			workspaceProjectSharedSchemes, err := xcodeproject.SharedSchemes(project)
			if err != nil {
				log.Fail("Failed to list project (%s) shared schemes, error: %s", project, err)
			}

			log.Details("workspace project shared schemes: %v", workspaceProjectSharedSchemes)

			workspaceSharedSchemes = append(workspaceSharedSchemes, workspaceProjectSharedSchemes...)
		}

		sharedSchemes = workspaceSharedSchemes
	} else {
		log.Info("Searching for project shared schemes...")

		projectSchemes, err := xcodeproject.SharedSchemes(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list project (%s) shared schemes, error: %s", projectOrWorkspacePth, err)
		}

		log.Details("project shared schemes: %v", projectSchemes)

		sharedSchemes = projectSchemes
	}

	if len(sharedSchemes) > 0 {
		log.Done("Shared schemes: %v", sharedSchemes)
		os.Exit(0)
	}

	// Generate schemes
	if isWorkspace {
		log.Info("No shared scheme found for workspace, generating default schemes...")

		projects, err := xcodeproject.WorkspaceProjectReferences(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace referred projects, error: %s", err)
		}

		for _, project := range projects {
			log.Details("generating default schemes for: %s", project)

			if err := xcodeproject.ReCreateProjectUserSchemes(project); err != nil {
				log.Fail("Failed to recreate project (%s) user schemes, error: %s", err)
			}
		}
	} else {
		log.Info("No shared scheme found for project, generating default schemes...")
		log.Details("generating default schemes for: %s", projectOrWorkspacePth)

		if err := xcodeproject.ReCreateProjectUserSchemes(projectOrWorkspacePth); err != nil {
			log.Fail("Failed to recreate project (%s) user schemes, error: %s", err)
		}
	}

	/*
		// Share user schemes
		if isWorkspace {
			projects, err := xcodeproject.WorkspaceProjectReferences(projectOrWorkspacePth)
			if err != nil {
				log.Fail("Failed to list workspace referred projects, error: %s", err)
			}

			for _, project := range projects {
				workspaceProjectUserSchemes, err := xcodeproject.UserSchemes(project)
				if err != nil {
					log.Fail("Failed to list project (%s) user schemes, error: %s", project, err)
				}

				for _, scheme := range workspaceProjectUserSchemes {
					if err := xcodeproject.ShareUserScheme(project, scheme); err != nil {
						log.Fail("Failed to recreate project (%s) user schemes (%s), error: %s", project, scheme, err)
					}
				}
			}
		} else {
			if err := xcodeproject.ReCreateProjectUserSchemes(projectOrWorkspacePth); err != nil {
				log.Fail("Failed to recreate project (%s) user schemes, error: %s", err)
			}

			projectUserSchemes, err := xcodeproject.UserSchemes(projectOrWorkspacePth)
			if err != nil {
				log.Fail("Failed to list project (%s) user schemes, error: %s", projectOrWorkspacePth, err)
			}

			for _, scheme := range projectUserSchemes {
				if err := xcodeproject.ShareUserScheme(projectOrWorkspacePth, scheme); err != nil {
					log.Fail("Failed to recreate project (%s) user schemes (%s), error: %s", projectOrWorkspacePth, scheme, err)
				}
			}
		}
	*/

	// Ensure user schemes
	userSchemes := []string{}

	if isWorkspace {
		log.Info("Ensure workspace generated user schemes")

		workspaceUserSchemes, err := xcodeproject.UserSchemes(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace (%s) shared schemes, error: %s", projectOrWorkspacePth, err)
		}

		log.Details("workspace user schemes: %v", workspaceUserSchemes)

		projects, err := xcodeproject.WorkspaceProjectReferences(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace referred projects, error: %s", err)
		}

		for _, project := range projects {
			workspaceProjectUserSchemes, err := xcodeproject.UserSchemes(project)
			if err != nil {
				log.Fail("Failed to list project (%s) shared schemes, error: %s", project, err)
			}

			log.Details("workspace project user schemes: %v", workspaceProjectUserSchemes)

			workspaceUserSchemes = append(workspaceUserSchemes, workspaceProjectUserSchemes...)
		}

		userSchemes = workspaceUserSchemes
	} else {
		log.Info("Ensure project generated user schemes")

		projectSchemes, err := xcodeproject.UserSchemes(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list project (%s) shared schemes, error: %s", projectOrWorkspacePth, err)
		}

		log.Details("project user schemes: %v", projectSchemes)

		userSchemes = projectSchemes
	}

	if len(userSchemes) == 0 {
		log.Fail("No user schemes generated")
	}

	fmt.Println("")
	log.Done("Generated user schemes: %v", userSchemes)

	fmt.Println("")
	log.Done("Done 🚀")
}
