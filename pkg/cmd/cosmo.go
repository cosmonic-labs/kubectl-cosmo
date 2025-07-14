package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func NewCmdCosmo(streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cosmo [command] [flags]",
		Short: "Interact with Cosmonic Control",
	}

	// add commands
	cmd.AddCommand(NewCmdNexus(streams))
	cmd.AddCommand(NewCmdHostgroup(streams))
	cmd.AddCommand(NewCmdConsole(streams))
	cmd.AddCommand(NewCmdDocs(streams))
	cmd.AddCommand(NewCmdVersion(streams))
	cmd.AddCommand(NewCmdLicense(streams))
	return cmd
}
