// Copyright (C) 2021-2022  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"reflect"

	"github.com/google/go-containerregistry/pkg/name"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/datawire/ocibuild/pkg/cliutil"
	"github.com/datawire/ocibuild/pkg/fsutil"
)

type configFlags struct {
	// https://github.com/opencontainers/image-spec/blob/main/config.md

	// User
	// ExposedPorts
	// Env
	envClear  bool
	envAppend []string
	// Entrypoint
	entrypoint []string
	// Cmd
	cmd []string
	// Volumes
	// WorkingDir
	workingDir string
	// Labels
	// StopSignal
	// Memory
	// MemorySwap
	// CpuShares
	// Healthcheck
}

func (flags *configFlags) AddFlagsTo(prefix string, flagset *pflag.FlagSet) {
	// https://github.com/opencontainers/image-spec/blob/main/config.md

	// User
	// ExposedPorts
	// Env
	flagset.BoolVarP(&flags.envClear, prefix+"Env.clear", "E", false,
		"Discard any environment variables set in the base image's config")
	flagset.StringArrayVarP(&flags.envAppend, prefix+"Env.append", "e", nil,
		"Append `KEY=VALUE` in the resulting image's environment")
	// Entrypoint
	flagset.StringArrayVar(&flags.entrypoint, prefix+"Entrypoint", nil,
		"Set the resulting image's `entrypoint`")
	// Cmd
	flagset.StringArrayVarP(&flags.cmd, prefix+"Cmd", "c", nil,
		"Set the resulting image's `command`")
	// Volumes
	// WorkingDir
	flagset.StringVarP(&flags.workingDir, prefix+"WorkingDir", "w", "",
		"Set the resulting image's `working-directory`")
	// Labels
	// StopSignal
	// Memory
	// MemorySwap
	// CpuShares
	// Healthcheck
}

func (flags configFlags) IsZero() bool {
	// Because it contains slices, we con't just use `== configFlags{}` because Go won't let you
	// compare slices.  We could manually check each field, but this is easier.
	return reflect.ValueOf(flags).IsZero()
}

func (flags configFlags) ApplyTo(config *ociv1.Config) {
	// https://github.com/opencontainers/image-spec/blob/main/config.md

	// User

	// ExposedPorts

	// Env
	if flags.envClear {
		config.Env = nil
	}
	config.Env = append(config.Env, flags.envAppend...)

	// Entrypoint
	if flags.entrypoint != nil {
		config.Entrypoint = flags.entrypoint
	}

	// Cmd
	if flags.cmd != nil {
		config.Cmd = flags.cmd
	}

	// Volumes

	// WorkingDir
	if flags.workingDir != "" {
		config.WorkingDir = flags.workingDir
	}

	// Labels

	// StopSignal

	// Memory

	// MemorySwap

	// CpuShares

	// Healthcheck
}

func init() {
	var flags struct {
		base   string
		tag    string
		config configFlags
	}
	cmd := &cobra.Command{
		Use:   "build [flags] IN_LAYERFILES... >OUT_IMAGEFILE",
		Short: "Combine layers in to a complete image",
		Args:  cliutil.WrapPositionalArgs(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			base := empty.Image
			if flags.base != "" {
				var err error
				base, err = fsutil.OpenImage(flags.base)
				if err != nil {
					return err
				}
			}
			var tag name.Reference
			if flags.tag != "" {
				var err error
				tag, err = name.NewTag(flags.tag)
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

			if !flags.config.IsZero() {
				configFile, _ := img.ConfigFile()

				flags.config.ApplyTo(&configFile.Config)

				img, err = mutate.Config(img, configFile.Config)
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

	cmd.Flags().StringVar(&flags.base, "base", "", "Use `IN_IMAGEFILE` as the base of the image")
	cmd.Flags().StringVarP(&flags.tag, "tag", "t", "", "Tag the resulting image as `TAG`")
	flags.config.AddFlagsTo("config.", cmd.Flags())

	argparserImage.AddCommand(cmd)
}
