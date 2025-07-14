package cmd

import (
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

const (
	cosmonicLicenseURL = "https://cosmonic.com/trial"
)

func NewCmdLicense(streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "license [flags]",
		Short: "To obtain a license, visit cosmonic.com and sign up for a free trial key",
		RunE: func(c *cobra.Command, args []string) error {
			return browser.OpenURL(cosmonicLicenseURL)
		},
	}

	return cmd
}
