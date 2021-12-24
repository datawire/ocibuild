// Package pep345 implements PEP 345 -- Metadata for Python Software Packages 1.2.
//
// Well, just enough of PEP 345 to implement PEP 503.
//
// https://www.python.org/dev/peps/pep-0345/
package pep345

import (
	"fmt"
	"strings"

	"github.com/datawire/ocibuild/pkg/python/pep440"
)

// HaveRequiredPython returns whether the `requirement` from the "Requires-Python" field is
// satisfied.
func HaveRequiredPython(have pep440.Version, requirement string) (bool, error) {
	req, err := ParseVersionSpecifier(requirement)
	if err != nil {
		return false, err
	}
	return req.Match(have), nil
}

type VersionSpecifier []VersionSpecifierClause

func ParseVersionSpecifier(str string) (VersionSpecifier, error) {
	clauseStrs := strings.FieldsFunc(str, func(r rune) bool { return r == ',' })
	ret := make(VersionSpecifier, 0, len(clauseStrs))
	for _, clauseStr := range clauseStrs {
		clause, err := parseVersionSpecifierClause(clauseStr)
		if err != nil {
			return nil, fmt.Errorf("pep345.ParseVersionSpecifier: %w", err)
		}
		ret = append(ret, clause)
	}
	return ret, nil
}

func (spec VersionSpecifier) Match(ver pep440.Version) bool {
	for _, clause := range spec {
		if !clause.Match(ver) {
			return false
		}
	}
	return true
}

type CmpOp int

const (
	CmpOp_LT CmpOp = iota
	CmpOp_GT
	CmpOp_LE
	CmpOp_GE
	CmpOp_EQ
	CmpOp_NE
)

func (op CmpOp) String() string {
	str, ok := map[CmpOp]string{
		CmpOp_LT: "<",
		CmpOp_GT: ">",
		CmpOp_LE: "<=",
		CmpOp_GE: ">=",
		CmpOp_EQ: "==",
		CmpOp_NE: "!=",
	}[op]
	if !ok {
		panic(fmt.Errorf("invalid CmpOp: %d", op))
	}
	return str
}

type VersionSpecifierClause struct {
	CmpOp   CmpOp
	Version pep440.Version
}

func parseVersionSpecifierClause(str string) (VersionSpecifierClause, error) {
	var ret VersionSpecifierClause
	str = strings.TrimSpace(str)
	switch {
	case strings.HasPrefix(str, "<") && !strings.HasPrefix(str, "<="):
		ret.CmpOp = CmpOp_LT
		str = str[1:]
	case strings.HasPrefix(str, ">") && !strings.HasPrefix(str, ">="):
		ret.CmpOp = CmpOp_GT
		str = str[1:]
	case strings.HasPrefix(str, "<="):
		ret.CmpOp = CmpOp_LE
		str = str[2:]
	case strings.HasPrefix(str, ">="):
		ret.CmpOp = CmpOp_GE
		str = str[2:]
	case strings.HasPrefix(str, "=="):
		ret.CmpOp = CmpOp_EQ
		str = str[2:]
	case strings.HasPrefix(str, "!="):
		ret.CmpOp = CmpOp_NE
		str = str[2:]
	default:
		ret.CmpOp = CmpOp_EQ
	}
	ver, err := pep440.ParseVersion(str)
	if err != nil {
		return ret, err
	}
	ret.Version = *ver
	return ret, nil
}

func (spec VersionSpecifierClause) Match(ver pep440.Version) bool {
	switch spec.CmpOp {
	case CmpOp_LT:
		// also exclude pre-releases
		excl := pep440.SpecifierClause{CmpOp: pep440.CmpOp_PrefixExclude, Version: spec.Version}
		if len(spec.Version.Local) > 0 || spec.Version.Dev != nil {
			// not allowed to use PrefixExclude in these cases
			excl.CmpOp = pep440.CmpOp_StrictExclude
		}
		return ver.Cmp(spec.Version) < 0 && excl.Match(ver)
	case CmpOp_LE:
		return ver.Cmp(spec.Version) <= 0
	case CmpOp_GT:
		return ver.Cmp(spec.Version) > 0
	case CmpOp_GE:
		return ver.Cmp(spec.Version) >= 0
	case CmpOp_EQ:
		// base part
		base := pep440.SpecifierClause{CmpOp: pep440.CmpOp_PrefixMatch, Version: spec.Version}
		if len(spec.Version.Local) > 0 || spec.Version.Dev != nil {
			// not allowed to use PrefixMatch in these cases
			base.CmpOp = pep440.CmpOp_StrictMatch
		}
		if !base.Match(ver) {
			return false
		}
		// also exclude pre-releases and post-releases
		switch {
		case spec.Version.Dev != nil:
			// allow anything
			return true
		case spec.Version.Post != nil:
			// dissallow dev
			return ver.Dev == nil
		case spec.Version.Pre != nil:
			// dissallow dev, post
			return ver.Dev == nil && ver.Post == nil
		default:
			// dissallow dev, post, pre
			return ver.Dev == nil && ver.Post == nil && ver.Pre == nil
		}
	case CmpOp_NE:
		spec.CmpOp = CmpOp_EQ
		return !spec.Match(ver)
	default:
		panic(fmt.Errorf("invalid CmpOp: %q", spec.CmpOp))
	}
}
