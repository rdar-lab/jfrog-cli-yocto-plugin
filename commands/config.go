package commands

import (
	"github.com/jfrog/jfrog-cli-core/common/commands"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
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
			return configCmd()
		},
	}
}

func configCmd() error {
	artConfigCommand := commands.NewConfigCommand()
	artConfigCommand.SetInteractive(true)
	return artConfigCommand.Run()
}
