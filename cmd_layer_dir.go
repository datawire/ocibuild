package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/datawire/ocibuild/pkg/dir"
	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/reproducible"
)

func init() {
	var flagPrefix dir.Prefix
	cmd := &cobra.Command{
		Use:   "dir [flags] IN_DIRNAME >OUT_LAYERFILE",
		Short: "Create a layer from a directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			var prefix *dir.Prefix
			if flagPrefix.DirName != "" {
				prefix = &flagPrefix
			}
			layer, err := dir.LayerFromDir(args[0], prefix, reproducible.Now())
			if err != nil {
				return err
			}

			if err := fsutil.WriteLayer(layer, os.Stdout); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagPrefix.DirName, "prefix", "",
		`Add a `+"`PREFIX`"+` to the filenames in the directory, should be forward-slash separated and should be absolute but NOT starting with a slash.  For example, "usr/local/bin".`)
	cmd.Flags().IntVar(&flagPrefix.UID, "prefix-uid", 0,
		`The numeric user ID of the --prefix directory`)
	cmd.Flags().StringVar(&flagPrefix.UName, "prefix-uname", "root",
		`The symbolic user name of the --prefix directory`)
	cmd.Flags().IntVar(&flagPrefix.GID, "prefix-gid", 0,
		`The numeric group ID of the --prefix directory`)
	cmd.Flags().StringVar(&flagPrefix.GName, "prefix-gname", "root",
		`The symbolic group name of the --prefix directory`)
	argparserLayer.AddCommand(cmd)
}
