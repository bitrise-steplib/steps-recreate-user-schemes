package schemes

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-xcode/xcodeproj"
)

func runRubyScriptForOutput(scriptContent, gemfileContent, inDir string, withEnvs []string) (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("bitrise")
	if err != nil {
		return "", err
	}

	// Write Gemfile to file and install
	if gemfileContent != "" {
		gemfilePth := path.Join(tmpDir, "Gemfile")
		if err := fileutil.WriteStringToFile(gemfilePth, gemfileContent); err != nil {
			return "", err
		}

		cmd := command.New("bundle", "install")

		if inDir != "" {
			cmd.SetDir(inDir)
		}

		withEnvs = append(withEnvs, "BUNDLE_GEMFILE="+gemfilePth)
		cmd.SetEnvs(withEnvs...)

		var outBuffer bytes.Buffer
		outWriter := bufio.NewWriter(&outBuffer)
		cmd.SetStdout(outWriter)

		var errBuffer bytes.Buffer
		errWriter := bufio.NewWriter(&errBuffer)
		cmd.SetStderr(errWriter)

		if err := cmd.Run(); err != nil {
			if errorutil.IsExitStatusError(err) {
				errMsg := ""
				if errBuffer.String() != "" {
					errMsg += fmt.Sprintf("error: %s\n", errBuffer.String())
				}
				if outBuffer.String() != "" {
					errMsg += fmt.Sprintf("output: %s", outBuffer.String())
				}
				if errMsg == "" {
					return "", err
				}

				return "", errors.New(errMsg)
			}
			return "", err
		}
	}

	// Write script to file and run
	rubyScriptPth := path.Join(tmpDir, "script.rb")
	if err := fileutil.WriteStringToFile(rubyScriptPth, scriptContent); err != nil {
		return "", err
	}

	var cmd *command.Model

	if gemfileContent != "" {
		cmd = command.New("bundle", "exec", "ruby", rubyScriptPth)
	} else {
		cmd = command.New("ruby", rubyScriptPth)
	}

	if inDir != "" {
		cmd.SetDir(inDir)
	}

	if len(withEnvs) > 0 {
		cmd.SetEnvs(withEnvs...)
	}

	var outBuffer bytes.Buffer
	outWriter := bufio.NewWriter(&outBuffer)
	cmd.SetStdout(outWriter)

	var errBuffer bytes.Buffer
	errWriter := bufio.NewWriter(&errBuffer)
	cmd.SetStderr(errWriter)

	if err := cmd.Run(); err != nil {
		if errorutil.IsExitStatusError(err) {
			errMsg := ""
			if errBuffer.String() != "" {
				errMsg += fmt.Sprintf("error: %s\n", errBuffer.String())
			}
			if outBuffer.String() != "" {
				errMsg += fmt.Sprintf("output: %s", outBuffer.String())
			}
			if errMsg == "" {
				return "", err
			}

			return "", errors.New(errMsg)
		}
		return "", err
	}

	return outBuffer.String(), nil
}

func runRubyScript(scriptContent, gemfileContent, inDir string, withEnvs []string) error {
	_, err := runRubyScriptForOutput(scriptContent, gemfileContent, inDir, withEnvs)
	return err
}

// ReCreateProjectUserSchemes ....
func ReCreateProjectUserSchemes(projectPth string) error {
	projectDir := filepath.Dir(projectPth)

	projectBase := filepath.Base(projectPth)
	envs := append(os.Environ(), "project_path="+projectBase, "LC_ALL=en_US.UTF-8")

	return runRubyScript(recreateUserSchemesRubyScriptContent, xcodeprojGemfileContent, projectDir, envs)
}

// ReCreateWorkspaceUserSchemes ...
func ReCreateWorkspaceUserSchemes(workspacePth string) error {
	projects, err := xcodeproj.WorkspaceProjectReferences(workspacePth)
	if err != nil {
		return err
	}

	for _, project := range projects {
		if err := ReCreateProjectUserSchemes(project); err != nil {
			return err
		}
	}

	return nil
}

func filesInDir(dir string) ([]string, error) {
	files := []string{}
	if err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	}); err != nil {
		return []string{}, err
	}
	return files, nil
}

func isUserSchemeFilePath(pth string) bool {
	regexpPattern := filepath.Join(".*[/]?xcuserdata", ".*[.]xcuserdatad", "xcschemes", ".+[.]xcscheme")
	regexp := regexp.MustCompile(regexpPattern)
	return (regexp.FindString(pth) != "")
}

func filterUserSchemeFilePaths(paths []string) []string {
	filteredPaths := []string{}
	for _, pth := range paths {
		if isUserSchemeFilePath(pth) {
			filteredPaths = append(filteredPaths, pth)
		}
	}

	sort.Strings(filteredPaths)

	return filteredPaths
}

func userSchemeFilePaths(projectOrWorkspacePth string) ([]string, error) {
	paths, err := filesInDir(projectOrWorkspacePth)
	if err != nil {
		return []string{}, err
	}
	return filterUserSchemeFilePaths(paths), nil
}

// ProjectUserSchemeFilePaths ...
func ProjectUserSchemeFilePaths(projectPth string) ([]string, error) {
	return userSchemeFilePaths(projectPth)
}

// WorkspaceUserSchemeFilePaths ...
func WorkspaceUserSchemeFilePaths(workspacePth string) ([]string, error) {
	workspaceSchemeFilePaths, err := userSchemeFilePaths(workspacePth)
	if err != nil {
		return []string{}, err
	}

	projects, err := xcodeproj.WorkspaceProjectReferences(workspacePth)
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		projectSchemeFilePaths, err := userSchemeFilePaths(project)
		if err != nil {
			return []string{}, err
		}
		workspaceSchemeFilePaths = append(workspaceSchemeFilePaths, projectSchemeFilePaths...)
	}

	sort.Strings(workspaceSchemeFilePaths)

	return workspaceSchemeFilePaths, nil
}

// SchemeNameFromPath ...
func SchemeNameFromPath(schemePth string) string {
	basename := filepath.Base(schemePth)
	ext := filepath.Ext(schemePth)
	if ext != xcodeproj.XCSchemeExt {
		return ""
	}
	return strings.TrimSuffix(basename, ext)
}

func userSchemes(projectOrWorkspacePth string) (map[string]bool, error) {
	schemePaths, err := userSchemeFilePaths(projectOrWorkspacePth)
	if err != nil {
		return map[string]bool{}, err
	}

	schemeMap := map[string]bool{}
	for _, schemePth := range schemePaths {
		schemeName := SchemeNameFromPath(schemePth)
		hasXCtest, err := xcodeproj.SchemeFileContainsXCTestBuildAction(schemePth)
		if err != nil {
			return map[string]bool{}, err
		}
		schemeMap[schemeName] = hasXCtest
	}

	return schemeMap, nil
}

// ProjectUserSchemes ...
func ProjectUserSchemes(projectPth string) (map[string]bool, error) {
	return userSchemes(projectPth)
}

// WorkspaceUserSchemes ...
func WorkspaceUserSchemes(workspacePth string) (map[string]bool, error) {
	schemeMap, err := userSchemes(workspacePth)
	if err != nil {
		return map[string]bool{}, err
	}

	projects, err := xcodeproj.WorkspaceProjectReferences(workspacePth)
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		projectSchemeMap, err := userSchemes(project)
		if err != nil {
			return map[string]bool{}, err
		}

		for name, hasXCtest := range projectSchemeMap {
			schemeMap[name] = hasXCtest
		}
	}

	return schemeMap, nil
}
