package main

import (
	"commatrix-cli-plugin/pkg/cmd"
	"os"

	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-commatrix", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewCmdCommatrix(genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
