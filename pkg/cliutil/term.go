// Copyright (C) 2020  Ambassador Labs (for Telepresence)
// Copyright (C) 2021  Ambassador Labs (for ocibuild)
//
// SPDX-License-Identifier: Apache-2.0
//
// Based on
// https://github.com/telepresenceio/telepresence/blob/b6dfa04ff014915b47386191cc3d8b1352522fea/pkg/client/cli/command_group.go#L35-L63

package cliutil

import (
	"os"
	"strconv"

	"golang.org/x/term"
)

// GetTerminalWidth returns the width of the terminal that you should wrap text to.
func GetTerminalWidth() int {
	// Copyright note: This code was originally written by LukeShu for Telepresence.

	// This is based off of what Docker does (github.com/docker/cli/cli/cobra.go), but is
	// adjusted to correct for the ways that Docker upsets me.

	// Obey COLUMNS if the shell or user sets it.  (Docker doesn't do this.)
	if cols, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil {
		return cols
	}

	// Try to detect the size of the stdout file descriptor.  (Docker checks stdin, not stdout.)
	if cols, _, err := term.GetSize(1); err == nil {
		return cols
	}

	// If stdout is a terminal but we were unable to get its size (I'm not sure how that can
	// happen), then fall back to assuming 80.
	if term.IsTerminal(1) {
		return 80
	}

	// If stdout isn't a terminal, then we leave cols as 0, meaning "don't wrap it".  (Docker
	// wraps it even if stdout isn't a terminal.)
	return 0
}
