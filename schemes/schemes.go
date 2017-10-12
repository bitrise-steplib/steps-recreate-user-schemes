package schemes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bitrise-io/go-utils/command/rubyscript"
	"github.com/bitrise-tools/go-xcode/xcodeproj"
)

// ReCreateSchemes ....
func ReCreateSchemes(projectOrWorkspacePth string) error {
	runner := rubyscript.New(recreateUserSchemesRubyScriptContent)
	bundleInstallCmd, err := runner.BundleInstallCommand(xcodeprojGemfileContent, "")
	if err != nil {
		return fmt.Errorf("failed to create bundle install command, error: %s", err)
	}

	if out, err := bundleInstallCmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return fmt.Errorf("bundle install failed, output: %s, error: %s", out, err)
	}

	runCmd, err := runner.RunScriptCommand()
	if err != nil {
		return fmt.Errorf("failed to create script runner command, error: %s", err)
	}

	envsToAppend := []string{"project_path=" + projectOrWorkspacePth}
	envs := append(runCmd.GetCmd().Env, envsToAppend...)

	runCmd.SetEnvs(envs...)

	out, err := runCmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run ruby script, output: %s, error: %s", out, err)
	}

	if out == "" {
		return nil
	}

	// OutputModel ...
	type OutputModel struct {
		Error string `json:"error"`
	}
	var output OutputModel
	if err := json.Unmarshal([]byte(out), &output); err != nil {
		return fmt.Errorf("failed to unmarshal output: %s", out)
	}

	if output.Error != "" {
		return fmt.Errorf("failed to get provisioning profile - bundle id mapping, error: %s", output.Error)
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
