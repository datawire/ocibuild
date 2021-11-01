package main

import (
	"os"

	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
)

func init() {
	var argBase string
	cmd := &cobra.Command{
		Use:   "image [flags] IN_LAYERFILES... >OUT_IMAGEFILE",
		Short: "Combine layers in to a complete image",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(flags *cobra.Command, args []string) error {
			base := empty.Image
			if argBase != "" {
				var err error
				base, err = OpenImage(argBase)
				if err != nil {
					return err
				}
			}

			layers := make([]v1.Layer, 0, len(args))
			for _, layerpath := range args {
				layer, err := OpenLayer(layerpath)
				if err != nil {
					return err
				}
				layers = append(layers, layer)
			}

			img, err := mutate.AppendLayers(base, layers...)
			if err != nil {
				return err
			}
			if err := tarball.Write(nil, img, os.Stdout); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&argBase, "base", "", "Use `IN_IMAGEFILE` as the base of the image")

	argparser.AddCommand(cmd)
}
