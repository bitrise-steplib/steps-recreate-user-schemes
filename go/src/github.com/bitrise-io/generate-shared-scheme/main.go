package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		workspaceName := filepath.Base(projectOrWorkspacePth)

		log.Info("Searching for workspace (%s) shared schemes...", workspaceName)

		workspaceSharedSchemes, err := xcodeproject.SharedSchemes(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace (%s) shared schemes, error: %s", projectOrWorkspacePth, err)
		}

		log.Details("workspace (%s) shared schemes: %v", workspaceName, workspaceSharedSchemes)

		projects, err := xcodeproject.WorkspaceProjectReferences(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace referred projects, error: %s", err)
		}

		for _, project := range projects {
			projectName := filepath.Base(project)

			workspaceProjectSharedSchemes, err := xcodeproject.SharedSchemes(project)
			if err != nil {
				log.Fail("Failed to list project (%s) shared schemes, error: %s", project, err)
			}

			log.Details("workspace project (%s) shared schemes: %v", projectName, workspaceProjectSharedSchemes)

			workspaceSharedSchemes = append(workspaceSharedSchemes, workspaceProjectSharedSchemes...)
		}

		sharedSchemes = workspaceSharedSchemes
	} else {
		projectName := filepath.Base(projectOrWorkspacePth)

		log.Info("Searching for project (%s) shared schemes...", projectName)

		projectSchemes, err := xcodeproject.SharedSchemes(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list project (%s) shared schemes, error: %s", projectOrWorkspacePth, err)
		}

		log.Details("project (%s) shared schemes: %v", projectName, projectSchemes)

		sharedSchemes = projectSchemes
	}

	if len(sharedSchemes) > 0 {
		log.Done("Shared schemes: %v", sharedSchemes)
		os.Exit(0)
	}

	// Generate schemes
	if isWorkspace {
		workspaceName := filepath.Base(projectOrWorkspacePth)

		log.Info("No shared scheme found for workspace (%s), generating default schemes...", workspaceName)

		projects, err := xcodeproject.WorkspaceProjectReferences(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace referred projects, error: %s", err)
		}

		for _, project := range projects {
			projectName := filepath.Base(project)

			log.Details("generating default schemes for: %s", projectName)

			if err := xcodeproject.ReCreateProjectUserSchemes(project); err != nil {
				log.Fail("Failed to recreate project (%s) user schemes, error: %s", project, err)
			}
		}
	} else {
		projectName := filepath.Base(projectOrWorkspacePth)

		log.Info("No shared scheme found for project (%s), generating default schemes...", projectName)
		log.Details("generating default schemes for: %s", projectName)

		if err := xcodeproject.ReCreateProjectUserSchemes(projectOrWorkspacePth); err != nil {
			log.Fail("Failed to recreate project (%s) user schemes, error: %s", projectOrWorkspacePth, err)
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

		workspaceName := filepath.Base(projectOrWorkspacePth)

		workspaceUserSchemes, err := xcodeproject.UserSchemes(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace (%s) shared schemes, error: %s", projectOrWorkspacePth, err)
		}

		log.Details("workspace (%s) user schemes: %v", workspaceName, workspaceUserSchemes)

		projects, err := xcodeproject.WorkspaceProjectReferences(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace referred projects, error: %s", err)
		}

		for _, project := range projects {
			projectName := filepath.Base(project)

			workspaceProjectUserSchemes, err := xcodeproject.UserSchemes(project)
			if err != nil {
				log.Fail("Failed to list project (%s) shared schemes, error: %s", project, err)
			}

			log.Details("workspace project (%s) user schemes: %v", projectName, workspaceProjectUserSchemes)

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
