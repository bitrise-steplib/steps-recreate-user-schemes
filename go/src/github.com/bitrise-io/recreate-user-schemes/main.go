package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/bitrise-io/recreate-user-schemes/logger"
	"github.com/bitrise-io/xcode-utils/xcodeproj"
)

// -----------------------
// --- Functions
// -----------------------

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

	isWorkspace := xcodeproj.IsXCWorkspace(projectOrWorkspacePth)
	if isWorkspace {
		log.Info("Analyzing workspace: %s", projectOrWorkspacePth)
	} else {
		log.Info("Analyzing project: %s", projectOrWorkspacePth)
	}

	projectOrWorkspaceName := filepath.Base(projectOrWorkspacePth)

	// Shared schemes
	log.Info("Searching for shared schemes...")

	sharedSchemes := []string{}

	if isWorkspace {
		workspaceSharedSchemes, err := xcodeproj.WorkspaceSharedSchemes(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace (%s) shared schemes, error: %s", projectOrWorkspacePth, err)
		}

		sharedSchemes = workspaceSharedSchemes
	} else {
		projectSchemes, err := xcodeproj.ProjectSharedSchemes(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list project (%s) shared schemes, error: %s", projectOrWorkspacePth, err)
		}

		sharedSchemes = projectSchemes
	}

	log.Details("shared schemes: %v", sharedSchemes)

	if len(sharedSchemes) > 0 {
		log.Done("Shared schemes: %v", sharedSchemes)
		os.Exit(0)
	}

	// Generate schemes
	fmt.Println("")
	log.Warn("No shared schemes found, generating default user schemes...")

	if isWorkspace {
		if err := xcodeproj.ReCreateWorkspaceUserSchemes(projectOrWorkspacePth); err != nil {
			log.Fail("Failed to recreate workspace (%s) user schemes, error: %s", projectOrWorkspaceName, err)
		}
	} else {
		if err := xcodeproj.ReCreateProjectUserSchemes(projectOrWorkspacePth); err != nil {
			log.Fail("Failed to recreate project (%s) user schemes, error: %s", projectOrWorkspacePth, err)
		}
	}

	// Ensure user schemes
	log.Info("Ensure generated user schemes")

	userSchemes := []string{}

	if isWorkspace {
		workspaceUserSchemes, err := xcodeproj.WorkspaceUserSchemes(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list workspace (%s) shared schemes, error: %s", projectOrWorkspacePth, err)
		}

		userSchemes = workspaceUserSchemes
	} else {
		projectSchemes, err := xcodeproj.ProjectUserSchemes(projectOrWorkspacePth)
		if err != nil {
			log.Fail("Failed to list project (%s) shared schemes, error: %s", projectOrWorkspacePth, err)
		}

		userSchemes = projectSchemes
	}

	log.Details("generated user schemes: %v", userSchemes)

	if len(userSchemes) == 0 {
		log.Fail("No user schemes generated")
	}

	fmt.Println("")
	log.Done("Generated user schemes: %v", userSchemes)
}
