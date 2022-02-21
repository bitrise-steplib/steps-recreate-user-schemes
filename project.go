package main

import (
	"fmt"

	"github.com/bitrise-io/go-utils/pathutil"
	xcodeproject "github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcscheme"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcworkspace"
)

type container interface {
	// schemes returns schemes mapped to the project or workspace path
	schemes() (map[string][]xcscheme.Scheme, error)
	projects() ([]xcodeproject.XcodeProj, []string, error)
}

// projectContainer ...
type projectContainer struct {
	project xcodeproject.XcodeProj
}

func newProject(path string) (projectContainer, error) {
	if !xcodeproject.IsXcodeProj(path) {
		return projectContainer{}, fmt.Errorf("%s is not an xcode project", path)
	}

	project, err := xcodeproject.Open(path)
	if err != nil {
		return projectContainer{}, fmt.Errorf("opening the xcode project at %s failed: %w", path, err)
	}

	return projectContainer{
		project: project,
	}, nil
}

func (p projectContainer) schemes() (map[string][]xcscheme.Scheme, error) {
	projectSchemes, err := p.project.Schemes()
	if err != nil {
		return nil, fmt.Errorf("listing schemes in xcode project at %s failed: %w", p.project.Path, err)
	}

	containerToSchemes := make(map[string][]xcscheme.Scheme)
	containerToSchemes[p.project.Path] = projectSchemes

	return containerToSchemes, nil
}

func (p projectContainer) projects() ([]xcodeproject.XcodeProj, []string, error) {
	return []xcodeproject.XcodeProj{p.project}, []string{}, nil
}

// workspaceContainer ...
type workspaceContainer struct {
	workspace xcworkspace.Workspace
}

func newWorkspace(path string) (workspaceContainer, error) {
	if !xcworkspace.IsWorkspace(path) {
		return workspaceContainer{}, fmt.Errorf("%s is not an xcode workspace", path)
	}

	workspace, err := xcworkspace.Open(path)
	if err != nil {
		return workspaceContainer{}, fmt.Errorf("opening the xcode workspace at %s failed: %w", path, err)
	}

	return workspaceContainer{
		workspace: workspace,
	}, nil
}

func (w workspaceContainer) schemes() (map[string][]xcscheme.Scheme, error) {
	containerToSchemes, err := w.workspace.Schemes()
	if err != nil {
		return nil, fmt.Errorf("listing schemes in xcode workspace at %s failed: %w", w.workspace.Path, err)
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
			return nil, nil, fmt.Errorf("list xcode projects in the workspace at %s failed: can not check if path (%s) exists: %w", w.workspace.Path, projPath, err)
		} else if !exist {
			missingProjects = append(missingProjects, projPath)
			continue
		}

		project, err := xcodeproject.Open(projPath)
		if err != nil {
			return nil, nil, fmt.Errorf("opening the xcode project (%s) in the workspace at %s failed: %w", projPath, w.workspace.Path, err)
		}

		projects = append(projects, project)
	}

	return projects, missingProjects, nil
}

func openContainer(path string) (container, error) {
	if xcodeproject.IsXcodeProj(path) {
		container, err := newProject(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open xcode project: %w", err)
		}

		return container, nil
	} else if xcworkspace.IsWorkspace(path) {
		container, err := newWorkspace(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open xcode workspace: %w", err)
		}

		return container, nil
	}

	return nil, fmt.Errorf("project path (%s) has an invalid extension, excepted '%s' or '%s'",
		path,
		xcodeproject.XcodeProjExtension,
		xcworkspace.XCWorkspaceExtension,
	)
}
