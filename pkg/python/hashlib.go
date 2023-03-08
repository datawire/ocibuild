// Copyright (C) 2021-2022  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package python

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
)

// HashlibAlgorithmsGuaranteed is Python `hashlib.algorithms_guaranteed`.
//
//nolint:gochecknoglobals // Would be 'const'.
var HashlibAlgorithmsGuaranteed = map[string]func() hash.Hash{
	// This list is (sans TODOs) in-sync with Python 3.9.9.
	"md5":    md5.New,
	"sha1":   sha1.New,
	"sha224": sha256.New224,
	"sha256": sha256.New,
	"sha384": sha512.New384,
	"sha512": sha512.New,
	// "blake2b":   TODO,
	// "blake2s":   TODO,
	// "sha3_224":  TODO,
	// "sha3_256":  TODO,
	// "sha3_384":  TODO,
	// "sha3_512":  TODO,
	// "shake_128": TODO,
	// "shake_256": TODO,
}
