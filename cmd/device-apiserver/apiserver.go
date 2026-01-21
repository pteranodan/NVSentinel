package main

import (
	"os"

	"k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"

	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app"
)

func main() {
	ctx := server.SetupSignalContext()
	command := app.NewAPIServerCommand()
	command.SetContext(ctx)
	code := cli.Run(command)
	os.Exit(code)
}
