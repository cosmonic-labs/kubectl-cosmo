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
	hostgroupRepoChartName      = "cosmonic-control-hostgroup"
	hostgroupInstalledChartName = "hostgroup"
)

type HostgroupConfig struct {
	manager        *chartManager.ChartManager
	configFlags    *genericclioptions.ConfigFlags
	forceUninstall *bool
	genericiooptions.IOStreams

	settings *cli.EnvSettings
	logger   *log.Logger
}

func NewCmdHostgroup(streams genericiooptions.IOStreams) *cobra.Command {
	hostGroup := &HostgroupConfig{configFlags: genericclioptions.NewConfigFlags(true), IOStreams: streams, logger: log.Default()}

	cmd := &cobra.Command{
		Use:   "hostgroup [command] [flags]",
		Short: "Manage hostgroups within the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := hostGroup.Initialize(cmd, args); err != nil {
				return err
			}
			if err := hostGroup.Validate(); err != nil {
				return err
			}

			if err := hostGroup.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	// install command
	var installCmd = &cobra.Command{
		Use:   "install",
		Short: "installs a new hostgroup instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := hostGroup.Initialize(cmd, args); err != nil {
				return err
			}

			return hostGroup.manager.Install(context.TODO(), hostgroupRepoChartName)
		},
	}

	// update command
	var updateCmd = &cobra.Command{
		Use:   "update",
		Short: "updates the hostgroup instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := hostGroup.Initialize(cmd, args); err != nil {
				return err
			}

			return hostGroup.manager.Update(hostgroupInstalledChartName)
		},
	}

	// uninstall command
	var uninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "uninstall all hostgroup instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := hostGroup.Initialize(cmd, args); err != nil {
				return err
			}

			return hostGroup.manager.UnInstall(hostgroupInstalledChartName)
		},
	}
	hostGroup.forceUninstall = uninstallCmd.Flags().Bool("force", false, "must specify force to uninstall the nexus control plane")
	uninstallCmd.MarkFlagRequired("force")

	// add subcommands
	cmd.AddCommand(installCmd)
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(uninstallCmd)

	return cmd
}

// Initialize configures the chart manager
func (hostGroup *HostgroupConfig) Initialize(cmd *cobra.Command, args []string) error {
	hostGroup.settings = cli.New()
	helmDriver := os.Getenv("HELM_DRIVER")
	manager, err := chartManager.New(hostGroup.IOStreams, helmDriver, log.Default())
	if err != nil {
		return err
	}
	hostGroup.manager = manager

	return err
}

// Valdiate checks the configuration
func (hostGroup *HostgroupConfig) Validate() error {
	return nil
}

// Run will display the installed version and available repo versions
func (hostGroup *HostgroupConfig) Run() error {
	ctx := context.Background()

	installedVer, err := hostGroup.manager.GetInstalledChartVersion(controlChartName)
	if err != nil {
		return err
	}

	repoVer, err := hostGroup.manager.GetRepoChartVersion(ctx, controlChartName)
	if err != nil {
		return err
	}

	hostGroup.logger.Printf("nexus installed version [%s], repo version [%s]\n", installedVer, repoVer)

	return nil
}
