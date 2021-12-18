package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/datawire/ocibuild/pkg/python/pep503"
	"github.com/datawire/ocibuild/pkg/python/pypa/bdist"
	"github.com/datawire/ocibuild/pkg/python/pypa/simple_repo_api"
)

func init() {
	var indexServer string
	cmd := &cobra.Command{
		Use:   "getwheel [flags] NAME_VERSION_PLATFORM.whl >NAME_VERSION_PLATFORM.whl",
		Short: "Download a wheel file from the Python Package Index",
		Args:  cobra.ExactArgs(1),

		Long: "Given a wheel filename, download it from a package index, writing the file " +
			"contents to stdout." +
			"\n\n" +
			"LIMITATION: Generating the list of wheel files to download is " +
			"non-obvious at this point; soon there will be an " +
			"`ocibuild python SOMETHING` command that will spit out a list of wheel " +
			"filenames, but it doesn't exist yet.  I'm not sure if you can get pip to " +
			"give it to you.  pip-compile only gives you (name, version) tuples, not " +
			"the full (name, version, platform) tuple." +
			"\n\n" +
			"LIMITATION: While checksums are verified, GPG signatures are not.",

		RunE: func(flags *cobra.Command, args []string) error {
			ctx := flags.Context()
			filename := args[0]
			filenameInfo, err := bdist.ParseFilename(filename)
			if err != nil {
				return err
			}
			client := simple_repo_api.NewClient(nil, nil)
			client.BaseURL = indexServer
			links, err := client.ListPackageFiles(ctx, filenameInfo.Distribution)
			if err != nil {
				return err
			}
			for _, link := range links {
				if link.Text == filename {
					content, err := link.Get(ctx)
					if err != nil {
						return err
					}
					if _, err := os.Stdout.Write(content); err != nil {
						return err
					}
					return nil
				}
			}
			return fmt.Errorf("package index does not have wheel %q", filename)
		},
	}
	cmd.Flags().StringVar(&indexServer, "index-server", pep503.PyPIBaseURL,
		"Index server to download the wheel from")

	argparserPython.AddCommand(cmd)
}
