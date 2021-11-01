// Command layertool deals with manipulation of Docker layer files.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/datawire/dlib/dlog"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/spf13/cobra"

	"github.com/datawire/layertool/pkg/cliutil"
)

var argparser = &cobra.Command{
	Use:   "layertool {[flags]|SUBCOMMAND...}",
	Short: "Manipulate Docker layers as files",

	Args: cliutil.OnlySubcommands,
	RunE: cliutil.RunSubcommands,

	SilenceErrors: true, // main() will handle this after .ExecuteContext() returns
	SilenceUsage:  true, // our FlagErrorFunc will handle it
}

func init() {
	argparser.SetFlagErrorFunc(cliutil.FlagErrorFunc)
	argparser.SetHelpTemplate(cliutil.HelpTemplate)
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
