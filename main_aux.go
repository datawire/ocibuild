//go:build aux

package main

import (
	"os"

	"github.com/datawire/ocibuild/pkg/cliutil"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func init() {
	// completion
	argparser.CompletionOptions.DisableDefaultCmd = false
	argparser.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		completionCmd, _, _ := cmd.Root().Find([]string{"completion"})
		completionCmd.Hidden = true
	}

	// man
	argparser.AddCommand(&cobra.Command{
		Hidden: true,
		Use:    "man OUT_DIRECTORY",
		Short:  "Generate man pages",
		Args:   cliutil.WrapPositionalArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
			if err := os.MkdirAll(dir, 0777); err != nil {
				return err
			}
			root := cmd.Root()
			root.DisableAutoGenTag = true
			header := &doc.GenManHeader{
				Source: "Ambassador Labs",
				Manual: root.Name(),
			}
			if err := doc.GenManTree(root, header, dir); err != nil {
				return err
			}
			return nil
		},
	})

	// mddoc
	argparser.AddCommand(&cobra.Command{
		Hidden: true,
		Use:    "mddoc OUT_DIRECTORY",
		Short:  "Generate markdown documentation",
		Args:   cliutil.WrapPositionalArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
			if err := os.MkdirAll(dir, 0777); err != nil {
				return err
			}
			root := cmd.Root()
			root.DisableAutoGenTag = true
			if err := doc.GenMarkdownTree(root, dir); err != nil {
				return err
			}
			return nil
		},
	})
}
