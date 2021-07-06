package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-steputils/input"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	xcodeproject "github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcscheme"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcworkspace"
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

type container interface {
	// schemes returns schemes mapped to the project or workspace path
	schemes() (map[string][]xcscheme.Scheme, error)
	projects() ([]xcodeproject.XcodeProj, []string, error)
}

type projectContainer struct {
	project xcodeproject.XcodeProj
}

func newProject(path string) (projectContainer, error) {
	if !xcodeproject.IsXcodeProj(path) {
		return projectContainer{}, fmt.Errorf("path (%s) is not a Project", path)
	}

	project, err := xcodeproject.Open(path)
	if err != nil {
		return projectContainer{}, fmt.Errorf("failed to open Project (%s): %v", path, err)
	}

	return projectContainer{
		project: project,
	}, nil
}

func (p projectContainer) schemes() (map[string][]xcscheme.Scheme, error) {
	projectSchemes, err := p.project.Schemes()
	if err != nil {
		return nil, fmt.Errorf("failed to list Schemes in Project (%s): %v", p.project.Path, err)
	}

	containerToSchemes := make(map[string][]xcscheme.Scheme)
	containerToSchemes[p.project.Path] = projectSchemes

	return containerToSchemes, nil
}

func (p projectContainer) projects() ([]xcodeproject.XcodeProj, []string, error) {
	return []xcodeproject.XcodeProj{p.project}, []string{}, nil
}

type workspaceContainer struct {
	workspace xcworkspace.Workspace
}

func newWorkspace(path string) (workspaceContainer, error) {
	if !xcworkspace.IsWorkspace(path) {
		return workspaceContainer{}, fmt.Errorf("path (%s) is not a Workspace", path)
	}

	workspace, err := xcworkspace.Open(path)
	if err != nil {
		return workspaceContainer{}, fmt.Errorf("failed to open Workspace (%s): %v", path, err)
	}

	return workspaceContainer{
		workspace: workspace,
	}, nil
}

func (w workspaceContainer) schemes() (map[string][]xcscheme.Scheme, error) {
	containerToSchemes, err := w.workspace.Schemes()
	if err != nil {
		return nil, fmt.Errorf("failed to list Schemes in Workspace (%s): %v", w.workspace.Path, err)
	}

	return containerToSchemes, nil
}

func (w workspaceContainer) projects() ([]xcodeproject.XcodeProj, []string, error) {
	projPaths, err := w.workspace.ProjectFileLocations()
	if err != nil {
		return nil, nil, err
	}

	var projects []xcodeproject.XcodeProj
	var missingProjects []string
	for _, projPath := range projPaths {
		if exist, err := pathutil.IsPathExists(projPath); err != nil {
			return nil, nil, fmt.Errorf("failed to list Projects in the Workspace (%s), can not check if path (%s) exists: %v", w.workspace.Path, projPath, err)
		} else if !exist {
			missingProjects = append(missingProjects, projPath)
			continue
		}

		project, err := xcodeproject.Open(projPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open Project (%s) in the Workspace (%s): %v", projPath, w.workspace.Path, err)
		}

		projects = append(projects, project)
	}

	return projects, missingProjects, nil
}

func main() {
	configs := createConfigsModelFromEnvs()
	configs.print()

	fmt.Println()

	if err := configs.validate(); err != nil {
		failf("Issue with input: %s", err)
	}

	var err error
	configs.ProjectPath, err = pathutil.AbsPath(configs.ProjectPath)
	if err != nil {
		failf("Failed to convert Project or Workspace path to absolute path: %v", err)
	}

	container, err := openContainer(configs.ProjectPath)
	if err != nil {
		failf("Error: %v", err)
	}

	// Schemes
	log.Infof("Searching for Project or Workspace shared Schemes...")
	containerToSchemes, err := container.schemes()
	if err != nil {
		failf("Could not list Schemes: %v", err)
	}

	fmt.Println()
	log.Donef("Projects:")
	for container := range containerToSchemes {
		log.Printf("- %s", relativeToWorkspace(container, configs.ProjectPath))
	}

	fmt.Println()
	log.Donef("Schemes:")
	var sharedSchemes []xcscheme.Scheme
	for container, schemes := range containerToSchemes {
		for _, scheme := range schemes {
			if scheme.IsShared {
				sharedSchemes = append(sharedSchemes, scheme)
				log.Printf(colorstring.Green(fmt.Sprintf("- %s (Shared in %s)", scheme.Name, filepath.Base(container))))
			} else {
				log.Printf(colorstring.Yellow(fmt.Sprintf("- %s (User in %s)", scheme.Name, filepath.Base(container))))
			}
		}
	}

	log.Printf("Shared scheme count: %d", len(sharedSchemes))
	if len(sharedSchemes) > 0 {
		os.Exit(0)
	}

	// Generate schemes
	fmt.Println()
	log.Errorf("No shared schemes found...")
	log.Errorf("The newly generated schemes, may differ from the ones in your project.")
	log.Errorf("Make sure to share your schemes, to prevent unexpected behaviour.")

	fmt.Println()
	log.Infof("Generating shared schemes")

	projects, missingProjects, err := container.projects()
	if err != nil {
		failf("Error: %v", err)
	}

	for _, missingProject := range missingProjects {
		log.Warnf("Skipping Project (%s), as it is not present", relativeToWorkspace(missingProject, configs.ProjectPath))
	}

	for _, project := range projects {
		log.Printf("Recreating schemes for: %s", filepath.Base(project.Path))
		if err := project.ReCreateSharedSchemes(); err != nil {
			failf("Failed to recreate schemes for project (%s): %v", filepath.Base(project.Path), err)
		}
	}

	// Ensure user schemes
	fmt.Println()
	log.Infof("Ensure generated schemes")

	container, err = openContainer(configs.ProjectPath)
	if err != nil {
		failf("Error: %v", err)
	}
	containerToSchemesNew, err := container.schemes()
	if err != nil {
		failf("Could not list Schemes: %v", err)
	}

	fmt.Println()
	log.Donef("Generated shared Schemes:")
	var sharedSchemesGenerated []xcscheme.Scheme
	var count int
	for container, schemes := range containerToSchemesNew {
		for _, scheme := range schemes {
			if scheme.IsShared {
				count += 1
				sharedSchemesGenerated = append(sharedSchemesGenerated, scheme)
				log.Printf(colorstring.Green(fmt.Sprintf("- %s (Shared in %s)", scheme.Name, filepath.Base(container))))
			}
		}
	}

	log.Printf("Generated shared Scheme count: %d", count)
}

func relativeToWorkspace(project, workspace string) string {
	parentDir, _ := filepath.Split(workspace)
	relPath, err := filepath.Rel(filepath.Join(parentDir), project)
	if err != nil {
		log.Warnf("%s", err)
		return project
	}

	return relPath
}

func openContainer(path string) (container, error) {
	if xcodeproject.IsXcodeProj(path) {
		container, err := newProject(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open Project: %v", err)
		}

		return container, nil
	} else if xcworkspace.IsWorkspace(path) {
		container, err := newWorkspace(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open Workspace: %v", err)
		}

		return container, nil
	}

	return nil, fmt.Errorf("project path (%s) has an invalid extension, excepted '%s' or '%s'",
		path,
		xcodeproject.XcodeProjExtension,
		xcworkspace.XCWorkspaceExtension,
	)
}
