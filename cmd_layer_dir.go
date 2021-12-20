package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/datawire/ocibuild/pkg/dir"
	"github.com/datawire/ocibuild/pkg/fsutil"
)

func init() {
	var flagPrefix string
	cmd := &cobra.Command{
		Use:   "dir [flags] IN_DIRNAME >OUT_LAYERFILE",
		Short: "Create a layer from a directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(flags *cobra.Command, args []string) error {
			layer, err := dir.LayerFromDir(args[0], flagPrefix)
			if err != nil {
				return err
			}

			if err := fsutil.WriteLayer(layer, os.Stdout); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagPrefix, "add-prefix", "",
		`Add a prefix to the filenames in the directory, should be forward-slash separated and should be absolute but NOT  starting with a slash.  For example, "usr/local/bin".`)
	argparserLayer.AddCommand(cmd)
}
