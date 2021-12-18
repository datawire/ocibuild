package main

import (
	"github.com/spf13/cobra"
	"os"

	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/gobuild"
)

func init() {
	cmd := &cobra.Command{
		Use:   "gobuild [flags] PACKAGES... >OUT_LAYERFILE",
		Short: "Create a layer from a directory",
		Long: "Works more or less like `go build`.  Passes through env-vars (except for " +
			"GOOS and GOARCH; naturally those need to be set to reflect the target " +
			"layer).  Use GOFLAGS to pass in extra flags.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(flags *cobra.Command, args []string) error {
			layer, err := gobuild.LayerFromGo(flags.Context(), args)
			if err != nil {
				return err
			}

			if err := fsutil.WriteLayer(layer, os.Stdout); err != nil {
				return err
			}
			return nil
		},
	}
	argparserLayer.AddCommand(cmd)
}
