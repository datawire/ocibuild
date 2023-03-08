// Copyright (C) 2021-2022  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/datawire/ocibuild/pkg/cliutil"
	"github.com/datawire/ocibuild/pkg/dir"
	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/reproducible"
)

func init() {
	var flagPrefix dir.Prefix
	var flagChOwn dir.Ownership
	cmd := &cobra.Command{
		Use:   "dir [flags] IN_DIRNAME >OUT_LAYERFILE",
		Short: "Create a layer from a directory",
		Args:  cliutil.WrapPositionalArgs(cobra.ExactArgs(1)),
		RunE: func(_ *cobra.Command, args []string) error {
			var prefix *dir.Prefix
			if flagPrefix.DirName != "" {
				prefix = &flagPrefix
			}
			layer, err := dir.LayerFromDir(args[0], prefix, &flagChOwn, reproducible.Now())
			if err != nil {
				return err
			}

			if err := fsutil.WriteLayer(layer, os.Stdout); err != nil {
				return err
			}
			return nil
		},
	}
	// synthetic prefix
	cmd.Flags().StringVar(&flagPrefix.DirName, "prefix", "", ``+
		`Add a `+"`PREFIX`"+` to the filenames in the directory, should be forward-slash `+
		`separated and should be absolute but NOT starting with a slash.  For example, `+
		`"usr/local/bin".`)
	cmd.Flags().IntVar(&flagPrefix.UID, "prefix-uid", 0,
		`The numeric user ID of the --prefix directory`)
	cmd.Flags().StringVar(&flagPrefix.UName, "prefix-uname", "root",
		`The symbolic user name of the --prefix directory`)
	cmd.Flags().IntVar(&flagPrefix.GID, "prefix-gid", 0,
		`The numeric group ID of the --prefix directory`)
	cmd.Flags().StringVar(&flagPrefix.GName, "prefix-gname", "root",
		`The symbolic group name of the --prefix directory`)
	// actual files
	cmd.Flags().IntVar(&flagChOwn.UID, "chown-uid", -1,
		"Force the numeric user ID of read files to be `UID`; a value of <0 uses the actual UID")
	cmd.Flags().StringVar(&flagChOwn.UName, "chown-uname", "",
		"Force symbolic user name of the read files to be `uname`; an empty value uses the user name")
	cmd.Flags().IntVar(&flagChOwn.GID, "chown-gid", -1,
		"Force the numeric group ID of read files to be `GID`; use a value <0 to use the actual GID")
	cmd.Flags().StringVar(&flagChOwn.GName, "chown-gname", "root",
		"Force symbolic group name of the read files to be `gname`; an empty value uses the actual group name")

	argparserLayer.AddCommand(cmd)
}
