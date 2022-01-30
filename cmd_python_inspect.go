package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
	"github.com/google/go-containerregistry/pkg/name"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/datawire/ocibuild/pkg/cliutil"
	"github.com/datawire/ocibuild/pkg/dockerutil"
	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/python"
	"github.com/datawire/ocibuild/pkg/python/pyinspect"
)

func init() {
	var flags struct {
		Interpreter string
		ImageFile   string
	}
	cmd := &cobra.Command{
		Use:   "inspect [flags] >PYTHON_PLATFORM.yml",
		Short: "Dump information about a Python environment",
		Args:  cliutil.WrapPositionalArgs(cobra.NoArgs),
		Long: "Inspect a Python environment, and dump information about it for " +
			"consumption by `ocibuild python wheel --platform-file=`.  The output " +
			"also includes some informative fields that are not used by " +
			"`ocibuild python wheel`." +
			"\n\n" +
			"LIMITATION: The --imagefile flag requires interacting with a running " +
			"Docker.",

		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			var image ociv1.Image
			if flags.ImageFile != "" {
				var err error
				image, err = fsutil.OpenImage(flags.ImageFile)
				if err != nil {
					return err
				}
			}

			sys := pyinspect.FS(pyinspect.NativeFS{})
			if image != nil {
				sys = &pyinspect.ImageFS{
					Image: image,
				}
			}

			var plat struct {
				python.Platform `yaml:",inline"`
				PyCompile       []string
			}
			var err error

			plat.ConsoleShebang, plat.GraphicalShebang, err = pyinspect.Shebangs(sys, flags.Interpreter)
			if err != nil {
				return err
			}

			var dyn *pyinspect.DynamicInfo
			if image == nil {
				dyn, err = pyinspect.Dynamic(ctx, plat.ConsoleShebang)
				if err != nil {
					return err
				}
			} else {
				if err := dockerutil.WithImage(ctx, "python-inspect",
					image,
					func(ctx context.Context, tag name.Tag) error {
						var err error
						dyn, err = pyinspect.Dynamic(ctx, "docker", "run",
							"--rm",
							"--entrypoint="+plat.ConsoleShebang,
							tag.String())
						return err
					},
				); err != nil {
					return err
				}
			}

			plat.Scheme = dyn.Scheme
			plat.VersionInfo = &dyn.VersionInfo
			plat.MagicNumber, err = base64.StdEncoding.DecodeString(dyn.MagicNumberB64)
			if err != nil {
				return err
			}
			plat.Tags = dyn.Tags

			dirs := []string{
				dyn.Scheme.PureLib,
				dyn.Scheme.PlatLib,
				dyn.Scheme.Headers,
				dyn.Scheme.Scripts,
				dyn.Scheme.Data,
			}
			foundOwner := false
			for _, dir := range dirs {
				info, err := sys.Stat(dir)
				if err != nil {
					continue
				}
				plat.UID = info.UID()
				plat.GID = info.GID()
				plat.UName = info.UName()
				plat.GName = info.GName()
				foundOwner = true
				break
			}
			if !foundOwner {
				return fmt.Errorf("could not stat any of the scheme directories: %#v", dyn.Scheme)
			}

			if image == nil {
				plat.PyCompile = []string{plat.ConsoleShebang, "-m", "compileall"}
			} else {
				names := []string{
					fmt.Sprintf("python%d.%d", dyn.VersionInfo.Major, dyn.VersionInfo.Minor),
					fmt.Sprintf("python%d", dyn.VersionInfo.Major),
					"python",
				}
				for _, name := range names {
					name, err := dexec.LookPath(name)
					if err != nil {
						continue
					}
					dlog.Infof(ctx, "inpsecting host %q...", name)
					nativeDyn, err := pyinspect.Dynamic(ctx, name)
					if err != nil {
						dlog.Infof(ctx, "... err %v", err)
						continue
					}
					if nativeDyn.MagicNumberB64 == dyn.MagicNumberB64 {
						plat.PyCompile = []string{name, "-m", "compileall"}
						break
					}
					magicForLog, err := base64.StdEncoding.DecodeString(nativeDyn.MagicNumberB64)
					if err != nil {
						dlog.Infof(ctx, "... err %v", err)
						continue
					}
					dlog.Infof(ctx, "... importlib.util.MAGIC_NUMBER=%q (sys.version_info=%+v)",
						magicForLog, nativeDyn.VersionInfo)
				}
				if plat.PyCompile == nil {
					return fmt.Errorf("unable to find a Python installatation on the host system that matches importlib.util.MAGIC_NUMBER=%q (sys.version_info=%+v)", //nolint:lll
						plat.MagicNumber, plat.VersionInfo)
				}
			}

			bs, err := yaml.Marshal(plat)
			if err != nil {
				return err
			}
			if _, err := os.Stdout.Write(bs); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flags.Interpreter, "interpreter", "python3",
		"The Python interpreter to inspect")
	cmd.Flags().StringVar(&flags.ImageFile, "imagefile", "",
		"Inspect a Docker image's Python rather than the host's Python")

	argparserPython.AddCommand(cmd)
}
