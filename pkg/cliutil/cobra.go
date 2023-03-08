// Copyright (C) 2020  Ambassador Labs (for Telepresence)
// Copyright (C) 2021-2022  Ambassador Labs (for ocibuild)
//
// SPDX-License-Identifier: Apache-2.0
//
// Contains code from
// https://github.com/telepresenceio/telepresence/blob/3b63073ceafae6b548c664a83f7ac90497eab2ae/pkg/client/cli/command.go

package cliutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// OnlySubcommands is a cobra.PositionalArgs that is similar to cobra.NoArgs, but prints a better
// error message.
func OnlySubcommands(cmd *cobra.Command, args []string) error {
	// Copyright note: This code was originally written by LukeShu for Telepresence.
	if len(args) != 0 {
		err := fmt.Errorf("invalid subcommand %q", args[0])

		if cmd.SuggestionsMinimumDistance <= 0 {
			cmd.SuggestionsMinimumDistance = 2
		}
		if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
			err = fmt.Errorf("%w\nDid you mean one of these?\n\t%s", err, strings.Join(suggestions, "\n\t"))
		}

		return cmd.FlagErrorFunc()(cmd, err)
	}
	return nil
}

// WrapPositionalArgs wraps a cobra.PositionalArgs to have it pass any errors through FlagErrorFunc,
// in order to have more consistent bad-usage reporting.
func WrapPositionalArgs(inner cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		return FlagErrorFunc(cmd, inner(cmd, args))
	}
}

// RunSubCommands is for use as a cobra.Command.RunE for commands that don't do anything themselves
// but have subcommands.  In such cases, it is important to set RunE even though there's nothing to
// run, because otherwise cobra will treat that as "success", and it shouldn't be "success" if the
// user typos a command and types something invalid.
func RunSubcommands(cmd *cobra.Command, args []string) error {
	// Copyright note: This code was originally written by LukeShu for Telepresence.
	cmd.SetOutput(cmd.ErrOrStderr())
	cmd.HelpFunc()(cmd, args)
	os.Exit(2)
	return nil
}

// FlagErrorFunc is a function to be passed to (*cobra.Command).SetFlagErrorFunc that establishes
// GNU-ish behavior for invalid flag usage.
//
// If there is an error, FlagErrorFunc calls os.Exit; it does NOT return.  This means that all
// errors returned from (*cobra.Command).Execute will be execution errors, not usage errors.
func FlagErrorFunc(cmd *cobra.Command, err error) error {
	// Copyright note: This code was originally written by LukeShu for Telepresence.
	if err == nil {
		return nil
	}

	// If the error is multiple lines, include an extra blank line before the "See --help" line.
	errStr := strings.TrimRight(err.Error(), "\n")
	if strings.Contains(errStr, "\n") {
		errStr += "\n"
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "%s: %s\nSee '%s --help' for more information.\n",
		cmd.CommandPath(), errStr, cmd.CommandPath())
	os.Exit(2)
	return nil
}
