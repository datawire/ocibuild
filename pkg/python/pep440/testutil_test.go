// Copyright (C) 2021  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package pep440_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/datawire/ocibuild/pkg/python/pep440"
)

func intPtr(x int) *int {
	return &x
}

func mustParseVersion(t *testing.T, str string) pep440.Version {
	t.Helper()
	ver, err := pep440.ParseVersion(str)
	require.NoError(t, err)
	require.NotNil(t, ver)
	return *ver
}
