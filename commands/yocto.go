package commands

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jfrog/jfrog-cli-core/artifactory/commands"
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
)

const (
	defaultBuildName = "yocto-build"
	defaultBuildNum  = "1.0"
	defaultRepoName  = "yocto"
	defaultModule    = "build"
	defaultProject   = ""

	tmpDirectory    = "/build/tmp"
	lockFile        = "/build/bitbake.lock"
	deployDirectory = "/build/tmp/deploy/"
	imagesDirectory = "/build/tmp/deploy/images/"

	uploadThreads = 10
	uploadRetries = 5
)

func GetConfigCommand() components.Command {
	return components.Command{
		Name:        "config",
		Description: "Configure artifactory settings",
		Aliases:     []string{"conf"},
		Arguments:   []components.Argument{},
		Flags:       []components.Flag{},
		EnvVars:     []components.EnvVar{},
		Action: func(c *components.Context) error {
			return configCmd(c)
		},
	}
}

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

func configCmd(c *components.Context) error {
	artConfigCommand := commands.NewConfigCommand()
	artConfigCommand.SetInteractive(true)
	return artConfigCommand.Run()
}

func getBakeArguments() []components.Argument {
	return []components.Argument{
		{
			Name:        "run-folder",
			Description: "The location of the root folder to run the process from",
		},
		{
			Name:        "build-env",
			Description: "The location of the \"oe-init-build-env\" to init the build env from",
		},
		{
			Name:        "target",
			Description: "The bake target. Examples: core-image-base, core-image-minimal",
		},
	}
}

func getBakeFlags() []components.Flag {
	return []components.Flag{
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
			Name:         "artifactName",
			Description:  "Artifact name to deploy on to RT",
			DefaultValue: "",
		},
		components.StringFlag{
			Name:         "buildName",
			Description:  "The build name",
			DefaultValue: defaultBuildName,
		},
		components.StringFlag{
			Name:         "buildNum",
			Description:  "The build number",
			DefaultValue: defaultBuildNum,
		},
		components.BoolFlag{
			Name:         "onlyImages",
			Description:  "Upload only the images as the artifacts of the build",
			DefaultValue: true,
		},
		components.StringFlag{
			Name:         "artId",
			Description:  "The artifactory server ID",
			DefaultValue: "",
		},
	}
}

func getBakeEnvVar() []components.EnvVar {
	return []components.EnvVar{}
}

type bakeConfiguration struct {
	runFolder    string
	buildEnv     string
	target       string
	clean        bool
	build        bool
	load         bool
	scan         bool
	buildName    string
	buildNum     string
	repo         string
	artifactName string
	onlyImages   bool
	artId        string
}

func bakeCmd(c *components.Context) error {
	if len(c.Arguments) != 3 {
		return errors.New("Wrong number of arguments. Expected: 3, " + "Received: " + strconv.Itoa(len(c.Arguments)))
	}
	var conf = new(bakeConfiguration)
	conf.runFolder = c.Arguments[0]
	conf.buildEnv = c.Arguments[1]
	conf.target = c.Arguments[2]
	conf.clean = c.GetBoolFlagValue("clean")
	conf.build = c.GetBoolFlagValue("build")
	conf.load = c.GetBoolFlagValue("load")
	conf.scan = c.GetBoolFlagValue("scan")
	conf.repo = c.GetStringFlagValue("repo")
	conf.artifactName = c.GetStringFlagValue("artifactName")
	conf.buildName = c.GetStringFlagValue("buildName")
	conf.buildNum = c.GetStringFlagValue("buildNum")
	conf.onlyImages = c.GetBoolFlagValue("onlyImages")
	conf.artId = c.GetStringFlagValue("artId")

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

func doBakeCommand(conf *bakeConfiguration) error {
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

	if conf.load {
		artConfExists, err := config.IsArtifactoryConfExists()

		if err != nil {
			return err
		}

		if !artConfExists {
			return errors.New("artifactory details are not set. please use 'conf' command first")
		}

		var rtDetails *config.ArtifactoryDetails

		if conf.artId != "" {
			rtDetails, err = config.GetArtifactorySpecificConfig(conf.artId, false, false)

			if err != nil {
				return err
			}
		} else {
			rtDetails, err = config.GetDefaultArtifactoryConf()

			if err != nil {
				return err
			}
		}

		err = loadResultToRT(conf, rtDetails)
		if err != nil {
			return err
		}

		if conf.scan {
			err = scanResults(conf, rtDetails)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func execCommand(folder string, command string) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = folder
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func executePreSteps(conf *bakeConfiguration) error {
	log.Output("Running pre steps. Running directory=" + conf.runFolder)

	if conf.clean {
		log.Output("Cleaning tmp folder")
		err := os.RemoveAll(conf.runFolder + tmpDirectory)
		if err != nil {
			return err
		}

		err = os.Remove(conf.runFolder + lockFile)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func executeBitBakeBuild(conf *bakeConfiguration) error {
	log.Output("Running Bit bake. target=" + conf.target + ". This may take a long time....")
	return execCommand(conf.runFolder, "source "+conf.buildEnv+" && bitbake "+conf.target)
}

func loadResultToRT(conf *bakeConfiguration, rtDetails *config.ArtifactoryDetails) error {
	log.Output("Loading the result to Artifactory")

	buildConf := generateBuildConfiguration(conf)

	if conf.clean {
		err := cleanBuildInfo(buildConf)
		if err != nil {
			return err
		}
	}

	buildPath, err := uploadBuildArtifact(conf, buildConf, rtDetails)
	if err != nil {
		return err
	}

	err = loadDependencies(conf, buildConf)
	if err != nil {
		return err
	}

	err = publishBuildInfo(conf, buildPath, buildConf, rtDetails)
	if err != nil {
		return err
	}

	return nil
}

func publishBuildInfo(conf *bakeConfiguration, buildURL string, buildConf *utils.BuildConfiguration, rtDetails *config.ArtifactoryDetails) error {
	artAuthDetails, err := rtDetails.CreateArtAuthConfig()
	if err != nil {
		return err
	}

	publishCommand := buildinfo.NewBuildPublishCommand()
	publishCommand.SetBuildConfiguration(buildConf)
	publishCommand.SetRtDetails(rtDetails)
	publishCommand.SetConfig(
		&rtBuildInfo.Configuration{
			ArtDetails: artAuthDetails,
			BuildUrl:   buildURL,
		},
	)
	return publishCommand.Run()
}

func uploadBuildArtifact(conf *bakeConfiguration, buildConf *utils.BuildConfiguration, rtDetails *config.ArtifactoryDetails) (string, error) {
	uploadCommand := generic.NewUploadCommand()

	var artifactFile string

	if conf.onlyImages {
		artifactFile = conf.runFolder + imagesDirectory
	} else {
		artifactFile = conf.runFolder + deployDirectory
	}

	_, err := fileutils.GetFileInfo(artifactFile, false)

	if err != nil {
		return "", err
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
		BuildSpec()

	uploadCommand.SetRtDetails(rtDetails)
	uploadCommand.SetSpec(uploadSpec)
	uploadCommand.SetBuildConfiguration(buildConf)
	uploadCommand.SetUploadConfiguration(
		&utils.UploadConfiguration{
			Threads: uploadThreads,
			Retries: uploadRetries,
			Symlink: true,
		},
	)

	return target, uploadCommand.Run()
}

func loadDependencies(conf *bakeConfiguration, buildConf *utils.BuildConfiguration) error {
	deps, err := loadDependenciesFromManifest(conf)

	if err != nil {
		return err
	}

	if deps != nil {

		buildInfo := &rtBuildInfo.BuildInfo{}
		var modules []rtBuildInfo.Module
		// Save build-info.
		module := rtBuildInfo.Module{Id: buildConf.Module, Dependencies: deps}
		modules = append(modules, module)

		buildInfo.Modules = modules
		return utils.SaveBuildInfo(buildConf.BuildName, buildConf.BuildNumber, buildInfo)
	} else {
		return nil
	}
}

func contains(dependencies []rtBuildInfo.Dependency, dep rtBuildInfo.Dependency) bool {
	for _, compDep := range dependencies {
		if compDep.Id == dep.Id {
			return true
		}
	}
	return false
}

func loadDependenciesFromManifest(conf *bakeConfiguration) ([]rtBuildInfo.Dependency, error) {
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
	}

	return resultDeps, nil
}

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

func generateBuildConfiguration(conf *bakeConfiguration) *utils.BuildConfiguration {
	buildConfiguration := utils.BuildConfiguration{
		BuildName:   conf.buildName,
		BuildNumber: conf.buildNum,
		Module:      defaultModule,
		Project:     defaultProject,
	}
	return &buildConfiguration
}

func cleanBuildInfo(buildConf *utils.BuildConfiguration) error {
	cleanCommand := buildinfo.NewBuildCleanCommand()
	cleanCommand.SetBuildConfiguration(buildConf)
	return cleanCommand.Run()
}

func scanResults(conf *bakeConfiguration, details *config.ArtifactoryDetails) error {
	log.Output("Scanning results")
	return errors.New("xray scanning is not yet supported")
}
