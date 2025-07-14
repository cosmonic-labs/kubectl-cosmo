package chartManager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/tools/clientcmd/api"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	cosmonicNamespace     = "cosmonic-system"
	cosmonicChartRegistry = "oci://ghcr.io/cosmonic"
)

type ChartManager struct {
	configFlags           *genericclioptions.ConfigFlags
	resultingContext      *api.Context
	resultingContextName  string
	args                  []string
	rawConfig             api.Config
	userSpecifiedCluster  string
	userSpecifiedContext  string
	userSpecifiedAuthInfo string
	genericiooptions.IOStreams

	settings   *cli.EnvSettings
	helmAction *action.Configuration
	logger     *log.Logger
}

// pass  os.Getenv("HELM_DRIVER") for helmDriver
func New(streams genericiooptions.IOStreams, helmDriver string, logger *log.Logger) (*ChartManager, error) {
	manager := &ChartManager{configFlags: genericclioptions.NewConfigFlags(true), IOStreams: streams, logger: logger}

	// initialize
	manager.settings = cli.New()
	var err error

	manager.helmAction = new(action.Configuration)
	if err := manager.helmAction.Init(
		manager.settings.RESTClientGetter(),
		cosmonicNamespace,
		helmDriver,
		logger.Printf); err != nil {
		return nil, err
	}

	return manager, err
}

func newRegistryClient(settings *cli.EnvSettings, plainHTTP bool) (*registry.Client, error) {
	opts := []registry.ClientOption{
		registry.ClientOptDebug(settings.Debug),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stderr),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
	}
	if plainHTTP {
		opts = append(opts, registry.ClientOptPlainHTTP())
	}

	// Create a new registry client
	registryClient, err := registry.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}

func (manager *ChartManager) GetInstalledChartVersion(chartName string) (string, error) {
	listClient := action.NewList(manager.helmAction)
	// Only list deployed
	//listClient.Deployed = true
	listClient.All = true
	listClient.Filter = chartName
	listClient.SetStateMask()

	results, err := listClient.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run list action: %w", err)
	}

	for _, rel := range results {
		return rel.Chart.AppVersion(), nil
	}

	return "", errors.New("chart not found")
}

func (manager *ChartManager) GetRepoChartVersion(ctx context.Context, chartName string) (string, error) {

	repo, err := remote.NewRepository(fmt.Sprintf("ghcr.io/cosmonic/%s", chartName))
	if err != nil {
		return "", err
	}

	var result []string
	var tagRetriever = func(tags []string) error {
		result = tags
		return nil
	}

	err = repo.Tags(ctx, "", tagRetriever)
	if err != nil {
		return "", err
	}

	if len(result) == 0 {
		return "", errors.New("repository tag not found")
	}

	// should be least to greatest, but sorting just in case
	sort.Strings(result)

	return result[len(result)-1], nil
}

func (manager *ChartManager) CheckDependencies(chart *chart.Chart, registryClient *registry.Client, chartPath string, keyRing string) error {
	providers := getter.All(manager.settings)

	// check for dependencies to download
	if dependencies := chart.Metadata.Dependencies; dependencies != nil {
		if err := action.CheckDependencies(chart, dependencies); err != nil {
			/*if !installClient.DependencyUpdate {
				return fmt.Errorf("failed checking dependencies, error %v", err)
			}*/

			manager := &downloader.Manager{
				Out:              manager.logger.Writer(),
				ChartPath:        chartPath,
				Keyring:          keyRing,
				SkipUpdate:       false,
				Getters:          providers,
				RepositoryConfig: manager.settings.RegistryConfig,
				RepositoryCache:  manager.settings.RepositoryCache,
				Debug:            manager.settings.Debug,
				RegistryClient:   registryClient,
			}

			if err := manager.Update(); err != nil {
				return err
			}

			// reload chart
			chart, err = loader.Load(chartPath)

			if err != nil {
				return fmt.Errorf("failed to reload chart after dependency update, error %v", err)
			}
		}
	}

	return nil
}

func (manager *ChartManager) chartRegistryName(chartName string) string {
	return fmt.Sprintf("%s/%s", cosmonicChartRegistry, chartName)
}

func (manager *ChartManager) Install(ctx context.Context, chartName string) error {
	// check if chart is already installed
	if ver, err := manager.GetInstalledChartVersion(chartName); err == nil && ver != "" {
		return errors.New("chart is already installed")
	}

	releaseVersion, err := manager.GetRepoChartVersion(ctx, chartName)
	if err != nil {
		return err
	}

	installClient := action.NewInstall(manager.helmAction)
	installClient.DryRunOption = "none" // set this if flag passed for export
	installClient.ReleaseName = "cosmonic-control"
	installClient.Version = releaseVersion

	registryClient, err := newRegistryClient(manager.settings, false)
	if err != nil {
		return err
	}
	installClient.SetRegistryClient(registryClient)

	registryName := manager.chartRegistryName(chartName)
	chartPath, err := installClient.ChartPathOptions.LocateChart(registryName, manager.settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}

	if err := manager.CheckDependencies(chart, installClient.GetRegistryClient(), chartPath,
		installClient.ChartPathOptions.Keyring); err != nil {
		return err
	}

	// TODO need the values to pass in
	var releaseValues map[string]interface{}

	_, err = installClient.RunWithContext(ctx, chart, releaseValues)

	if err != nil {
		return err
	}
	return nil
}

func (manager *ChartManager) UnInstall(chartName string) error {
	// ensure that the --force flag is passed

	// uninstall helm chart
	uninstallClient := action.NewUninstall(manager.helmAction)
	uninstallClient.DeletionPropagation = "foreground"

	result, err := uninstallClient.Run(chartName)
	if err != nil {
		return err
	}

	if result != nil {
		return fmt.Errorf("uninstall returned %v", result)
	}

	// TODO remove cosmonic-system namespace
	return nil
}

func (manager *ChartManager) Update(chartName string) error {
	ctx := context.Background()

	// get latest version from oci registry
	repoVersion, err := manager.GetRepoChartVersion(ctx, chartName)
	if err != nil {
		return err
	}
	// get version of helm chart installed
	installedVersion, err := manager.GetInstalledChartVersion(chartName)
	if err != nil {
		return err
	}

	// if no update available then return
	if repoVersion == installedVersion || installedVersion > repoVersion {
		return fmt.Errorf("chart is already at the latest version %s", repoVersion)
	}

	// update
	upgradeClient := action.NewUpgrade(manager.helmAction)
	upgradeClient.Namespace = cosmonicNamespace
	// set dry-run if flag is set for export
	upgradeClient.DryRunOption = "none"
	upgradeClient.Version = repoVersion

	registryClient, err := newRegistryClient(manager.settings, false)
	if err != nil {
		return err
	}

	upgradeClient.SetRegistryClient(registryClient)

	registryName := manager.chartRegistryName(chartName)
	chartPath, err := upgradeClient.ChartPathOptions.LocateChart(registryName, manager.settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}

	if err := manager.CheckDependencies(chart, registryClient, chartPath,
		upgradeClient.ChartPathOptions.Keyring); err != nil {
		return err
	}

	// TODO need the values to pass in
	var releaseValues map[string]interface{}
	_, err = upgradeClient.RunWithContext(ctx, chartName, chart, releaseValues)

	return err
}
