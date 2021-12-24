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

type VersionSpecifierClause struct {
	CmpOp   string
	Version pep440.Version
}

func parseVersionSpecifierClause(str string) (VersionSpecifierClause, error) {
	var ret VersionSpecifierClause
	str = strings.TrimSpace(str)
	switch {
	case strings.HasPrefix("<", str) && !strings.HasPrefix("<=", str):
		ret.CmpOp = str[:1]
		str = str[1:]
	case strings.HasPrefix(">", str) && !strings.HasPrefix(">=", str):
		ret.CmpOp = str[:1]
		str = str[1:]
	case strings.HasPrefix("<=", str):
		ret.CmpOp = str[:2]
		str = str[2:]
	case strings.HasPrefix(">=", str):
		ret.CmpOp = str[:2]
		str = str[2:]
	case strings.HasPrefix("==", str):
		ret.CmpOp = str[:2]
		str = str[2:]
	case strings.HasPrefix("!=", str):
		ret.CmpOp = str[:2]
		str = str[2:]
	}
	ver, err := pep440.ParseVersion(str)
	if err != nil {
		return ret, err
	}
	if !ver.IsFinal() {
		return ret, fmt.Errorf("version in specifier is not a final version: %q", ver.String())
	}
	ret.Version = *ver
	return ret, nil
}

func (spec VersionSpecifierClause) Match(ver pep440.Version) bool {
	if !ver.IsFinal() {
		return false
	}
	if len(ver.Release) < len(spec.Version.Release) {
		ver.Release = ver.Release[:len(spec.Version.Release)]
	}
	switch spec.CmpOp {
	case "<":
		return spec.Version.Cmp(ver) < 0
	case "<=":
		return spec.Version.Cmp(ver) <= 0
	case ">":
		return spec.Version.Cmp(ver) > 0
	case ">=":
		return spec.Version.Cmp(ver) >= 0
	case "==":
		return spec.Version.Cmp(ver) == 0
	case "!=":
		return spec.Version.Cmp(ver) != 0
	default:
		panic(fmt.Errorf("invalid CmpOp: %q", spec.CmpOp))
	}
}
