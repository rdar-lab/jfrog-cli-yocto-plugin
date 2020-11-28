package commands

import (
	"context"
	"errors"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"strconv"
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
			Name:         "load",
			Description:  "Load the resulting build to artifactory",
			DefaultValue: true,
		},
		components.BoolFlag{
			Name:         "scan",
			Description:  "Scan the result with Xray",
			DefaultValue: false,
		},
	}
}

func getBakeEnvVar() []components.EnvVar {
	return []components.EnvVar{}
}

type bakeConfiguration struct {
	runFolder string
	buildEnv  string
	target    string
	load      bool
	scan      bool
}

func bakeCmd(c *components.Context) error {
	if len(c.Arguments) != 3 {
		return errors.New("Wrong number of arguments. Expected: 3, " + "Received: " + strconv.Itoa(len(c.Arguments)))
	}
	var conf = new(bakeConfiguration)
	conf.runFolder = c.Arguments[0]
	conf.buildEnv = c.Arguments[1]
	conf.target = c.Arguments[2]
	conf.load = c.GetBoolFlagValue("load")
	conf.scan = c.GetBoolFlagValue("scan")

	if conf.scan && !conf.load {
		return errors.New("scanning can only be done after loading the result to Artifactory")
	}

	err := doBakeCommand(conf)
	if err != nil {
		return err
	}

	return nil
}

func doBakeCommand(conf *bakeConfiguration) error {
	ctx := context.Background()
	ctx, err := executePreSteps(ctx, conf)
	if err != nil {
		return err
	}

	ctx, err = executeBitBake(ctx, conf)
	if err != nil {
		return err
	}

	if conf.load {
		ctx, err = loadResultToRT(ctx, conf)
		if err != nil {
			return err
		}

		if conf.scan {
			ctx, err = scanResults(ctx, conf)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func executePreSteps(ctx context.Context, conf *bakeConfiguration) (context.Context, error) {
	log.Output("Running pre steps. Running directory=" + conf.runFolder)
	return ctx, nil
}

func executeBitBake(ctx context.Context, conf *bakeConfiguration) (context.Context, error) {
	log.Output("Running Bit bake. target=" + conf.target + ". This may take a long time....")
	return ctx, nil
}

func loadResultToRT(ctx context.Context, conf *bakeConfiguration) (context.Context, error) {
	log.Output("Loading the result to Artifactory")
	return ctx, nil
}

func scanResults(ctx context.Context, conf *bakeConfiguration) (context.Context, error) {
	log.Output("Scanning results")
	return ctx, errors.New("xray scanning is not yet supported")
}
