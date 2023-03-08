// Copyright (C) 2021-2022  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"

	"github.com/datawire/ocibuild/pkg/cliutil"
	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/squash"
)

func init() {
	cmd := &cobra.Command{
		Use:   "squash [flags] IN_LAYERFILES... >OUT_LAYERFILE",
		Short: "Squash several layers in to a single layer",
		Args:  cliutil.WrapPositionalArgs(cobra.MinimumNArgs(2)),
		RunE: func(flags *cobra.Command, args []string) error {
			layers := make([]ociv1.Layer, 0, len(args))
			for _, layerpath := range args {
				layer, err := fsutil.OpenLayer(layerpath)
				if err != nil {
					return err
				}
				layers = append(layers, layer)
			}

			layer, err := squash.Squash(layers)
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
