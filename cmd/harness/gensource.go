// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package harness

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/dancewing/revel"
)

// GenerateSource the app:
// 1. Generate the the main.go file.
// 2. Run the appropriate "go build" command.
// Requires that revel.Init has been called previously.
// Returns the path to the built binary, and an error if there was a problem building it.
func GenerateSource(buildFlags ...string) (app *App, compileError *revel.Error) {
	// First, clear the generated files (to avoid them messing with ProcessSource).

	cleanSource("tmp", "routes")

	sourceInfo, compileError := ProcessSource(revel.CodePaths)
	if compileError != nil {
		return nil, compileError
	}

	// Add the db.import to the import paths.
	if dbImportPath, found := revel.Config.String("db.import"); found {
		sourceInfo.InitImportPaths = append(sourceInfo.InitImportPaths, dbImportPath)
	}

	// Generate two source files.
	templateArgs := map[string]interface{}{
		"Controllers":    sourceInfo.ControllerSpecs(),
		"ValidationKeys": sourceInfo.ValidationKeys,
		"ImportPaths":    calcImportAliases(sourceInfo),
		"TestSuites":     sourceInfo.TestSuites(),
	}
	genSource("tmp", "main.go", RevelMainTemplate, templateArgs)
	genSource("routes", "routes.go", RevelRoutesTemplate, templateArgs)

	// Read build config.
	buildTags := revel.Config.StringDefault("build.tags", "")

	// Build the user program (all code under app).
	// It relies on the user having "go" installed.
	goPath, err := exec.LookPath("go")
	if err != nil {
		revel.ERROR.Fatalf("Go executable not found in PATH.")
	}

	pkg, err := build.Default.Import(revel.ImportPath, "", build.FindOnly)
	if err != nil {
		revel.ERROR.Fatalln("Failure importing", revel.ImportPath)
	}

	// Binary path is a combination of $GOBIN/revel.d directory, app's import path and its name.
	binName := filepath.Join(pkg.BinDir, "revel.d", revel.ImportPath, filepath.Base(revel.BasePath))

	// Change binary path for Windows build
	goos := runtime.GOOS
	if goosEnv := os.Getenv("GOOS"); goosEnv != "" {
		goos = goosEnv
	}
	if goos == "windows" {
		binName += ".exe"
	}

	gotten := make(map[string]struct{})
	for {
		appVersion := getAppVersion()

		buildTime := time.Now().UTC().Format(time.RFC3339)
		versionLinkerFlags := fmt.Sprintf("-X %s/app.AppVersion=%s -X %s/app.BuildTime=%s",
			revel.ImportPath, appVersion, revel.ImportPath, buildTime)

		// TODO remove version check for versionLinkerFlags after Revel becomes Go min version to go1.5
		goVersion, err := strconv.ParseFloat(runtime.Version()[2:5], 64)
		// runtime.Version() may return commit hash, we assume it is above 1.5
		if goVersion < 1.5 && err == nil {
			versionLinkerFlags = fmt.Sprintf("-X %s/app.AppVersion \"%s\" -X %s/app.BuildTime \"%s\"",
				revel.ImportPath, appVersion, revel.ImportPath, buildTime)
		}
		flags := []string{
			"build",
			"-i",
			"-ldflags", versionLinkerFlags,
			"-tags", buildTags,
			"-o", binName}

		// Add in build flags
		flags = append(flags, buildFlags...)

		// This is Go main path
		// Note: It's not applicable for filepath.* usage
		flags = append(flags, path.Join(revel.ImportPath, "app", "tmp"))

		buildCmd := exec.Command(goPath, flags...)
		revel.TRACE.Println("Exec:", buildCmd.Args)
		output, err := buildCmd.CombinedOutput()

		// If the build succeeded, we're done.
		if err == nil {
			return NewApp(binName), nil
		}
		revel.ERROR.Println(string(output))

		// See if it was an import error that we can go get.
		matches := importErrorPattern.FindStringSubmatch(string(output))
		if matches == nil {
			return nil, newCompileError(output)
		}

		// Ensure we haven't already tried to go get it.
		pkgName := matches[1]
		if _, alreadyTried := gotten[pkgName]; alreadyTried {
			return nil, newCompileError(output)
		}
		gotten[pkgName] = struct{}{}

		// Execute "go get <pkg>"
		getCmd := exec.Command(goPath, "get", pkgName)
		revel.TRACE.Println("Exec:", getCmd.Args)
		getOutput, err := getCmd.CombinedOutput()
		if err != nil {
			revel.ERROR.Println(string(getOutput))
			return nil, newCompileError(output)
		}

		// Success getting the import, attempt to build again.
	}

	// TODO remove this unreachable code and document it
	revel.ERROR.Fatalf("Not reachable")
	return nil, nil
}
