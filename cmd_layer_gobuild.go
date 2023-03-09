// Copyright (C) 2021-2022  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"io"
	"os"

	"github.com/datawire/dlib/dlog"
	"github.com/spf13/cobra"

	"github.com/datawire/ocibuild/pkg/cliutil"
	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/gobuild"
	"github.com/datawire/ocibuild/pkg/reproducible"
)

func init() {
	var outputFilename string
	cmd := &cobra.Command{
		Use:   "gobuild [flags] PACKAGES... >OUT_LAYERFILE",
		Short: "Create a layer of Go binaries",
		Long: "Works more or less like `go build`.  Passes through env-vars (except for " +
			"GOOS and GOARCH; naturally those need to be set to reflect the target " +
			"layer).  Use GOFLAGS to pass in extra flags." +
			"\n\n" +
			"When directing stdout to a file, the timestamps within the resulting " +
			"layer file will be the current time (clamped by SOURCE_DATE_EPOCH).  " +
			"If SOURCE_DATE_EPOCH is not set, this may result in unnecessary layer " +
			"changes; to prevent this, use the --output=FILENAME flag, which avoids " +
			"updating the layer file if the only changes are timestamps.",
		Args: cliutil.WrapPositionalArgs(cobra.MinimumNArgs(1)),
		RunE: func(flags *cobra.Command, args []string) (err error) {
			maybeSetErr := func(_err error) {
				if _err != nil && err == nil {
					err = _err
				}
			}

			layer, err := gobuild.LayerFromGo(flags.Context(), reproducible.Now(), args)
			if err != nil {
				return err
			}

			outputWriter := io.Writer(os.Stdout)
			if outputFilename != "" {
				// Check if the layer changed.
				if oldLayer, err := fsutil.OpenLayer(outputFilename); err != nil {
					if !errors.Is(err, os.ErrNotExist) {
						return err
					}
				} else {
					equal, err := fsutil.LayersEqualExceptTimestamps(layer, oldLayer)
					if err != nil {
						return err
					}
					if equal {
						dlog.Infoln(flags.Context(), "Layer didn't change")
						return nil
					}
				}

				// Open the file for writing.
				if err := os.Remove(outputFilename); err != nil && !errors.Is(err, os.ErrNotExist) {
					return err
				}
				outputFile, err := os.OpenFile(outputFilename, os.O_CREATE|os.O_WRONLY, 0o666)
				if err != nil {
					return err
				}
				defer func() {
					maybeSetErr(outputFile.Close())
				}()
				outputWriter = outputFile
			}

			if err := fsutil.WriteLayer(layer, outputWriter); err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&outputFilename, "output", "o", "", ""+
		"Write the layer to `FILENAME`, rather than stdout.  "+
		"Using this rather than directing stdout to a file may prevent unnescessary timestamp bumps.")

	argparserLayer.AddCommand(cmd)
}
