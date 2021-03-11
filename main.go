package main

import (
	"github.com/jfrog/jfrog-cli-core/plugins"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/rdar-lab/jfrog-cli-yocto-plugin/commands"
)

func main() {
	plugins.PluginMain(getApp())
}

func getApp() components.App {
	app := components.App{}
	app.Name = "jfrog-yocto"
	app.Description = "Jfrog Yocto Build CLI plugin"
	app.Version = "v0.1.1"
	app.Commands = getCommands()
	return app
}

func getCommands() []components.Command {
	return []components.Command{
		commands.GetBakeCommand(),
		commands.GetConfigCommand(),
	}
}
