// Command ocibuild deals with manipulation of OCI/Docker images and layers as regular files.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/datawire/dlib/dlog"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/spf13/cobra"

	"github.com/datawire/ocibuild/pkg/cliutil"
)

var (
	argparser = &cobra.Command{
		Use:   "ocibuild {[flags]|SUBCOMMAND...}",
		Short: "Manipulate OCI/Docker images and layers as regular files",

		Args: cliutil.OnlySubcommands,
		RunE: cliutil.RunSubcommands,

		SilenceErrors: true, // main() will handle this after .ExecuteContext() returns
		SilenceUsage:  true, // our FlagErrorFunc will handle it
	}
	argparserImage = &cobra.Command{
		Use:   "image {[flags]|SUBCOMMAND...}",
		Short: "Manipulate complete images",

		Args: cliutil.OnlySubcommands,
		RunE: cliutil.RunSubcommands,
	}
	argparserLayer = &cobra.Command{
		Use:   "layer {[flags]|SUBCOMMAND...}",
		Short: "Manipulate individual layers for use in an image",

		Args: cliutil.OnlySubcommands,
		RunE: cliutil.RunSubcommands,
	}
)

func init() {
	argparser.SetFlagErrorFunc(cliutil.FlagErrorFunc)
	argparser.SetHelpTemplate(cliutil.HelpTemplate)
	argparser.AddCommand(argparserImage)
	argparser.AddCommand(argparserLayer)
}

func main() {
	ctx := context.Background()

	logs.Warn = dlog.StdLogger(ctx, dlog.LogLevelWarn)
	logs.Progress = dlog.StdLogger(ctx, dlog.LogLevelInfo)
	logs.Debug = dlog.StdLogger(ctx, dlog.LogLevelDebug)

	if err := argparser.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(argparser.ErrOrStderr(), "%s: error: %v\n", argparser.CommandPath(), err)
		os.Exit(1)
	}
}
