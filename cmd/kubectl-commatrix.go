package main

import (
	"os"

	"github.com/openshift-kni/commatrix-cli-plugin/pkg/commatrix"
	"github.com/openshift-kni/commatrix/pkg/client"

	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-commatrix", pflag.ExitOnError)
	pflag.CommandLine = flags
	ioStreams := genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	cs, err := client.New()
	if err != nil {
		os.Exit(1)
	}
	root := commatrix.NewCmd(cs, ioStreams)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
