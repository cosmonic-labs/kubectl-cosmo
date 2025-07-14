package cmd

import (
	"context"
	"log"
	"os"

	chartManager "github.com/cosmonic/kubectl-cosmo/pkg/internal/chartmanager"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

const (
	controlChartName = "cosmonic-control"
)

type NexusConfig struct {
	manager        *chartManager.ChartManager
	configFlags    *genericclioptions.ConfigFlags
	forceUninstall *bool
	genericiooptions.IOStreams

	settings *cli.EnvSettings
	logger   *log.Logger
}

func NewCmdNexus(streams genericiooptions.IOStreams) *cobra.Command {
	nexus := &NexusConfig{configFlags: genericclioptions.NewConfigFlags(true), IOStreams: streams, logger: log.Default()}
	cmd := &cobra.Command{
		Use:   "nexus [command] [flags]",
		Short: "Manage the Nexus Cosmonic control-plane",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := nexus.Initialize(cmd, args); err != nil {
				return err
			}
			if err := nexus.Validate(); err != nil {
				return err
			}

			if err := nexus.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	// install command
	var installCmd = &cobra.Command{
		Use:   "install",
		Short: "installs nexus helm chart",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := nexus.Initialize(cmd, args); err != nil {
				return err
			}

			return nexus.manager.Install(context.TODO(), controlChartName)
		},
	}

	// update command
	var updateCmd = &cobra.Command{
		Use:   "update",
		Short: "update nexus helm chart",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := nexus.Initialize(cmd, args); err != nil {
				return err
			}

			return nexus.manager.Update(controlChartName)
		},
	}

	// uninstall command
	var uninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "uninstall nexus helm chart",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := nexus.Initialize(cmd, args); err != nil {
				return err
			}

			return nexus.manager.UnInstall(controlChartName)
		},
	}
	nexus.forceUninstall = uninstallCmd.Flags().Bool("force", false, "must specify force to uninstall the nexus control plane")
	uninstallCmd.MarkFlagRequired("force")

	// add subcommands
	cmd.AddCommand(installCmd)
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(uninstallCmd)

	return cmd
}

// Initialize configures the chart manager
func (nexus *NexusConfig) Initialize(cmd *cobra.Command, args []string) error {
	nexus.settings = cli.New()
	helmDriver := os.Getenv("HELM_DRIVER")
	manager, err := chartManager.New(nexus.IOStreams, helmDriver, log.Default())
	if err != nil {
		return err
	}
	nexus.manager = manager

	return err
}

// Valdiate checks the configuration
func (nexus *NexusConfig) Validate() error {
	return nil
}

// Default nexus command will display the installed version and the latest available repo version
func (nexus *NexusConfig) Run() error {
	ctx := context.Background()

	installedVer, err := nexus.manager.GetInstalledChartVersion(controlChartName)
	if err != nil {
		return err
	}

	repoVer, err := nexus.manager.GetRepoChartVersion(ctx, controlChartName)
	if err != nil {
		return err
	}

	nexus.logger.Printf("nexus installed version [%s], repo version [%s]\n", installedVer, repoVer)

	return nil
}
