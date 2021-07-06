package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/input"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
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
	targets() ([]xcodeproj.Target, error)
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

func (p projectContainer) targets() ([]xcodeproject.Target, error) {
	return p.project.Proj.Targets, nil
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

func (w workspaceContainer) targets() ([]xcodeproject.Target, error) {
	return []xcodeproj.Target{}, nil
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

	var container container
	if xcodeproject.IsXcodeProj(configs.ProjectPath) {
		var err error
		if container, err = newProject(configs.ProjectPath); err != nil {
			failf("Failed to open project: %v", err)
		}
	} else if xcworkspace.IsWorkspace(configs.ProjectPath) {
		var err error
		if container, err = newWorkspace(configs.ProjectPath); err != nil {
			failf("Failed to open workspace: %v", err)
		}
	} else {
		failf("Project path has an invalid extension: '%s' or '%s' excepted, got (%s)",
			xcodeproject.XcodeProjExtension,
			xcworkspace.XCWorkspaceExtension,
			configs.ProjectPath,
		)
	}

	// Schemes
	log.Infof("Searching for Project or Workspace shared Schemes...")
	containerToSchemes, err := container.schemes()
	if err != nil {
		failf("Could not list Schemes: %v", err)
	}

	fmt.Println()
	log.Donef("Schemes:")
	var sharedSchemes []xcscheme.Scheme
	for container, schemes := range containerToSchemes {
		log.Printf(" %s", container)
		for _, scheme := range schemes {
			if scheme.IsShared {
				sharedSchemes = append(sharedSchemes, scheme)
				log.Printf(colorstring.Green(" - %s (Shared scheme)", scheme.Name))
			} else {
				log.Printf(colorstring.Yellow(" - %s (User scheme)", scheme.Name))
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
	log.Infof("Generating user schemes")

	// targets :=
	// dependentTargets := mainTarget.DependentExecutableProductTargets()

	// var uiTestTargets []xcodeproj.Target
	// for _, target := range xcproj.Proj.Targets {
	// 	if target.IsUITestProduct() && target.DependesOn(mainTarget.ID) {
	// 		uiTestTargets = append(uiTestTargets, target)
	// 	}
	// }

	// Ensure user schemes
	fmt.Println()
	log.Infof("Ensure generated schemes")

	var sharedSchemesGenerated []xcscheme.Scheme
	for container, schemes := range containerToSchemes {
		log.Printf(" %s", container)
		for _, scheme := range schemes {
			if scheme.IsShared {
				sharedSchemesGenerated = append(sharedSchemesGenerated, scheme)
				log.Printf(colorstring.Green(" - %s (Shared scheme)", scheme.Name))
			}
		}
	}

	log.Printf("Shared scheme count: %d", len(sharedSchemesGenerated))

	//
	// if isWorkspace {
	// 	projects, err := xcodeproj.WorkspaceProjectReferences(configs.ProjectPath)
	// 	if err != nil {
	// 		failf("Failed to get workspace referred projects, error: %s", err)
	// 	}

	// 	for _, project := range projects {
	// 		if exist, err := pathutil.IsPathExists(project); err != nil {
	// 			failf("Failed to check if path (%s) exist, error: %s", project, err)
	// 		} else if !exist {
	// 			log.Warnf("skip recreating user schemes for: %s, issue: file not exists", project)
	// 			continue
	// 		}

	// 		log.Printf("recreating user schemes for: %s", project)

	// 		if err := schemes.ReCreateProjectUserSchemes(project); err != nil {
	// 			failf("Failed to recreate user schemes for project (%s), error: %s", project, err)
	// 		}
	// 	}

	// } else {
	// 	log.Printf("recreating user schemes for: %s", configs.ProjectPath)
	// 	err = schemes.ReCreateProjectUserSchemes(configs.ProjectPath)
	// }

	// if err != nil {
	// 	failf("Failed to recreate user schemes, error: %s", err)
	// }
}

/*
const recreateUserSchemesRubyScriptContent = `require 'xcodeproj'

project_path = ENV['project_path']

begin
  raise 'empty path' if project_path.empty?

  project = Xcodeproj::Project.open project_path

  #-----
  # Separate targets
  native_targets = project.native_targets

  build_targets = []
  test_targets = []

  native_targets.each do |target|
    test_targets << target if target.test_target_type?
    build_targets << target unless target.test_target_type?
  end

  raise 'no build target found' unless build_targets.count

  #-----
  # Map targets
  target_mapping = {}

  build_targets.each do |target|
    target_mapping[target] = []
  end

  test_targets.each do |target|
    target_dependencies = target.dependencies

    dependent_targets = []
    target_dependencies.each do |target_dependencie|
      dependent_targets << target_dependencie.target
    end

    dependent_targets.each do |dependent_target|
      if build_targets.include? dependent_target
        target_mapping[dependent_target] = [] unless target_mapping[dependent_target]
        target_mapping[dependent_target] << target
      end
    end
  end

  #-----
  # Create schemes
  target_mapping.each do |build_t, test_ts|
    scheme = Xcodeproj::XCScheme.new

    scheme.set_launch_target build_t
    scheme.add_build_target build_t

    test_ts.each do |test_t|
      scheme.add_test_target test_t
    end

    scheme.save_as project_path, build_t.name
  end
rescue => ex
  puts ex.inspect.to_s
  puts '--- Stack trace: ---'
  puts ex.backtrace.to_s
  exit 1
end
`*/
