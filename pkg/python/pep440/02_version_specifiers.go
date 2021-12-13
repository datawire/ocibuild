package pep440

import (
	"fmt"
	"strings"
)

// Version specifiers
// ==================
//
// A version specifier consists of a series of version clauses, separated by
// commas. For example::
//
//    ~= 0.9, >= 1.0, != 1.3.4.*, < 2.0
//
// The comparison operator determines the kind of version clause:
//
// * ``~=``: `Compatible release`_ clause
// * ``==``: `Version matching`_ clause
// * ``!=``: `Version exclusion`_ clause
// * ``<=``, ``>=``: `Inclusive ordered comparison`_ clause
// * ``<``, ``>``: `Exclusive ordered comparison`_ clause
// * ``===``: `Arbitrary equality`_ clause.
//
// The comma (",") is equivalent to a logical **and** operator: a candidate
// version must match all given version clauses in order to match the
// specifier as a whole.
//
// Whitespace between a conditional operator and the following version
// identifier is optional, as is the whitespace around the commas.
//
// When multiple candidate versions match a version specifier, the preferred
// version SHOULD be the latest version as determined by the consistent
// ordering defined by the standard `Version scheme`_. Whether or not
// pre-releases are considered as candidate versions SHOULD be handled as
// described in `Handling of pre-releases`_.
//
// Except where specifically noted below, local version identifiers MUST NOT be
// permitted in version specifiers, and local version labels MUST be ignored
// entirely when checking if candidate versions match a given version
// specifier.

type Specifier []SpecifierClause

func ParseSpecifier(str string) (Specifier, error) {
	clauseStrs := strings.FieldsFunc(str, func(r rune) bool { return r == ',' })
	ret := make(Specifier, 0, len(clauseStrs))
	for _, clauseStr := range clauseStrs {
		clause, err := parseSpecifierClause(clauseStr)
		if err != nil {
			return nil, fmt.Errorf("pep440.ParseSpecifier: %w", err)
		}
		ret = append(ret, clause)
	}
	return ret, nil
}

func (s Specifier) Match(v Version) bool {
	for _, clause := range s {
		if !clause.Match(v) {
			return false
		}
	}
	return true
}

type CmpOp int

const (
	CmpOp_Compatible CmpOp = iota
	CmpOp_StrictMatch
	CmpOp_PrefixMatch
	CmpOp_StrictExclude
	CmpOp_PrefixExclude
	CmpOp_LE
	CmpOp_GE
	CmpOp_LT
	CmpOp_GT
	//CmpOp_Arbitrary
)

func (op CmpOp) String() string {
	str, ok := map[CmpOp]string{
		CmpOp_Compatible:    "~=",
		CmpOp_StrictMatch:   "strict ==",
		CmpOp_PrefixMatch:   "prefix ==",
		CmpOp_StrictExclude: "strict !=",
		CmpOp_PrefixExclude: "prefix !=",
		CmpOp_LE:            "<=",
		CmpOp_GE:            ">=",
		CmpOp_LT:            "<",
		CmpOp_GT:            ">",
	}[op]
	if !ok {
		panic(fmt.Errorf("invalid CmpOp: %d", op))
	}
	return str
}

func (op CmpOp) match(s, v Version) bool {
	fn, ok := map[CmpOp]func(s, v Version) bool{
		CmpOp_Compatible:    matchCompatible,
		CmpOp_StrictMatch:   matchStrictMatch,
		CmpOp_PrefixMatch:   matchPrefixMatch,
		CmpOp_StrictExclude: matchStrictExclude,
		CmpOp_PrefixExclude: matchPrefixExclude,
		CmpOp_LE:            matchLE,
		CmpOp_GE:            matchGE,
		CmpOp_LT:            matchLT,
		CmpOp_GT:            matchGT,
	}[op]
	if !ok {
		panic(fmt.Errorf("invalid CmpOp: %d", op))
	}
	return fn(s, v)
}

type SpecifierClause struct {
	CmpOp   CmpOp
	Version Version
}

func parseSpecifierClause(str string) (SpecifierClause, error) {
	var ret SpecifierClause
	str = strings.TrimSpace(str)
	devOK := true
	localOK := false
	switch {
	case strings.HasPrefix(str, "~="):
		ret.CmpOp = CmpOp_Compatible
		str = str[2:]
	case strings.HasPrefix(str, "==") && !strings.HasPrefix(str, "==="):
		ret.CmpOp = CmpOp_StrictMatch
		str = str[2:]
		localOK = true
		if strings.HasSuffix(str, ".*") {
			ret.CmpOp = CmpOp_PrefixMatch
			str = strings.TrimSuffix(str, ".*")
			devOK = false
			localOK = false
		}
	case strings.HasPrefix(str, "!="):
		ret.CmpOp = CmpOp_StrictExclude
		str = str[2:]
		localOK = true
		if strings.HasSuffix(str, ".*") {
			ret.CmpOp = CmpOp_PrefixExclude
			str = strings.TrimSuffix(str, ".*")
			devOK = false
			localOK = false
		}
	case strings.HasPrefix(str, "<="):
		ret.CmpOp = CmpOp_LE
		str = str[2:]
	case strings.HasPrefix(str, ">="):
		ret.CmpOp = CmpOp_GE
		str = str[2:]
	case strings.HasPrefix(str, "<"):
		ret.CmpOp = CmpOp_LT
		str = str[2:]
	case strings.HasPrefix(str, ">"):
		ret.CmpOp = CmpOp_GT
		str = str[2:]
	case strings.HasPrefix(str, "==="):
		return ret, fmt.Errorf("the === specifier is not supported; versions must be PEP 440 compliant")
	}
	v, err := ParseVersion(str)
	if err != nil {
		return ret, err
	}
	if v.Dev != nil && !devOK {
		return ret, fmt.Errorf("dev-part not permitted in %s specifier clauses", ret.CmpOp)
	}
	if len(v.Local) > 0 && !localOK {
		return ret, fmt.Errorf("local-part not permitted in %s specifier clauses", ret.CmpOp)
	}
	ret.Version = *v
	return ret, nil
}

func (s SpecifierClause) Match(v Version) bool {
	return s.CmpOp.match(s.Version, v)
}

//
//

// Compatible release
// ------------------
//
// A compatible release clause consists of the compatible release operator ``~=``
// and a version identifier. It matches any candidate version that is expected
// to be compatible with the specified version.
//
// The specified version identifier must be in the standard format described in
// `Version scheme`_. Local version identifiers are NOT permitted in this
// version specifier.
//
// For a given release identifier ``V.N``, the compatible release clause is
// approximately equivalent to the pair of comparison clauses::
//
//     >= V.N, == V.*
//
// This operator MUST NOT be used with a single segment version number such as
// ``~=1``.
//
// For example, the following groups of version clauses are equivalent::
//
//     ~= 2.2
//     >= 2.2, == 2.*
//
//     ~= 1.4.5
//     >= 1.4.5, == 1.4.*
//
// If a pre-release, post-release or developmental release is named in a
// compatible release clause as ``V.N.suffix``, then the suffix is ignored
// when determining the required prefix match::
//
//     ~= 2.2.post3
//     >= 2.2.post3, == 2.*
//
//     ~= 1.4.5a4
//     >= 1.4.5a4, == 1.4.*
//
// The padding rules for release segment comparisons means that the assumed
// degree of forward compatibility in a compatible release clause can be
// controlled by appending additional zeros to the version specifier::
//
//     ~= 2.2.0
//     >= 2.2.0, == 2.2.*
//
//     ~= 1.4.5.0
//     >= 1.4.5.0, == 1.4.5.*
func matchCompatible(s, v Version) bool {
	prefix := s
	prefix.Pre = nil
	prefix.Post = nil
	prefix.Dev = nil
	return matchGE(s, v) && matchPrefixMatch(prefix, v)
}

//
//

// Version matching
// ----------------
//
// A version matching clause includes the version matching operator ``==``
// and a version identifier.
//
// The specified version identifier must be in the standard format described in
// `Version scheme`_, but a trailing ``.*`` is permitted on public version
// identifiers as described below.
//
// By default, the version matching operator is based on a strict equality
// comparison: the specified version must be exactly the same as the requested
// version. The *only* substitution performed is the zero padding of the
// release segment to ensure the release segments are compared with the same
// length.
//
// Whether or not strict version matching is appropriate depends on the specific
// use case for the version specifier. Automated tools SHOULD at least issue
// warnings and MAY reject them entirely when strict version matches are used
// inappropriately.
//
// Prefix matching may be requested instead of strict comparison, by appending
// a trailing ``.*`` to the version identifier in the version matching clause.
// This means that additional trailing segments will be ignored when
// determining whether or not a version identifier matches the clause. If the
// specified version includes only a release segment, than trailing components
// (or the lack thereof) in the release segment are also ignored.
//
// For example, given the version ``1.1.post1``, the following clauses would
// match or not as shown::
//
//     == 1.1        # Not equal, so 1.1.post1 does not match clause
//     == 1.1.post1  # Equal, so 1.1.post1 matches clause
//     == 1.1.*      # Same prefix, so 1.1.post1 matches clause
//
// For purposes of prefix matching, the pre-release segment is considered to
// have an implied preceding ``.``, so given the version ``1.1a1``, the
// following clauses would match or not as shown::
//
//     == 1.1        # Not equal, so 1.1a1 does not match clause
//     == 1.1a1      # Equal, so 1.1a1 matches clause
//     == 1.1.*      # Same prefix, so 1.1a1 matches clause
//
// An exact match is also considered a prefix match (this interpretation is
// implied by the usual zero padding rules for the release segment of version
// identifiers). Given the version ``1.1``, the following clauses would
// match or not as shown::
//
//     == 1.1        # Equal, so 1.1 matches clause
//     == 1.1.0      # Zero padding expands 1.1 to 1.1.0, so it matches clause
//     == 1.1.dev1   # Not equal (dev-release), so 1.1 does not match clause
//     == 1.1a1      # Not equal (pre-release), so 1.1 does not match clause
//     == 1.1.post1  # Not equal (post-release), so 1.1 does not match clause
//     == 1.1.*      # Same prefix, so 1.1 matches clause
//
// It is invalid to have a prefix match containing a development or local release
// such as ``1.0.dev1.*`` or ``1.0+foo1.*``. If present, the development release
// segment is always the final segment in the public version, and the local version
// is ignored for comparison purposes, so using either in a prefix match wouldn't
// make any sense.
//
// The use of ``==`` (without at least the wildcard suffix) when defining
// dependencies for published distributions is strongly discouraged as it
// greatly complicates the deployment of security fixes. The strict version
// comparison operator is intended primarily for use when defining
// dependencies for repeatable *deployments of applications* while using
// a shared distribution index.
//
// If the specified version identifier is a public version identifier (no
// local version label), then the local version label of any candidate versions
// MUST be ignored when matching versions.
//
// If the specified version identifier is a local version identifier, then the
// local version labels of candidate versions MUST be considered when matching
// versions, with the public version identifier being matched as described
// above, and the local version label being checked for equivalence using a
// strict string equality comparison.
func matchStrictMatch(s, v Version) bool {
	if len(s.Local) == 0 {
		return s.PublicVersion.Cmp(v.PublicVersion) == 0
	}
	return s.Cmp(v) == 0
}
func matchPrefixMatch(_s, _v Version) bool {
	s, v := _s.PublicVersion, _v.PublicVersion
	const (
		partRel = iota
		partPre
		partPost
	)
	var part int
	switch {
	case s.Post != nil:
		part = partPost
	case s.Pre != nil:
		part = partPre
	default:
		part = partRel
	}

	if cmpEpoch(s, v) != 0 {
		return false
	}

	if part == partRel {
		return cmpRelease(s, v) <= 0
	}
	if cmpRelease(s, v) != 0 {
		return false
	}

	if part == partPre {
		order := map[string]int{
			"a":     -3,
			"alpha": -3,

			"b":    -2,
			"beta": -2,

			"rc":      -1,
			"c":       -1,
			"pre":     -1,
			"preview": -1,
		}
		return v.Pre != nil && order[s.Pre.L] == order[v.Pre.L]
	}
	if cmpPreRelease(s, v) != 0 {
		return false
	}

	if part == partPost {
		return v.Post != nil
	}
	panic("not reached")
}

//
//

// Version exclusion
// -----------------
//
// A version exclusion clause includes the version exclusion operator ``!=``
// and a version identifier.
//
// The allowed version identifiers and comparison semantics are the same as
// those of the `Version matching`_ operator, except that the sense of any
// match is inverted.
//
// For example, given the version ``1.1.post1``, the following clauses would
// match or not as shown::
//
//     != 1.1        # Not equal, so 1.1.post1 matches clause
//     != 1.1.post1  # Equal, so 1.1.post1 does not match clause
//     != 1.1.*      # Same prefix, so 1.1.post1 does not match clause
func matchStrictExclude(s, v Version) bool {
	return !matchStrictMatch(s, v)
}
func matchPrefixExclude(s, v Version) bool {
	return !matchPrefixMatch(s, v)
}

//
//
// Inclusive ordered comparison
// ----------------------------
//
// An inclusive ordered comparison clause includes a comparison operator and a
// version identifier, and will match any version where the comparison is correct
// based on the relative position of the candidate version and the specified
// version given the consistent ordering defined by the standard
// `Version scheme`_.
//
// The inclusive ordered comparison operators are ``<=`` and ``>=``.
//
// As with version matching, the release segment is zero padded as necessary to
// ensure the release segments are compared with the same length.
//
// Local version identifiers are NOT permitted in this version specifier.
func matchLE(s, v Version) bool {
	return s.Cmp(v) >= 0
}
func matchGE(s, v Version) bool {
	return s.Cmp(v) <= 0
}

//
//
// Exclusive ordered comparison
// ----------------------------
//
// The exclusive ordered comparisons ``>`` and ``<`` are similar to the inclusive
// ordered comparisons in that they rely on the relative position of the candidate
// version and the specified version given the consistent ordering defined by the
// standard `Version scheme`_. However, they specifically exclude pre-releases,
// post-releases, and local versions of the specified version.
//
// The exclusive ordered comparison ``>V`` **MUST NOT** allow a post-release
// of the given version unless ``V`` itself is a post release. You may mandate
// that releases are later than a particular post release, including additional
// post releases, by using ``>V.postN``. For example, ``>1.7`` will allow
// ``1.7.1`` but not ``1.7.0.post1`` and ``>1.7.post2`` will allow ``1.7.1``
// and ``1.7.0.post3`` but not ``1.7.0``.
//
// The exclusive ordered comparison ``>V`` **MUST NOT** match a local version of
// the specified version.
//
// The exclusive ordered comparison ``<V`` **MUST NOT** allow a pre-release of
// the specified version unless the specified version is itself a pre-release.
// Allowing pre-releases that are earlier than, but not equal to a specific
// pre-release may be accomplished by using ``<V.rc1`` or similar.
//
// As with version matching, the release segment is zero padded as necessary to
// ensure the release segments are compared with the same length.
//
// Local version identifiers are NOT permitted in this version specifier.
func matchLT(s, v Version) bool {
	return s.Cmp(v) > 0
}
func matchGT(s, v Version) bool {
	return s.Cmp(v) < 0
}

//
//
// Arbitrary equality
// ------------------
//
// Arbitrary equality comparisons are simple string equality operations which do
// not take into account any of the semantic information such as zero padding or
// local versions. This operator also does not support prefix matching as the
// ``==`` operator does.
//
// The primary use case for arbitrary equality is to allow for specifying a
// version which cannot otherwise be represented by this PEP. This operator is
// special and acts as an escape hatch to allow someone using a tool which
// implements this PEP to still install a legacy version which is otherwise
// incompatible with this PEP.
//
// An example would be ``===foobar`` which would match a version of ``foobar``.
//
// This operator may also be used to explicitly require an unpatched version
// of a project such as ``===1.0`` which would not match for a version
// ``1.0+downstream1``.
//
// Use of this operator is heavily discouraged and tooling MAY display a warning
// when it is used.
//
//
// Handling of pre-releases
// ------------------------
//
// Pre-releases of any kind, including developmental releases, are implicitly
// excluded from all version specifiers, *unless* they are already present
// on the system, explicitly requested by the user, or if the only available
// version that satisfies the version specifier is a pre-release.
//
// By default, dependency resolution tools SHOULD:
//
// * accept already installed pre-releases for all version specifiers
// * accept remotely available pre-releases for version specifiers where
//   there is no final or post release that satisfies the version specifier
// * exclude all other pre-releases from consideration
//
// Dependency resolution tools MAY issue a warning if a pre-release is needed
// to satisfy a version specifier.
//
// Dependency resolution tools SHOULD also allow users to request the
// following alternative behaviours:
//
// * accepting pre-releases for all version specifiers
// * excluding pre-releases for all version specifiers (reporting an error or
//   warning if a pre-release is already installed locally, or if a
//   pre-release is the only way to satisfy a particular specifier)
//
// Dependency resolution tools MAY also allow the above behaviour to be
// controlled on a per-distribution basis.
//
// Post-releases and final releases receive no special treatment in version
// specifiers - they are always included unless explicitly excluded.

type PreReleaseBehavior struct {
	AllowAll  bool
	AllowList []Version
}

func (prereleases *PreReleaseBehavior) allow(v Version) bool {
	if !v.IsPreRelease() {
		return true
	}
	if prereleases != nil {
		if prereleases.AllowAll {
			return true
		}
		for _, item := range prereleases.AllowList {
			if item.Cmp(v) == 0 {
				return true
			}
		}
	}
	return false
}

func (s Specifier) Select(choices []Version, prereleases *PreReleaseBehavior) *Version {
	var best *Version
	var bestPrerelease *Version
	for _, choice := range choices {
		if s.Match(choice) {
			if prereleases.allow(choice) {
				if best == nil || best.Cmp(choice) < 0 {
					v := choice
					best = &v
				}
			} else {
				if bestPrerelease == nil || bestPrerelease.Cmp(choice) < 0 {
					v := choice
					bestPrerelease = &v
				}
			}
		}
	}
	if best != nil {
		return best
	}
	if bestPrerelease != nil {
		return bestPrerelease
	}
	return nil
}

//
//
// Examples
// --------
//
// * ``~=3.1``: version 3.1 or later, but not version 4.0 or later.
// * ``~=3.1.2``: version 3.1.2 or later, but not version 3.2.0 or later.
// * ``~=3.1a1``: version 3.1a1 or later, but not version 4.0 or later.
// * ``== 3.1``: specifically version 3.1 (or 3.1.0), excludes all pre-releases,
//   post releases, developmental releases and any 3.1.x maintenance releases.
// * ``== 3.1.*``: any version that starts with 3.1. Equivalent to the
//   ``~=3.1.0`` compatible release clause.
// * ``~=3.1.0, != 3.1.3``: version 3.1.0 or later, but not version 3.1.3 and
//   not version 3.2.0 or later.
