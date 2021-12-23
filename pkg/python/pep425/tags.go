// Package pep425 implements PEP 425 -- Database of Installed Python Distributions.
//
// https://www.python.org/dev/peps/pep-0425/
package pep425

import (
	"strings"
)

type Tag struct {
	Python   string
	ABI      string
	Platform string
}

func (t Tag) Decompress() []Tag {
	var ret []Tag
	for _, x := range strings.Split(t.Python, ".") {
		for _, y := range strings.Split(t.ABI, ".") {
			for _, z := range strings.Split(t.Platform, ".") {
				ret = append(ret, Tag{x, y, z})
			}
		}
	}
	return ret
}

func (t Tag) String() string {
	return t.Python + "-" + t.ABI + "-" + t.Platform
}

// Intersect returns whether any tag in tag-list 'a' matches any tag in tag-list 'b'; considering
// compressed tag sets.
func Intersect(a, b []Tag) bool {
	for _, a1 := range a {
		for _, a2 := range a1.Decompress() {
			for _, b1 := range b {
				for _, b2 := range b1.Decompress() {
					if a2 == b2 {
						return true
					}
				}
			}
		}
	}
	return false
}

// Installer is a list of tags that an installer supports, ordered from most-preferred to
// least-preferred.
//
// To get this for a live Python install, use the command:
//
//     python -c $'import packaging.tags\nfor tag in packaging.tags.sys_tags(): print(tag)'
type Installer []Tag

func (inst Installer) Supports(t Tag) bool {
	return Intersect([]Tag(inst), []Tag{t})
}

// Preference returns a numeric representation of how much this Tag is preferred by the installer;
// may be used to sort things by Tag preference; lower values are more preferred.  The returned
// value is in the range [1,len(inst+1)]; the zero value is safe to use as "unset".
func (inst Installer) Preference(t Tag) int {
	for i, it := range inst {
		if Intersect([]Tag{it}, []Tag{t}) {
			return i + 1
		}
	}
	return len(inst) + 1
}
