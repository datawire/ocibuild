package main

import (
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"

	"github.com/datawire/ocibuild/pkg/fsutil"
)

func init() {
	var argBase string
	var argTag string
	var argEntrypoint string

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
			var tag name.Reference
			if argTag != "" {
				var err error
				tag, err = name.NewTag(argTag)
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

			if argEntrypoint != "" {
				config, _ := img.ConfigFile()
				config.Config.Entrypoint = []string{argEntrypoint}

				img, err = mutate.Config(img, config.Config)

				if err != nil {
					return err
				}
			}

			if err := ociv1tarball.Write(tag, img, os.Stdout); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&argBase, "base", "", "Use `IN_IMAGEFILE` as the base of the image")
	cmd.Flags().StringVarP(&argTag, "tag", "t", "", "Tag the resulting image as `TAG`")
	cmd.Flags().StringVar(&argEntrypoint, "entrypoint", "", "Set the resulting image's `ENTRYPOINT`")

	argparserImage.AddCommand(cmd)
}
