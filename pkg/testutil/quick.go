// Copyright (C) 2021  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

// QuickCheck is similar to testing/quick.Check, but takes an additional list of static items to
// feed as inputs.
func QuickCheck(t *testing.T, fn interface{}, cfg quick.Config, testcases ...[]interface{}) {
	t.Helper()
	err := quick.Check(fn, &cfg)
	assert.NoError(t, err)
	var setupErr quick.SetupError
	if errors.As(err, &setupErr) {
		return
	}

	fnVal := reflect.ValueOf(fn)
	for i, tc := range testcases {
		if len(tc) != fnVal.Type().NumIn() {
			t.Errorf("static#%d has %d args, but the function takes %d args",
				i, len(tc), fnVal.Type().NumIn())
			continue
		}
		args := make([]reflect.Value, len(tc))
		for j := range args {
			args[j] = reflect.ValueOf(tc[i])
		}
		if !fnVal.Call(args)[0].Bool() {
			assert.NoError(t, fmt.Errorf("static%w", &quick.CheckError{
				Count: i + 1,
				In:    toInterfaces(args),
			}))
		}
	}
}

// QuickCheckEqual is similar to testing/quick.CheckEqual, but takes an additional list of static
// items to feed as inputs.
func QuickCheckEqual(t *testing.T, fn1, fn2 interface{}, cfg quick.Config, testcases ...[]interface{}) {
	t.Helper()
	err := quick.CheckEqual(fn1, fn2, &cfg)
	assert.NoError(t, err)
	var setupErr quick.SetupError
	if errors.As(err, &setupErr) {
		return
	}

	fn1Val := reflect.ValueOf(fn1)
	fn2Val := reflect.ValueOf(fn2)
	for i, tc := range testcases {
		if len(tc) != fn1Val.Type().NumIn() {
			t.Errorf("static#%d has %d args, but the functions take %d args",
				i, len(tc), fn1Val.Type().NumIn())
			continue
		}
		args := make([]reflect.Value, len(tc))
		for j := range args {
			args[j] = reflect.ValueOf(tc[j])
		}
		ret1 := toInterfaces(fn1Val.Call(args))
		ret2 := toInterfaces(fn2Val.Call(args))
		if !reflect.DeepEqual(ret1, ret2) {
			assert.NoError(t, fmt.Errorf("static%w", &quick.CheckEqualError{
				CheckError: quick.CheckError{
					Count: i + 1,
					In:    toInterfaces(args),
				},
				Out1: ret1,
				Out2: ret2,
			}))
		}
	}
}

func toInterfaces(values []reflect.Value) []interface{} {
	ret := make([]interface{}, len(values))
	for i, val := range values {
		ret[i] = val.Interface()
	}
	return ret
}

type QuickConfig = quick.Config
