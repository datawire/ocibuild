package main

import (
	"io"
	"os"

	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"

	"github.com/datawire/ocibuild/pkg/squash"
)

func init() {
	cmd := &cobra.Command{
		Use:   "squash IN_LAYERFILES... >OUT_LAYERFILE",
		Short: "Squash several layers in to a single layer",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(flags *cobra.Command, args []string) error {
			layers := make([]v1.Layer, 0, len(args))
			for _, layerpath := range args {
				layer, err := OpenLayer(layerpath)
				if err != nil {
					return err
				}
				layers = append(layers, layer)
			}

			layer, err := squash.Squash(layers)
			if err != nil {
				return err
			}

			layerReader, err := layer.Uncompressed()
			if err != nil {
				return err
			}
			if _, err := io.Copy(os.Stdout, layerReader); err != nil {
				return err
			}
			return nil
		},
	}
	argparser.AddCommand(cmd)
}
