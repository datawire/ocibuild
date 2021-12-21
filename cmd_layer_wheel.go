package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/python"
	"github.com/datawire/ocibuild/pkg/python/pypa/bdist"
	"github.com/datawire/ocibuild/pkg/python/pypa/entry_points"
	"github.com/datawire/ocibuild/pkg/python/pypa/recording_installs"
)

func init() {
	var platFile string
	cmd := &cobra.Command{
		Use:   "wheel [flags] IN_WHEELFILE.whl >OUT_LAYERFILE",
		Short: "Turn a Python wheel in to a layer",
		Long: "Given a Python wheel file, transform it in to a layer." +
			"\n\n" +
			"In order to transform the wheel in to a layer, ocibuild needs to know a " +
			"few things about the target environment.  You must supply this to " +
			"ocibuild using the --platform-file flag, pointing it at a YAML file that " +
			"is as follows:" +
			"\n\n" +
			"    # file locations\n" +
			"    ConsoleShebang: /usr/bin/python3.9\n" +
			"    GraphicalShebang: /usr/bin/python3.9\n" +
			"    # You can obtain the scheme paths for a running Python instance with\n" +
			"    #     import json\n" +
			"    #     from pip._internal.locations import get_scheme\n" +
			"    #     scheme=get_scheme(')\n" +
			"    #     print(json.dumps({slot: getattr(scheme, slot) for slot in scheme.__slots__}))\n" +
			"    Scheme:\n" +
			"      purelib: /usr/lib/python3.9/site-packages\n" +
			"      platlib: /usr/lib/python3.9/site-packages\n" +
			"      headers: /usr/include/site/python3.9/\n" +
			"      scripts: /usr/bin\n" +
			"      data: /usr\n" +
			"\n" +
			"    # user account\n" +
			"    UID: 0\n" +
			"    GID: 0\n" +
			"    UName: root\n" +
			"    GName: root\n" +
			"\n" +
			"    # command to run on the host (not target) system to generate .pyc\n" +
			"    # files.  The Python version number must match the target Python's\n" +
			"    # version number rather precisely; or rather their\n" +
			"    # `importlib.util.MAGIC_NUMBER` values must match.\n" +
			"    PyCompile: ['python3.9', '-m', 'compileall']\n" +
			"\n" +
			"LIMITATION: It is 'TODO' to create an 'ocibuild python WHATEVER' command " +
			"that can inspect an image's Python installation and emit the appropriate " +
			"YAML description of it.\n" +
			"\n" +
			"LIMITATION: While checksums are verified, signatures are not.",
		Args: cobra.ExactArgs(1),
		RunE: func(flags *cobra.Command, args []string) error {

			yamlBytes, err := os.ReadFile(platFile)
			if err != nil {
				return err
			}
			var plat struct {
				python.Platform
				PyCompile []string
			}
			if err := yaml.Unmarshal(yamlBytes, &plat, yaml.DisallowUnknownFields); err != nil {
				return fmt.Errorf("%s: %w", platFile, err)
			}
			plat.Platform.PyCompile, err = python.ExternalCompiler(plat.PyCompile...)
			if err != nil {
				return err
			}

			ctx := flags.Context()

			layer, err := bdist.InstallWheel(ctx,
				plat.Platform,
				time.Time{}, // minTime: zero; don't enforce minTime
				time.Time{}, // maxTime: zero; auto based on the timestamps in the wheel
				args[0],     // filename
				bdist.PostInstallHooks(
					entry_points.CreateScripts(plat.Platform),
					recording_installs.Record(
						"sha256",
						"ocibuild layer wheel",
						nil, // direct_url
					),
				),
			)
			if err != nil {
				return err
			}

			if err := fsutil.WriteLayer(layer, os.Stdout); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&platFile, "platform-file", "",
		"Read `IN_YAML_FILE` to determine details about the target platform")
	if err := cmd.MarkFlagRequired("platform-file"); err != nil {
		panic(err)
	}
	argparserLayer.AddCommand(cmd)
}
