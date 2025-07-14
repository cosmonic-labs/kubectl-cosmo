package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	chartManager "github.com/cosmonic/kubectl-cosmo/pkg/internal/chartmanager"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type VersionConfig struct {
	manager     *chartManager.ChartManager
	configFlags *genericclioptions.ConfigFlags
	genericiooptions.IOStreams

	settings *cli.EnvSettings
	logger   *log.Logger
}

func NewCmdVersion(streams genericiooptions.IOStreams) *cobra.Command {
	versionCfg := &VersionConfig{configFlags: genericclioptions.NewConfigFlags(true), IOStreams: streams, logger: log.Default()}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Returns the versions of all resources installed for Cosmonic Control",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := versionCfg.Initialize(cmd, args); err != nil {
				return err
			}
			if err := versionCfg.Validate(); err != nil {
				return err
			}

			if err := versionCfg.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// Initialize configures the chart manager
func (verCfg *VersionConfig) Initialize(cmd *cobra.Command, args []string) error {
	verCfg.settings = cli.New()
	helmDriver := os.Getenv("HELM_DRIVER")
	manager, err := chartManager.New(verCfg.IOStreams, helmDriver, log.Default())
	if err != nil {
		return err
	}
	verCfg.manager = manager

	return err
}

// Valdiate checks the configuration
func (verCfg *VersionConfig) Validate() error {
	return nil
}

// Default nexus command will display the installed version and the latest available repo version for nexus and hostgroups
func (verCfg *VersionConfig) Run() error {
	ctx := context.Background()

	logTo, err := verCfg.logChartVersion(ctx, "nexus control", controlChartName, controlChartName)
	if err != nil {
		return err
	}
	fmt.Println(logTo)

	logTo, err = verCfg.logChartVersion(ctx, "hostgroup", hostgroupInstalledChartName, hostgroupRepoChartName)
	if err != nil {
		return err
	}
	fmt.Println(logTo)
	return nil
}

func (verCfg *VersionConfig) logChartVersion(ctx context.Context, componentName string, installedChartName string, repoChartName string) (string, error) {
	installedVer, err := verCfg.manager.GetInstalledChartVersion(installedChartName)
	if err != nil {
		return "", err
	}
	repoVer, err := verCfg.manager.GetRepoChartVersion(ctx, repoChartName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("component %s: installed [%s], available [%s]", componentName, installedVer, repoVer), nil
}
