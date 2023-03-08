#!/usr/bin/env bash
# Copyright (C) 2019  Ambassador Labs
#
# SPDX-License-Identifier: Apache-2.0

if ! cmp -s "$1" "$2"; then
	if [[ -n "$CI" && -e "$2" ]]; then
		echo "error: This should not happen in CI: $2 should not change" >&2
		diff -u "$2" "$1" >&2
		exit 1
	fi
	cp -f "$1" "$2"
fi
