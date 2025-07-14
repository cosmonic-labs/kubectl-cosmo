package cmd

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

const CosmonicDocumentationURL = "https://cosmonic.com/docs"

func NewCmdDocs(streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: fmt.Sprintf("Open the default browser to %s", CosmonicDocumentationURL),
		RunE: func(c *cobra.Command, args []string) error {
			return browser.OpenURL(CosmonicDocumentationURL)
		},
	}

	return cmd
}
