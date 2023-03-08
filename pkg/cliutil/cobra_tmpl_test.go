// Copyright (C) 2021  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package cliutil_test

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/datawire/ocibuild/pkg/cliutil"
)

//nolint:paralleltest // can't use .Parallel() with .Setenv()
func TestHelpTemplate(t *testing.T) {
	t.Setenv("COLUMNS", "80")
	noopRunE := func(_ *cobra.Command, _ []string) error {
		return nil
	}
	type testcase struct {
		InputCmd     *cobra.Command
		ExpectedHelp string
	}
	testcases := map[string]testcase{
		"basic": {
			InputCmd: func() *cobra.Command {
				cmd := &cobra.Command{
					Use:   "frobnicate [flags] VARS_ARE_UNDERSCORE_AND_CAPITAL",
					Args:  cobra.ExactArgs(1),
					Short: "One line description of program, no period",
					Long: "Longer description of program.  This is a paragraph.  " +
						"Because it is a paragraph, it may be quite long and " +
						"may need to be word-wrapped.",
					RunE: noopRunE,
				}
				cmd.Flags().BoolP("bar", "b", false, "Barzooble the baz")
				cmd.Flags().StringP("filename", "f", "", "Use `FILENAME` for the "+
					"complex thing that requires a long explanation that will "+
					"need to be wrapped on multiple lines")
				return cmd
			}(),
			ExpectedHelp: "" +
				// 0      1         2         3         4         5         6         7         8
				// 345678901234567890123456789012345678901234567890123456789012345678901234567890
				//                                                                          \n"  \n"
				"Usage: frobnicate [flags] VARS_ARE_UNDERSCORE_AND_CAPITAL\n" +
				"One line description of program, no period\n" +
				"\n" +
				"Longer description of program.  This is a paragraph.  Because it is a\n" +
				"paragraph, it may be quite long and may need to be word-wrapped.\n" +
				"\n" +
				"Flags:\n" +
				"  -b, --bar                 Barzooble the baz\n" +
				"  -f, --filename FILENAME   Use FILENAME for the complex thing that\n" +
				"                            requires a long explanation that will need to\n" +
				"                            be wrapped on multiple lines\n" +
				"",
		},
		"no-short": {
			InputCmd: func() *cobra.Command {
				cmd := &cobra.Command{
					Use:  "frobnicate [flags] VARS_ARE_UNDERSCORE_AND_CAPITAL",
					Args: cobra.ExactArgs(1),
					Long: "Longer description of program.  This is a paragraph.  " +
						"Because it is a paragraph, it may be quite long and " +
						"may need to be word-wrapped.",
					RunE: noopRunE,
				}
				cmd.Flags().BoolP("bar", "b", false, "Barzooble the baz")
				cmd.Flags().StringP("filename", "f", "", "Use `FILENAME` for the "+
					"complex thing that requires a long explanation that will "+
					"need to be wrapped on multiple lines")
				return cmd
			}(),
			ExpectedHelp: "" +
				// 0      1         2         3         4         5         6         7         8
				// 345678901234567890123456789012345678901234567890123456789012345678901234567890
				//                                                                          \n"  \n"
				"Usage: frobnicate [flags] VARS_ARE_UNDERSCORE_AND_CAPITAL\n" +
				"\n" +
				"Longer description of program.  This is a paragraph.  Because it is a\n" +
				"paragraph, it may be quite long and may need to be word-wrapped.\n" +
				"\n" +
				"Flags:\n" +
				"  -b, --bar                 Barzooble the baz\n" +
				"  -f, --filename FILENAME   Use FILENAME for the complex thing that\n" +
				"                            requires a long explanation that will need to\n" +
				"                            be wrapped on multiple lines\n" +
				"",
		},
		"no-long": {
			InputCmd: func() *cobra.Command {
				cmd := &cobra.Command{
					Use:   "frobnicate [flags] VARS_ARE_UNDERSCORE_AND_CAPITAL",
					Args:  cobra.ExactArgs(1),
					Short: "One line description of program, no period",
					RunE:  noopRunE,
				}
				cmd.Flags().BoolP("bar", "b", false, "Barzooble the baz")
				cmd.Flags().StringP("filename", "f", "", "Use `FILENAME` for the "+
					"complex thing that requires a long explanation that will "+
					"need to be wrapped on multiple lines")
				return cmd
			}(),
			ExpectedHelp: "" +
				// 0      1         2         3         4         5         6         7         8
				// 345678901234567890123456789012345678901234567890123456789012345678901234567890
				//                                                                          \n"  \n"
				"Usage: frobnicate [flags] VARS_ARE_UNDERSCORE_AND_CAPITAL\n" +
				"One line description of program, no period\n" +
				"\n" +
				"Flags:\n" +
				"  -b, --bar                 Barzooble the baz\n" +
				"  -f, --filename FILENAME   Use FILENAME for the complex thing that\n" +
				"                            requires a long explanation that will need to\n" +
				"                            be wrapped on multiple lines\n" +
				"",
		},
		"subcommandWrap": {
			InputCmd: func() *cobra.Command {
				cmd := &cobra.Command{
					Use:   "frobnicate [flags] VARS_ARE_UNDERSCORE_AND_CAPITAL",
					Args:  cobra.ExactArgs(1),
					Short: "One line description of program, no period",
					Long: "Longer description of program.  This is a paragraph.  " +
						"Because it is a paragraph, it may be quite long and " +
						"may need to be word-wrapped.",
					RunE: noopRunE,
				}
				cmd.Flags().BoolP("bar", "b", false, "Barzooble the baz")
				cmd.Flags().StringP("filename", "f", "", "Use `FILENAME` for the "+
					"complex thing that requires a long explanation that will "+
					"need to be wrapped on multiple lines")
				cmd.AddCommand(&cobra.Command{
					Use:   "example-subcommand [flags]",
					Args:  cobra.ExactArgs(0),
					Short: "One line description of subcommand, one line on own, but wrapped in table", //nolint:lll
					RunE:  noopRunE,
				})
				return cmd
			}(),
			ExpectedHelp: "" +
				// 0      1         2         3         4         5         6         7         8
				// 345678901234567890123456789012345678901234567890123456789012345678901234567890
				//                                                                         \n"   \n"
				"Usage: frobnicate [flags] VARS_ARE_UNDERSCORE_AND_CAPITAL\n" +
				"One line description of program, no period\n" +
				"\n" +
				"Longer description of program.  This is a paragraph.  Because it is a\n" +
				"paragraph, it may be quite long and may need to be word-wrapped.\n" +
				"\n" +
				"Available Commands:\n" +
				"  example-subcommand   One line description of subcommand, one line on\n" +
				"                       own, but wrapped in table\n" +
				"\n" +
				"Flags:\n" +
				"  -b, --bar                 Barzooble the baz\n" +
				"  -f, --filename FILENAME   Use FILENAME for the complex thing that\n" +
				"                            requires a long explanation that will need to\n" +
				"                            be wrapped on multiple lines\n" +
				"\n" +
				"Use \"frobnicate [command] --help\" for more information about a command.\n" +
				"",
		},
	}
	for tcName, tcData := range testcases {
		tcData := tcData
		t.Run(tcName, func(t *testing.T) {
			tcData.InputCmd.SetHelpTemplate(cliutil.HelpTemplate)

			var out strings.Builder
			tcData.InputCmd.SetOutput(&out)
			tcData.InputCmd.HelpFunc()(tcData.InputCmd, []string{"--help"})

			assert.Equal(t, tcData.ExpectedHelp, out.String())
		})
	}
}
