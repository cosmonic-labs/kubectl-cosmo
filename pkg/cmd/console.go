package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"k8s.io/client-go/tools/clientcmd"
)

const (
	// startPort and endPort are ranges to find the first avaiable port open to port-forward to the console UI
	startPort = 8080
	endPort   = 8280

	cosmonicNamespace = "cosmonic-system"
	consoleDeployment = "console"
)

var (
	errNoContext = fmt.Errorf("no context is currently set, use %q to select a new one", "kubectl config use-context <context>")
)

type ConsoleConfig struct {
	configFlags           *genericclioptions.ConfigFlags
	resultingContext      *api.Context
	resultingContextName  string
	args                  []string
	rawConfig             api.Config
	userSpecifiedCluster  string
	userSpecifiedContext  string
	userSpecifiedAuthInfo string
	genericiooptions.IOStreams
}

func NewCmdConsole(streams genericiooptions.IOStreams) *cobra.Command {
	console := &ConsoleConfig{configFlags: genericclioptions.NewConfigFlags(true), IOStreams: streams}

	cmd := &cobra.Command{
		Use:   "console [command] [flags]",
		Short: "launch the Cosmonic console",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := console.Complete(cmd, args); err != nil {
				return err
			}
			if err := console.Validate(); err != nil {
				return err
			}

			if err := console.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// Complete sets the k8s context etc.
func (c *ConsoleConfig) Complete(cmd *cobra.Command, args []string) error {
	c.args = args

	var err error
	c.rawConfig, err = c.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}

	currentContext, exists := c.rawConfig.Contexts[c.rawConfig.CurrentContext]
	if !exists {
		return errNoContext
	}

	c.resultingContext = api.NewContext()
	c.resultingContext.Cluster = currentContext.Cluster
	c.resultingContext.AuthInfo = currentContext.AuthInfo

	// if a target context is explicitly provided by the user,
	// use that as our reference for the final, resulting context
	if len(c.userSpecifiedContext) > 0 {
		c.resultingContextName = c.userSpecifiedContext
		if userCtx, exists := c.rawConfig.Contexts[c.userSpecifiedContext]; exists {
			c.resultingContext = userCtx.DeepCopy()
		}
	}

	if len(c.userSpecifiedCluster) > 0 {
		c.resultingContext.Cluster = c.userSpecifiedCluster
	}
	if len(c.userSpecifiedAuthInfo) > 0 {
		c.resultingContext.AuthInfo = c.userSpecifiedAuthInfo
	}

	// generate a unique context name based on its new values if
	// user did not explicitly request a context by name
	if len(c.userSpecifiedContext) == 0 {
		c.resultingContextName = generateContextName(c.resultingContext)
	}

	return nil
}

func generateContextName(fromContext *api.Context) string {
	name := fromContext.Namespace
	if len(fromContext.Cluster) > 0 {
		name = fmt.Sprintf("%s/%s", name, fromContext.Cluster)
	}
	if len(fromContext.AuthInfo) > 0 {
		cleanAuthInfo := strings.Split(fromContext.AuthInfo, "/")[0]
		name = fmt.Sprintf("%s/%s", name, cleanAuthInfo)
	}

	return name
}

// Valdiate checks the configuration of the cluster to make sure that the Console pod is running..
func (c *ConsoleConfig) Validate() error {
	if len(c.rawConfig.CurrentContext) == 0 {
		return errNoContext
	}

	// verify that the console deployment is running
	hasConsole, err := c.verifyConsoleDeployment()

	if err != nil {
		return err
	}

	if hasConsole {
		return nil
	}

	return fmt.Errorf("error with retrieving console deployment status")
}

// Run will create the Forward proxy and launch the URL to the port
func (c *ConsoleConfig) Run() error {
	ctx := context.Background()

	// get localPort available
	localPort, err := c.findLocalPort()
	if err != nil {
		return err
	}

	readyCh := make(chan struct{})
	stopCh := make(chan struct{}, 1)
	errChan := make(chan error)

	go func() {
		errChan <- c.PortForward(ctx, readyCh, stopCh, errChan, localPort)
	}()

	select {
	case <-readyCh:
		// Display message to "press enter" to open console, Control+C when finished
		fmt.Printf("Press enter to connect to the console at http://localhost:%d\nCtrl+C when finished\n", localPort)
	case err = <-errChan:
		fmt.Printf("Error with starting port forwarding, error %v", err)
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	// Wait until the newline character is read
	reader.ReadBytes('\n')
	browser.OpenURL(fmt.Sprintf("http://localhost:%d", localPort))

	// pause until user breaks out
	select {}
}

func (c *ConsoleConfig) PortForward(ctx context.Context, readyCh chan struct{}, stopCh chan struct{}, errChan chan error, localPort int) error {
	if ctx.Err() != nil {
		return fmt.Errorf("context is already closed")
	}
	client, config, err := c.k8sClient()
	if err != nil {
		return err
	}

	consoleDeploy, err := client.AppsV1().Deployments(cosmonicNamespace).Get(ctx, consoleDeployment, v1.GetOptions{})
	if err != nil {
		return err
	}

	podList, err := client.CoreV1().Pods(cosmonicNamespace).List(ctx, v1.ListOptions{
		LabelSelector: labels.Set(consoleDeploy.Spec.Selector.MatchLabels).AsSelector().String(),
	})

	if err != nil {
		return err
	}

	if len(podList.Items) == 0 {
		return errors.New("no pod found for console deployment")
	}

	hostIP := strings.TrimLeft(config.Host, "htps:/")

	log.Printf("hostIP is %s", hostIP)

	transport, upgrader, err := spdy.RoundTripperFor(config)

	if err != nil {
		return err
	}

	podPath := client.CoreV1().RESTClient().Post().Resource("pods").Namespace(cosmonicNamespace).Name(podList.Items[0].Name).SubResource("portforward")

	dialer := spdy.NewDialer(
		upgrader,
		&http.Client{Transport: transport},
		http.MethodPost,
		podPath.URL(),
	)

	iostream := genericclioptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr}

	portForwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, 8080)}, stopCh, readyCh, iostream.Out, iostream.ErrOut)
	if err != nil {
		return err
	}

	return portForwarder.ForwardPorts()
}

func (c *ConsoleConfig) findLocalPort() (int, error) {

	for portNum := startPort; portNum <= endPort; portNum++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
		if err == nil {
			listener.Close()
			return portNum, err
		}
	}
	return 0, errors.New("local port for port-forwarding not found")
}

func (c *ConsoleConfig) k8sClient() (*kubernetes.Clientset, *rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return client, config, nil
}

func (c *ConsoleConfig) verifyConsoleDeployment() (bool, error) {

	ctx := context.Background()
	client, _, err := c.k8sClient()
	if err != nil {
		return false, err
	}
	consoleDeployment, err := client.AppsV1().Deployments(cosmonicNamespace).Get(ctx, consoleDeployment, v1.GetOptions{})

	if err != nil {
		return false, err
	}

	if consoleDeployment.Status.ReadyReplicas > 0 {
		return true, nil
	}
	return false, nil
}
