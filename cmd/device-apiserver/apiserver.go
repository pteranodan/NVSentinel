package main

import (
	"os"

	"k8s.io/component-base/cli"

	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app"
)

func main() {
	command := app.NewAPIServerCommand()
	code := cli.Run(command)
	os.Exit(code)
}
