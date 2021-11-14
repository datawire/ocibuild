package main

import (
	"os"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"

	"github.com/datawire/ocibuild/pkg/fsutil"
)

func init() {
	var argBase string
	cmd := &cobra.Command{
		Use:   "build [flags] IN_LAYERFILES... >OUT_IMAGEFILE",
		Short: "Combine layers in to a complete image",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(flags *cobra.Command, args []string) error {
			base := empty.Image
			if argBase != "" {
				var err error
				base, err = fsutil.OpenImage(argBase)
				if err != nil {
					return err
				}
			}

			layers := make([]ociv1.Layer, 0, len(args))
			for _, layerpath := range args {
				layer, err := fsutil.OpenLayer(layerpath)
				if err != nil {
					return err
				}
				layers = append(layers, layer)
			}

			img, err := mutate.AppendLayers(base, layers...)
			if err != nil {
				return err
			}
			if err := ociv1tarball.Write(nil, img, os.Stdout); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&argBase, "base", "", "Use `IN_IMAGEFILE` as the base of the image")

	argparserImage.AddCommand(cmd)
}
