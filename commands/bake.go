package commands

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jfrog/jfrog-cli-core/artifactory/commands/buildinfo"
	"github.com/jfrog/jfrog-cli-core/artifactory/commands/generic"
	"github.com/jfrog/jfrog-cli-core/artifactory/spec"
	"github.com/jfrog/jfrog-cli-core/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/jfrog/jfrog-cli-core/utils/config"
	rtBuildInfo "github.com/jfrog/jfrog-client-go/artifactory/buildinfo"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBuildName = "yocto-build"
	defaultBuildNum  = "1"
	defaultRepoName  = "yocto"
	defaultModule    = "build"
	defaultProject   = ""

	tmpDirectory    = "/build/tmp"
	lockFiles       = "/build/bitbake.lock,/build/bitbake.sock,/build/hashserve.sock"
	deployDirectory = "/build/tmp/deploy/"
	imagesDirectory = "/build/tmp/deploy/images/"

	uploadThreads = 10
	uploadRetries = 5
)

func GetBakeCommand() components.Command {
	return components.Command{
		Name:        "bake",
		Description: "Bake your firmware",
		Aliases:     []string{"build"},
		Arguments:   getBakeArguments(),
		Flags:       getBakeFlags(),
		EnvVars:     getBakeEnvVar(),
		Action: func(c *components.Context) error {
			return bakeCmd(c)
		},
	}
}

func getBakeArguments() []components.Argument {
	return []components.Argument{
		{
			Name:        "target",
			Description: "The bake target. Examples: core-image-base, core-image-minimal",
		},
	}
}

func getBakeFlags() []components.Flag {
	return []components.Flag{
		components.StringFlag{
			Name:         "run-folder",
			Description:  "The location of the root folder to run the process from",
			DefaultValue: ".",
		},
		components.StringFlag{
			Name:         "build-env",
			Description:  "The name of the \"oe-init-build-env\" to init the build env from",
			DefaultValue: "oe-init-build-env",
		},
		components.BoolFlag{
			Name:         "clean",
			Description:  "Clean before the build, and clean the build-info on start",
			DefaultValue: true,
		},
		components.BoolFlag{
			Name:         "build",
			Description:  "Perform a build. should be true unless you want to bypass and manually build",
			DefaultValue: true,
		},
		components.BoolFlag{
			Name:         "load",
			Description:  "Load the resulting build to artifactory",
			DefaultValue: true,
		},
		components.BoolFlag{
			Name:         "scan",
			Description:  "Scan the result with Xray",
			DefaultValue: false,
		},
		components.StringFlag{
			Name:         "repo",
			Description:  "Target repository to deploy to",
			DefaultValue: defaultRepoName,
		},
		components.StringFlag{
			Name:         "artifact-name",
			Description:  "Artifact name to deploy on to RT",
			DefaultValue: "",
		},
		components.StringFlag{
			Name:         "build-name",
			Description:  "The build name",
			DefaultValue: defaultBuildName,
		},
		components.StringFlag{
			Name:         "build-num",
			Description:  "The build number",
			DefaultValue: defaultBuildNum,
		},
		components.BoolFlag{
			Name:         "only-images",
			Description:  "Upload only the images as the artifacts of the build",
			DefaultValue: true,
		},
		components.StringFlag{
			Name:         "art-id",
			Description:  "The artifactory server ID",
			DefaultValue: "",
		},
	}
}

func getBakeEnvVar() []components.EnvVar {
	return []components.EnvVar{}
}

type bakeConfiguration struct {
	// The folder to run the commands from (i.e. working folder)
	runFolder string

	// The name of the bash script to run to init the build environment variables
	buildEnv string

	// Build target, see yocto/bitbake documentation
	target string

	// Clean the tmp build folder, and the build-info before performing the other commands
	clean bool

	// Perform the build. This will trigger the call to the bitbake command
	build bool

	// Load the result to artifactory. The results will be loaded as an artifact (for the result files), and build-info
	load bool

	// Scan the result with Xray
	scan bool

	// The  build name to use
	buildName string

	// The build number to use
	buildNum string

	// The repository name to deploy to
	repo string

	// The artifact name to use, if empty a default name will be generated based on the other configuration parameters
	artifactName string

	// Deploy only the resulting images to artifactory, if false it will deploy all the 'deploy' folder content
	onlyImages bool

	// The identification of artifactory configuration to use for the connection
	artId string
}

// Handler for the bakeCmd, reads the configuration, validates it and calls the doBakeCommand
func bakeCmd(c *components.Context) error {
	if len(c.Arguments) != 1 {
		return errors.New("Wrong number of arguments. Expected: 1, " + "Received: " + strconv.Itoa(len(c.Arguments)))
	}
	var conf = new(bakeConfiguration)
	conf.target = c.Arguments[0]
	conf.runFolder = c.GetStringFlagValue("run-folder")
	conf.buildEnv = c.GetStringFlagValue("build-env")
	conf.clean = c.GetBoolFlagValue("clean")
	conf.build = c.GetBoolFlagValue("build")
	conf.load = c.GetBoolFlagValue("load")
	conf.scan = c.GetBoolFlagValue("scan")
	conf.repo = c.GetStringFlagValue("repo")
	conf.artifactName = c.GetStringFlagValue("artifact-name")
	conf.buildName = c.GetStringFlagValue("build-name")
	conf.buildNum = c.GetStringFlagValue("build-num")
	conf.onlyImages = c.GetBoolFlagValue("only-images")
	conf.artId = c.GetStringFlagValue("art-id")

	if conf.scan && !conf.load {
		return errors.New("scanning can only be done after loading the result to Artifactory")
	}

	dirExists, err := fileutils.IsDirExists(conf.runFolder, true)

	if err != nil {
		return err
	}

	if !dirExists {
		return errors.New("run-folder does not exist")
	}

	err = doBakeCommand(conf)
	if err != nil {
		return err
	}

	return nil
}

// The main logic handler for the bakeCommand
func doBakeCommand(conf *bakeConfiguration) error {
	startedTime := time.Now().Format(rtBuildInfo.TimeFormat)
	// If the build flag is enabled we will execute the pre-steps (usually clean if enabled) and the build
	if conf.build {
		err := executePreSteps(conf)
		if err != nil {
			return err
		}

		err = executeBitBakeBuild(conf)
		if err != nil {
			return err
		}
	}

	// If the load flag is enabled we will deploy the results to artifactory
	if conf.load {
		// Calculate the artifactory configuration
		artConfExists, err := config.IsServerConfExists()

		if err != nil {
			return err
		}

		if !artConfExists {
			return errors.New("artifactory details are not set. please use 'conf' command first")
		}

		var rtDetails *config.ServerDetails

		if conf.artId != "" {
			rtDetails, err = config.GetSpecificConfig(conf.artId, false, false)

			if err != nil {
				return err
			}
		} else {
			rtDetails, err = config.GetDefaultServerConf()

			if err != nil {
				return err
			}
		}

		// Load/deploy results to RT
		err = loadResultToRT(conf, rtDetails, startedTime)
		if err != nil {
			return err
		}

		// Scan with Xray
		if conf.scan {
			err = scanResults()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Exec external command (with bash -c)
func execCommand(folder string, command string) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = folder
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

// Pre-steps for the build stage. Performs clean if enabled
func executePreSteps(conf *bakeConfiguration) error {
	log.Output("Running pre steps. Running directory=" + conf.runFolder)

	if conf.clean {
		log.Output("Cleaning tmp folder")
		err := removeContents(conf.runFolder + tmpDirectory)
		if err != nil {
			return err
		}

		for _, lockFile := range strings.Split(lockFiles, ",") {
			err = os.Remove(conf.runFolder + lockFile)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	return nil
}

// Execute the bitbake build stage
func executeBitBakeBuild(conf *bakeConfiguration) error {
	log.Output("Running Bit bake. target=" + conf.target + ". This may take a long time....")
	return execCommand(conf.runFolder, "source "+conf.buildEnv+" && bitbake "+conf.target)
}

// Loads/Deploy results to artifactory
func loadResultToRT(conf *bakeConfiguration, rtDetails *config.ServerDetails, startedTime string) error {
	log.Output("Loading the result to Artifactory")

	// Generate the general build configuration
	buildConf := generateBuildConfiguration(conf)

	// Clears aggregated local cache of build-info
	if conf.clean {
		err := cleanBuildInfo(buildConf)
		if err != nil {
			return err
		}
	}

	artifacts, err := handleBuildArtifacts(conf, buildConf, rtDetails)
	if err != nil {
		return err
	}

	dependencies, err := parseDependenciesFromManifest(conf)
	if err != nil {
		return err
	}

	err = createAndSaveBuildInfo(buildConf, artifacts, dependencies, startedTime)
	if err != nil {
		return err
	}

	return publishBuildInfo(buildConf, rtDetails)
}

func publishBuildInfo(buildConf *utils.BuildConfiguration, rtDetails *config.ServerDetails) error {
	artAuthDetails, err := rtDetails.CreateArtAuthConfig()
	if err != nil {
		return err
	}
	publishCommand := buildinfo.NewBuildPublishCommand()
	publishCommand.SetBuildConfiguration(buildConf)
	publishCommand.SetServerDetails(rtDetails)
	publishCommand.SetConfig(
		&rtBuildInfo.Configuration{
			ArtDetails: artAuthDetails,
		},
	)
	return publishCommand.Run()
}

// Create and save build-info
func createAndSaveBuildInfo(buildConf *utils.BuildConfiguration, artifacts []rtBuildInfo.Artifact, dependencies []rtBuildInfo.Dependency, startedTime string) error {
	// Construct the build module
	var modules []rtBuildInfo.Module
	module := rtBuildInfo.Module{Id: buildConf.Module, Type: "cpp", Artifacts: artifacts, Dependencies: dependencies}
	modules = append(modules, module)

	// Set properties
	buildInfo := &rtBuildInfo.BuildInfo{}
	buildInfo.Name = buildConf.BuildName
	buildInfo.Number = buildConf.BuildNumber
	buildInfo.Started = startedTime
	buildInfo.Modules = modules

	return utils.SaveBuildInfo(buildConf.BuildName, buildConf.BuildNumber, buildConf.Project, buildInfo)
}

func handleBuildArtifacts(conf *bakeConfiguration, buildConf *utils.BuildConfiguration, rtDetails *config.ServerDetails) ([]rtBuildInfo.Artifact, error) {
	var artifacts []rtBuildInfo.Artifact
	err := uploadBuildArtifact(conf, buildConf, rtDetails)
	if err != nil {
		return artifacts, err
	}
	return getDeployedBuildInfoArtifacts(buildConf)
}

// Upload the artifact files to artifactory
func uploadBuildArtifact(conf *bakeConfiguration, buildConf *utils.BuildConfiguration, rtDetails *config.ServerDetails) error {
	uploadCommand := generic.NewUploadCommand()

	var artifactFile string

	if conf.onlyImages {
		artifactFile = conf.runFolder + imagesDirectory
	} else {
		artifactFile = conf.runFolder + deployDirectory
	}

	_, err := fileutils.GetFileInfo(artifactFile, false)

	if err != nil {
		return err
	}

	repoName := conf.repo
	artifactName := conf.artifactName
	if len(artifactName) == 0 {
		artifactName = conf.buildName + ":" + conf.buildNum
	}

	target := fmt.Sprintf("%s/%s/", repoName, artifactName)

	uploadSpec := spec.NewBuilder().
		Pattern(artifactFile).
		Target(target).
		Recursive(true).
		IncludeDirs(true).
		Flat(conf.onlyImages).
		BuildSpec()

	uploadCommand.SetServerDetails(rtDetails)
	uploadCommand.SetSpec(uploadSpec)
	uploadCommand.SetBuildConfiguration(buildConf)
	uploadCommand.SetUploadConfiguration(
		&utils.UploadConfiguration{
			Threads: uploadThreads,
			Retries: uploadRetries,
		},
	)

	return uploadCommand.Run()
}

// Extract the deployed artifacts from the partial file.
// Build artifacts were deployed by the upload-command, and build-info was produced and saved as Partial on the file-system.
func getDeployedBuildInfoArtifacts(buildConf *utils.BuildConfiguration) ([]rtBuildInfo.Artifact, error) {
	uploadPartials, err := utils.ReadPartialBuildInfoFiles(buildConf.BuildName, buildConf.BuildNumber, "")
	if len(uploadPartials) < 1 || len(uploadPartials[0].Artifacts) < 1 || err != nil {
		return nil, err
	}
	return uploadPartials[0].Artifacts, nil
}

// Checks if the 'dep' dependency already exists in the 'dependencies' list
func contains(dependencies []rtBuildInfo.Dependency, dep rtBuildInfo.Dependency) bool {
	for _, compDep := range dependencies {
		if compDep.Id == dep.Id {
			return true
		}
	}
	return false
}

// Parses the dependencies from the Yocto manifest file/s and returns the full list
func parseDependenciesFromManifest(conf *bakeConfiguration) ([]rtBuildInfo.Dependency, error) {
	resultDeps := make([]rtBuildInfo.Dependency, 0)
	manifestFiles, err := findManifestFiles(conf.runFolder)
	if err != nil {
		return nil, err
	}

	for _, manifestFile := range manifestFiles {
		data, err := ioutil.ReadFile(manifestFile)
		if err != nil {
			return nil, err
		}

		dataStr := string(data)
		resultDeps = parseManifestContent(dataStr, resultDeps)
	}

	return resultDeps, nil
}

// Parses the manifestContent in the input str, adds dependencies which are not already detected, and returns the updated slice
func parseManifestContent(dataStr string, resultDeps []rtBuildInfo.Dependency) []rtBuildInfo.Dependency {
	depsLines := strings.Split(dataStr, "\n")

	for _, depLine := range depsLines {
		depLine = strings.TrimSpace(depLine)
		if depLine != "" {
			depParts := strings.Split(depLine, " ")
			if len(depParts) == 3 {
				depId := depParts[0] + ":" + depParts[2]
				hasher := sha1.New()
				hasher.Write([]byte(depId))
				sha := hex.EncodeToString(hasher.Sum(nil))

				dep := rtBuildInfo.Dependency{
					Id:       depId,
					Checksum: &rtBuildInfo.Checksum{Sha1: sha},
					Type:     "os-package",
					Scopes:   []string{depParts[1]},
				}
				if !contains(resultDeps, dep) {
					resultDeps = append(resultDeps, dep)
				}
			}
		}
	}
	return resultDeps
}

// Lookup files matching a pattern within a specific root directory
func walkMatch(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

// Look for the Yocto manifest files within the deployment images directory
func findManifestFiles(folder string) ([]string, error) {
	results, err := walkMatch(folder+imagesDirectory, "*.manifest")
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.New("was unable to locate the manifest file")
	}
	return results, nil
}

// Generate a build configuration
func generateBuildConfiguration(conf *bakeConfiguration) *utils.BuildConfiguration {
	buildConfiguration := utils.BuildConfiguration{
		BuildName:   conf.buildName,
		BuildNumber: conf.buildNum,
		Module:      defaultModule,
		Project:     defaultProject,
	}
	return &buildConfiguration
}

// Executes a build-info clear command to clear the local cache of build-info on the specific build
func cleanBuildInfo(buildConf *utils.BuildConfiguration) error {
	cleanCommand := buildinfo.NewBuildCleanCommand()
	cleanCommand.SetBuildConfiguration(buildConf)
	return cleanCommand.Run()
}

// Xray scan - to be implemented in the future
func scanResults() error {
	log.Output("Scanning results")
	return errors.New("xray scanning is not yet supported")
}
