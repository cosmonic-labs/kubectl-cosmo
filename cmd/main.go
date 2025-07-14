package main

import (
	"os"

	"github.com/cosmonic/kubectl-cosmo/pkg/cmd"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-cosmo", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewCmdCosmo(genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
