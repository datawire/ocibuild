package main

import (
	"github.com/spf13/cobra"

	"github.com/datawire/layertool/pkg/squash"
)

func init() {
	cmd := &cobra.Command{
		Use:   "squash IN_LAYERFILES... >OUT_LAYERFILE",
		Short: "Squash several layers in to a single layer",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(flags *cobra.Command, args []string) error {
			_, err := squash.Squash(nil, nil)
			return err
		},
	}
	argparser.AddCommand(cmd)
}
