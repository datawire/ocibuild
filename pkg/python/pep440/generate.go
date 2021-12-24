package pep440

import (
	"math/rand"
	"reflect"
	"testing/quick"

	"k8s.io/apimachinery/pkg/util/intstr"
)

func randBool(rand *rand.Rand) bool {
	return rand.Intn(2) == 1
}

func randSeg(rand *rand.Rand) int {
	return rand.Intn(3000)
}

func bound(low, val, high int) int {
	if val < low {
		val = low
	}
	if val > high {
		val = high
	}
	return val
}

func (ver PublicVersion) generate(rand *rand.Rand, size int) PublicVersion {
	if randBool(rand) {
		ver.Epoch = randSeg(rand)
	}
	ver.Release = make([]int, 1+rand.Intn(bound(1, size, 10)))
	for i := range ver.Release {
		ver.Release[i] = randSeg(rand)
	}
	if randBool(rand) {
		ver.Pre = &PreRelease{
			L: []string{"a", "b", "rc"}[rand.Intn(3)],
			N: randSeg(rand),
		}
	}
	if randBool(rand) {
		n := randSeg(rand)
		ver.Post = &n
	}
	if randBool(rand) {
		n := randSeg(rand)
		ver.Dev = &n
	}
	return ver
}

// Generate implements testing/quick.Generator.
func (ver PublicVersion) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(ver.generate(rand, size))
}

func (ver LocalVersion) generate(rand *rand.Rand, size int) LocalVersion {
	if randBool(rand) {
		ver.Local = make([]intstr.IntOrString, 1+rand.Intn(bound(1, size, 10)))
		size -= len(ver.Local)
		for i := range ver.Local {
			if randBool(rand) {
				ver.Local[i] = intstr.FromInt(randSeg(rand))
			} else {
				buf := make([]byte, 1+rand.Intn(bound(1, size, 10)))
				size -= len(buf)
				const (
					alpha    = "abcdefghijklmnopqrstuvwxyz"
					alphadig = alpha + "0123456789"
				)
				for i := range buf {
					if i == 0 {
						buf[i] = alpha[rand.Intn(len(alpha))]
					} else {
						buf[i] = alphadig[rand.Intn(len(alphadig))]
					}
				}
				ver.Local[i] = intstr.FromString(string(buf))
			}
		}
	}

	ver.PublicVersion = ver.PublicVersion.generate(rand, size)

	return ver
}

// Generate implements testing/quick.Generator.
func (ver LocalVersion) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(ver.generate(rand, size))
}

//nolint:exhaustivestruct
var _ quick.Generator = LocalVersion{}

func (op CmpOp) generate(rand *rand.Rand, _ int) CmpOp {
	return CmpOp(rand.Intn(int(_CmpOpEnd)))
}

// Generate implements testing/quick.Generator.
func (op CmpOp) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(op.generate(rand, size))
}
