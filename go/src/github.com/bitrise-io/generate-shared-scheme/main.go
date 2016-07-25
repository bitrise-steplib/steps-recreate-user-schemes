package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/bitrise-io/generate-shared-scheme/logger"
	"github.com/bitrise-io/generate-shared-scheme/retry"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
)

const (
	schemeFileExtension = ".xcscheme"
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

func fileList(dir string) ([]string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return []string{}, err
	}

	fileList := []string{}

	if err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		fileList = append(fileList, rel)

		return nil
	}); err != nil {
		return []string{}, err
	}
	return fileList, nil
}

func filterFilesWithExtensions(fileList []string, extension ...string) []string {
	filteredFileList := []string{}

	for _, file := range fileList {
		ext := filepath.Ext(file)

		for _, desiredExt := range extension {
			if ext == desiredExt {
				filteredFileList = append(filteredFileList, file)
				break
			}
		}
	}

	return filteredFileList
}

func properReturn(err error, out string) error {
	if err == nil {
		return nil
	}

	if errorutil.IsExitStatusError(err) && out != "" {
		return errors.New(out)
	}
	return err
}

func runCommand(envs []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if len(envs) > 0 {
		cmd.Env = append(cmd.Env, envs...)
	}
	outBytes, err := cmd.CombinedOutput()
	outStr := string(outBytes)
	return strings.TrimSpace(outStr), err
}

func reCreateUserSchemes(pth string) error {
	parseProjectRubyContent := `require 'xcodeproj'
require 'json'

project_path = ENV['project_path']
begin
  project = Xcodeproj::Project.open(project_path)
  project.recreate_user_schemes
  project.save
rescue => ex
  puts(ex.inspect.to_s)
  puts('--- Stack trace: ---')
  puts(ex.backtrace.to_s)
  exit(1)
end`

	tmpDir, err := pathutil.NormalizedOSTempDirPath("bitrise-init")
	if err != nil {
		return err
	}

	parseProjectRubyPth := path.Join(tmpDir, "reCreateSchemes.rb")
	if err := fileutil.WriteStringToFile(parseProjectRubyPth, parseProjectRubyContent); err != nil {
		return err
	}

	projectPthEnv := "project_path=" + pth

	out, err := runCommand([]string{projectPthEnv}, "ruby", parseProjectRubyPth)
	return properReturn(err, out)
}

func sharedSchemes(projectPth string) ([]string, error) {
	pattern := filepath.Join(projectPth, "xcshareddata", "xcschemes", "*.xcscheme")

	schemeFiles, err := filepath.Glob(pattern)
	if err != nil {
		return []string{}, err
	}

	regexp := regexp.MustCompile(filepath.Join(projectPth, "xcshareddata", "xcschemes", "(?P<scheme>.+).xcscheme"))

	schemeMap := map[string]bool{}
	for _, schemeFile := range schemeFiles {
		match := regexp.FindStringSubmatch(schemeFile)
		if len(match) > 1 {
			schemeMap[match[1]] = true
		}
	}

	schemes := []string{}
	for scheme := range schemeMap {
		schemes = append(schemes, scheme)
	}

	return schemes, nil
}

func schemes(projectPth string) ([]string, error) {
	pattern := filepath.Join(projectPth, "xcuserdata", "*.xcuserdatad", "xcschemes", "*.xcscheme")

	schemeFiles, err := filepath.Glob(pattern)
	if err != nil {
		return []string{}, err
	}

	regexp := regexp.MustCompile(filepath.Join(projectPth, "xcuserdata", ".*.xcuserdatad", "xcschemes", "(?P<scheme>.+).xcscheme"))

	schemes := []string{}
	for _, schemeFile := range schemeFiles {
		match := regexp.FindStringSubmatch(schemeFile)
		if len(match) > 1 {
			schemes = append(schemes, match[1])
		}
	}

	return schemes, nil
}

// -----------------------
// --- Main
// -----------------------

func main() {
	// Validate options
	projectPth := os.Getenv("project_path")

	log.Configs(projectPth)

	validateRequiredInput("project_path", projectPth)

	// Shared schemes
	log.Info("Searching for shared schemes...")

	sharedSchemes, err := sharedSchemes(projectPth)
	if err != nil {
		log.Fail("Failed to list shared schemes, error: %s", err)
	}

	log.Details("Found %d shared schemes", len(sharedSchemes))

	if len(sharedSchemes) > 0 {
		log.Done("Shared schemes: %v", sharedSchemes)
		os.Exit(0)
	}

	// User schemes
	log.Info("Searching for user schemes...")

	userSchemes, err := schemes(projectPth)
	if err != nil {
		log.Fail("Failed to list shared schemes, error: %s", err)
	}

	log.Details("Found %d user schemes", len(userSchemes))

	if len(userSchemes) > 0 {
		log.Done("User schemes: %v", userSchemes)
		os.Exit(0)
	}

	// Generating schemes
	log.Info("No shared or user schemes found, generating user schemes...")

	if err := reCreateUserSchemes(projectPth); err != nil {
		log.Fail("Failed to re create user schemes for poroject (%s), error: %s", projectPth, err)
	}

	// Wait some time to finish scheme generation
	if err := retry.Times(3).Wait(10).Retry(func(attempt uint) error {
		schemes, listErr := schemes(projectPth)
		if listErr != nil {
			return listErr
		}
		if len(schemes) == 0 {
			return errors.New("no user schemes generated")
		}

		userSchemes = schemes
		return nil
	}); err != nil {
		log.Fail("Generating user schemes failed: %s", err)
	}

	log.Details("Generated %d user schemes", len(userSchemes))

	if len(userSchemes) == 0 {
		log.Fail("No user schemes generated")
	}

	log.Done("User schemes: %v", userSchemes)

	fmt.Println("")
	log.Done("Done 🚀")
}
