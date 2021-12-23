package pep440

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// Version scheme
// ==============
//
// Distributions are identified by a public version identifier which
// supports all defined version comparison operations
//
// The version scheme is used both to describe the distribution version
// provided by a particular distribution archive, as well as to place
// constraints on the version of dependencies needed in order to build or
// run the software.
//
//

type Version = LocalVersion

// Public version identifiers
// --------------------------
//
// The canonical public version identifiers MUST comply with the following
// scheme::
//
//     [N!]N(.N)*[{a|b|rc}N][.postN][.devN]
//
// Public version identifiers MUST NOT include leading or trailing whitespace.
//
// Public version identifiers MUST be unique within a given distribution.
//
// Installation tools SHOULD ignore any public versions which do not comply with
// this scheme but MUST also include the normalizations specified below.
// Installation tools MAY warn the user when non-compliant or ambiguous versions
// are detected.
//
// See also `Appendix B : Parsing version strings with regular expressions` which
// provides a regular expression to check strict conformance with the canonical
// format, as well as a more permissive regular expression accepting inputs that
// may require subsequent normalization.

// ParseVersion parses a string to a Version object, performing normalization.
func ParseVersion(str string) (*Version, error) {
	ver, err := parseVersion(str) // the routine from Appendix B
	if err != nil {
		return nil, fmt.Errorf("pep440.ParseVersion: %w", err)
	}
	return ver, nil
}

//
// Public version identifiers are separated into up to five segments:
//

type PublicVersion struct {
	// * Epoch segment: ``N!``
	Epoch int
	// * Release segment: ``N(.N)*``
	Release []int
	// * Pre-release segment: ``{a|b|rc}N``
	Pre *PreRelease
	// * Post-release segment: ``.postN``
	Post *int
	// * Development release segment: ``.devN``
	Dev *int
}

type PreRelease struct {
	L string
	N int
}

// GoString implements fmt.GoStringer.
func (ver PublicVersion) GoString() string {
	pre := "nil"
	if ver.Pre != nil {
		pre = fmt.Sprintf("&%#v", *ver.Pre)
	}
	post := "nil"
	if ver.Post != nil {
		post = fmt.Sprintf("intPtr(%#v)", *ver.Post)
	}
	dev := "nil"
	if ver.Dev != nil {
		dev = fmt.Sprintf("intPtr(%#v)", *ver.Dev)
	}
	return fmt.Sprintf("pep440.PublicVersion{Epoch:%d, Release:%#v, Pre:%s, Post:%s, Dev:%s}",
		ver.Epoch, ver.Release, pre, post, dev)
}

func (ver PublicVersion) writeTo(ret *strings.Builder) {
	if ver.Epoch > 0 {
		fmt.Fprintf(ret, "%d!", ver.Epoch)
	}
	if len(ver.Release) == 0 {
		panic("invalid version: no release segments")
	}
	fmt.Fprintf(ret, "%d", ver.Release[0])
	for _, segment := range ver.Release[1:] {
		fmt.Fprintf(ret, ".%d", segment)
	}
	if ver.Pre != nil {
		fmt.Fprintf(ret, "%s%d", ver.Pre.L, ver.Pre.N)
	}
	if ver.Post != nil {
		fmt.Fprintf(ret, ".post%d", *ver.Post)
	}
	if ver.Dev != nil {
		fmt.Fprintf(ret, ".dev%d", *ver.Dev)
	}
}

// String implements fmt.Stringer.  String does not perform any normalization.
func (ver PublicVersion) String() string {
	var ret strings.Builder
	ver.writeTo(&ret)
	return ret.String()
}

//
// Any given release will be a "final release", "pre-release", "post-release" or
// "developmental release" as defined in the following sections.
//
// All numeric components MUST be non-negative integers represented as sequences
// of ASCII digits.
//
// All numeric components MUST be interpreted and ordered according to their
// numeric value, not as text strings.
//
// All numeric components MAY be zero. Except as described below for the
// release segment, a numeric component of zero has no special significance
// aside from always being the lowest possible value in the version ordering.
//
// .. note::
//
//    Some hard to read version identifiers are permitted by this scheme in
//    order to better accommodate the wide range of versioning practices
//    across existing public and private Python projects.
//
//    Accordingly, some of the versioning practices which are technically
//    permitted by the PEP are strongly discouraged for new projects. Where
//    this is the case, the relevant details are noted in the following
//    sections.
//
//
// Local version identifiers
// -------------------------
//
// Local version identifiers MUST comply with the following scheme::
//
//     <public version identifier>[+<local version label>]
//
// They consist of a normal public version identifier (as defined in the
// previous section), along with an arbitrary "local version label", separated
// from the public version identifier by a plus. Local version labels have
// no specific semantics assigned, but some syntactic restrictions are imposed.
//
// Local version identifiers are used to denote fully API (and, if applicable,
// ABI) compatible patched versions of upstream projects. For example, these
// may be created by application developers and system integrators by applying
// specific backported bug fixes when upgrading to a new upstream release would
// be too disruptive to the application or other integrated system (such as a
// Linux distribution).
//
// The inclusion of the local version label makes it possible to differentiate
// upstream releases from potentially altered rebuilds by downstream
// integrators. The use of a local version identifier does not affect the kind
// of a release but, when applied to a source distribution, does indicate that
// it may not contain the exact same code as the corresponding upstream release.
//
// To ensure local version identifiers can be readily incorporated as part of
// filenames and URLs, and to avoid formatting inconsistencies in hexadecimal
// hash representations, local version labels MUST be limited to the following
// set of permitted characters:
//
// * ASCII letters (``[a-zA-Z]``)
// * ASCII digits (``[0-9]``)
// * periods (``.``)
//
// Local version labels MUST start and end with an ASCII letter or digit.
//

type LocalVersion struct {
	PublicVersion
	Local []intstr.IntOrString
}

// GoString implements fmt.GoStringer.
func (ver LocalVersion) GoString() string {
	return fmt.Sprintf("pep440.LocalVersion{PublicVersion:%#v, Local:%#v}",
		ver.PublicVersion, ver.Local)
}

// String implements fmt.Stringer.  String does not perform any normalization.
func (ver LocalVersion) String() string {
	var ret strings.Builder
	ver.PublicVersion.writeTo(&ret)
	sep := "+"
	for _, local := range ver.Local {
		ret.WriteString(sep)
		ret.WriteString(local.String())
		sep = "."
	}
	return ret.String()
}

// Comparison and ordering of local versions considers each segment of the local
// version (divided by a ``.``) separately. If a segment consists entirely of
// ASCII digits then that section should be considered an integer for comparison
// purposes and if a segment contains any ASCII letters then that segment is
// compared lexicographically with case insensitivity. When comparing a numeric
// and lexicographic segment, the numeric section always compares as greater than
// the lexicographic segment. Additionally a local version with a great number of
// segments will always compare as greater than a local version with fewer
// segments, as long as the shorter local version's segments match the beginning
// of the longer local version's segments exactly.

func cmpLocalSegment(a, b *intstr.IntOrString) int {
	// handle one or both of them being nil
	switch {
	case a == nil && b == nil:
		panic("should not happen: cmpLocal shouldn't have bothered calling this")
	case a == nil && b != nil:
		return -1
	case a != nil && b == nil:
		return 1
	}
	switch {
	case a.Type == intstr.Int && b.Type == intstr.Int:
		return int(a.IntVal - b.IntVal)
	case a.Type == intstr.String && b.Type == intstr.String:
		switch {
		case a.StrVal < b.StrVal:
			return -1
		case a.StrVal > b.StrVal:
			return 1
		}
		return 0
	case a.Type == intstr.Int && b.Type == intstr.String:
		return 1
	case a.Type == intstr.String && b.Type == intstr.Int:
		return -1
	default:
		panic("should not happen: invalid intstr.IntOrString")
	}
}

func cmpLocal(a, b LocalVersion) int {
	for i := 0; i < len(a.Local) || i < len(b.Local); i++ {
		var aSeg, bSeg *intstr.IntOrString
		if i < len(a.Local) {
			aSeg = &(a.Local[i])
		}
		if i < len(b.Local) {
			bSeg = &(b.Local[i])
		}
		if d := cmpLocalSegment(aSeg, bSeg); d != 0 {
			return d
		}
	}
	return 0
}

// Cmp returns a number < 0 if version 'a' is less than version 'b', > 0 if 'a' is greater than 'b',
// or 0 if they are equal.  This is similar to the C-language strcmp.  You may think of this as
// returning the result of arithmetic subtraction "a-b"; though only the sign is defined; the
// magnitude may be anything.
func (a LocalVersion) Cmp(b LocalVersion) int {
	if d := a.PublicVersion.Cmp(b.PublicVersion); d != 0 {
		return d
	}
	return cmpLocal(a, b)
}

//
// An "upstream project" is a project that defines its own public versions. A
// "downstream project" is one which tracks and redistributes an upstream project,
// potentially backporting security and bug fixes from later versions of the
// upstream project.
//
// Local version identifiers SHOULD NOT be used when publishing upstream
// projects to a public index server, but MAY be used to identify private
// builds created directly from the project source. Local
// version identifiers SHOULD be used by downstream projects when releasing a
// version that is API compatible with the version of the upstream project
// identified by the public version identifier, but contains additional changes
// (such as bug fixes). As the Python Package Index is intended solely for
// indexing and hosting upstream projects, it MUST NOT allow the use of local
// version identifiers.
//
// Source distributions using a local version identifier SHOULD provide the
// ``python.integrator`` extension metadata (as defined in :pep:`459`).
//
//
// Final releases
// --------------
//
// A version identifier that consists solely of a release segment and optionally
// an epoch identifier is termed a "final release".

func (ver PublicVersion) IsFinal() bool {
	return ver.Pre == nil && ver.Post == nil && ver.Dev == nil
}

func (ver LocalVersion) IsFinal() bool {
	return ver.PublicVersion.IsFinal() && len(ver.Local) == 0
}

//
// The release segment consists of one or more non-negative integer
// values, separated by dots::
//
//     N(.N)*
//
// Final releases within a project MUST be numbered in a consistently
// increasing fashion, otherwise automated tools will not be able to upgrade
// them correctly.
//
// Comparison and ordering of release segments considers the numeric value
// of each component of the release segment in turn. When comparing release
// segments with different numbers of components, the shorter segment is
// padded out with additional zeros as necessary.

func (ver PublicVersion) releaseSegment(n int) int {
	if n < len(ver.Release) {
		return ver.Release[n]
	}
	return 0
}

func cmpRelease(a, b PublicVersion) int {
	for i := 0; i < len(a.Release) || i < len(b.Release); i++ {
		if diff := a.releaseSegment(i) - b.releaseSegment(i); diff != 0 {
			return diff
		}
	}
	return 0
}

//
// While any number of additional components after the first are permitted
// under this scheme, the most common variants are to use two components
// ("major.minor") or three components ("major.minor.micro").

func (ver PublicVersion) Major() int { return ver.releaseSegment(0) }
func (ver PublicVersion) Minor() int { return ver.releaseSegment(1) }
func (ver PublicVersion) Micro() int { return ver.releaseSegment(2) }

//
// For example::
//
//     0.9
//     0.9.1
//     0.9.2
//     ...
//     0.9.10
//     0.9.11
//     1.0
//     1.0.1
//     1.1
//     2.0
//     2.0.1
//     ...
//
// A release series is any set of final release numbers that start with a
// common prefix. For example, ``3.3.1``, ``3.3.5`` and ``3.3.9.45`` are all
// part of the ``3.3`` release series.
//
// .. note::
//
//    ``X.Y`` and ``X.Y.0`` are not considered distinct release numbers, as
//    the release segment comparison rules implicit expand the two component
//    form to ``X.Y.0`` when comparing it to any release segment that includes
//    three components.
//
// Date based release segments are also permitted. An example of a date based
// release scheme using the year and month of the release::
//
//     2012.4
//     2012.7
//     2012.10
//     2013.1
//     2013.6
//     ...
//
//
// Pre-releases
// ------------
//
// Some projects use an "alpha, beta, release candidate" pre-release cycle to
// support testing by their users prior to a final release.
//
// If used as part of a project's development cycle, these pre-releases are
// indicated by including a pre-release segment in the version identifier::
//
//     X.YaN   # Alpha release
//     X.YbN   # Beta release
//     X.YrcN  # Release Candidate
//     X.Y     # Final release
//
// A version identifier that consists solely of a release segment and a
// pre-release segment is termed a "pre-release".
//
// The pre-release segment consists of an alphabetical identifier for the
// pre-release phase, along with a non-negative integer value. Pre-releases for
// a given release are ordered first by phase (alpha, beta, release candidate)
// and then by the numerical component within that phase.
//
// Installation tools MAY accept both ``c`` and ``rc`` releases for a common
// release segment in order to handle some existing legacy distributions.
//
// Installation tools SHOULD interpret ``c`` versions as being equivalent to
// ``rc`` versions (that is, ``c1`` indicates the same version as ``rc1``).
//
// Build tools, publication tools and index servers SHOULD disallow the creation
// of both ``rc`` and ``c`` releases for a common release segment.
//
//

//nolint:gochecknoglobals // Would be 'const'.
var preReleaseOrder = map[string]int{
	"a":     -3,
	"alpha": -3,

	"b":    -2,
	"beta": -2,

	"rc":      -1,
	"c":       -1,
	"pre":     -1,
	"preview": -1,

	// absent: 0,
}

func cmpPreRelease(a, b PublicVersion) int {
	var aL, aN, bL, bN int
	var ok bool
	if a.Pre != nil {
		aL, ok = preReleaseOrder[a.Pre.L]
		if !ok {
			panic(fmt.Errorf("invalid pre-release string: %q", a.Pre.L))
		}
		aN = a.Pre.N
	} else if a.Dev != nil && a.Post == nil {
		aL = -4
	}
	if b.Pre != nil {
		bL, ok = preReleaseOrder[b.Pre.L]
		if !ok {
			panic(fmt.Errorf("invalid pre-release string: %q", b.Pre.L))
		}
		bN = b.Pre.N
	} else if b.Dev != nil && b.Post == nil {
		bL = -4
	}
	if aL != bL {
		return aL - bL
	}
	return aN - bN
}

// Post-releases
// -------------
//
// Some projects use post-releases to address minor errors in a final release
// that do not affect the distributed software (for example, correcting an error
// in the release notes).
//
// If used as part of a project's development cycle, these post-releases are
// indicated by including a post-release segment in the version identifier::
//
//     X.Y.postN    # Post-release
//
// A version identifier that includes a post-release segment without a
// developmental release segment is termed a "post-release".
//
// The post-release segment consists of the string ``.post``, followed by a
// non-negative integer value. Post-releases are ordered by their
// numerical component, immediately following the corresponding release,
// and ahead of any subsequent release.
//
// .. note::
//
//    The use of post-releases to publish maintenance releases containing
//    actual bug fixes is strongly discouraged. In general, it is better
//    to use a longer release number and increment the final component
//    for each maintenance release.
//
// Post-releases are also permitted for pre-releases::
//
//     X.YaN.postM   # Post-release of an alpha release
//     X.YbN.postM   # Post-release of a beta release
//     X.YrcN.postM  # Post-release of a release candidate
//
// .. note::
//
//    Creating post-releases of pre-releases is strongly discouraged, as
//    it makes the version identifier difficult to parse for human readers.
//    In general, it is substantially clearer to simply create a new
//    pre-release by incrementing the numeric component.

func cmpPostRelease(a, b PublicVersion) int {
	aPost := -1
	if a.Post != nil {
		aPost = *a.Post
	}
	bPost := -1
	if b.Post != nil {
		bPost = *b.Post
	}
	return aPost - bPost
}

//
//
// Developmental releases
// ----------------------
//
// Some projects make regular developmental releases, and system packagers
// (especially for Linux distributions) may wish to create early releases
// directly from source control which do not conflict with later project
// releases.
//
// If used as part of a project's development cycle, these developmental
// releases are indicated by including a developmental release segment in the
// version identifier::
//
//     X.Y.devN    # Developmental release
//
// A version identifier that includes a developmental release segment is
// termed a "developmental release".
//
// The developmental release segment consists of the string ``.dev``,
// followed by a non-negative integer value. Developmental releases are ordered
// by their numerical component, immediately before the corresponding release
// (and before any pre-releases with the same release segment), and following
// any previous release (including any post-releases).
//
// Developmental releases are also permitted for pre-releases and
// post-releases::
//
//     X.YaN.devM       # Developmental release of an alpha release
//     X.YbN.devM       # Developmental release of a beta release
//     X.YrcN.devM      # Developmental release of a release candidate
//     X.Y.postN.devM   # Developmental release of a post-release
//
// .. note::
//
//    While they may be useful for continuous integration purposes, publishing
//    developmental releases of pre-releases to general purpose public index
//    servers is strongly discouraged, as it makes the version identifier
//    difficult to parse for human readers. If such a release needs to be
//    published, it is substantially clearer to instead create a new
//    pre-release by incrementing the numeric component.
//
//    Developmental releases of post-releases are also strongly discouraged,
//    but they may be appropriate for projects which use the post-release
//    notation for full maintenance releases which may include code changes.
//
//

func (ver PublicVersion) IsPreRelease() bool {
	return ver.Pre != nil || ver.Dev != nil
}

func cmpDevRelease(a, b PublicVersion) int {
	switch {
	case a.Dev == nil && b.Dev == nil:
		return 0
	case a.Dev == nil && b.Dev != nil:
		return 1
	case a.Dev != nil && b.Dev == nil:
		return -1
	default:
		return (*a.Dev) - (*b.Dev)
	}
}

// Version epochs
// --------------
//
// If included in a version identifier, the epoch appears before all other
// components, separated from the release segment by an exclamation mark::
//
//     E!X.Y  # Version identifier with epoch
//
// If no explicit epoch is given, the implicit epoch is ``0``.
//
// Most version identifiers will not include an epoch, as an explicit epoch is
// only needed if a project *changes* the way it handles version numbering in
// a way that means the normal version ordering rules will give the wrong
// answer. For example, if a project is using date based versions like
// ``2014.04`` and would like to switch to semantic versions like ``1.0``, then
// the new releases would be identified as *older* than the date based releases
// when using the normal sorting scheme::
//
//     1.0
//     1.1
//     2.0
//     2013.10
//     2014.04
//
// However, by specifying an explicit epoch, the sort order can be changed
// appropriately, as all versions from a later epoch are sorted after versions
// from an earlier epoch::
//
//     2013.10
//     2014.04
//     1!1.0
//     1!1.1
//     1!2.0

func cmpEpoch(a, b PublicVersion) int {
	return a.Epoch - b.Epoch
}

//
// Normalization
// -------------
//
// In order to maintain better compatibility with existing versions there are a
// number of "alternative" syntaxes that MUST be taken into account when parsing
// versions. These syntaxes MUST be considered when parsing a version, however
// they should be "normalized" to the standard syntax defined above.

func (ver PublicVersion) Normalize() (*PublicVersion, error) {
	n, err := ParseVersion(ver.String())
	if err != nil {
		return nil, err
	}
	return &n.PublicVersion, nil
}

func (ver LocalVersion) Normalize() (*LocalVersion, error) {
	n, err := ParseVersion(ver.String())
	if err != nil {
		return nil, err
	}
	return n, nil
}

//
//
// Case sensitivity
// ~~~~~~~~~~~~~~~~
//
// All ascii letters should be interpreted case insensitively within a version and
// the normal form is lowercase. This allows versions such as ``1.1RC1`` which
// would be normalized to ``1.1rc1``.
//
//
// Integer Normalization
// ~~~~~~~~~~~~~~~~~~~~~
//
// All integers are interpreted via the ``int()`` built in and normalize to the
// string form of the output. This means that an integer version of ``00`` would
// normalize to ``0`` while ``09000`` would normalize to ``9000``. This does not
// hold true for integers inside of an alphanumeric segment of a local version
// such as ``1.0+foo0100`` which is already in its normalized form.
//
//
// Pre-release separators
// ~~~~~~~~~~~~~~~~~~~~~~
//
// Pre-releases should allow a ``.``, ``-``, or ``_`` separator between the
// release segment and the pre-release segment. The normal form for this is
// without a separator. This allows versions such as ``1.1.a1`` or ``1.1-a1``
// which would be normalized to ``1.1a1``. It should also allow a separator to
// be used between the pre-release signifier and the numeral. This allows versions
// such as ``1.0a.1`` which would be normalized to ``1.0a1``.
//
//
// Pre-release spelling
// ~~~~~~~~~~~~~~~~~~~~
//
// Pre-releases allow the additional spellings of ``alpha``, ``beta``, ``c``,
// ``pre``, and ``preview`` for ``a``, ``b``, ``rc``, ``rc``, and ``rc``
// respectively. This allows versions such as ``1.1alpha1``, ``1.1beta2``, or
// ``1.1c3`` which normalize to ``1.1a1``, ``1.1b2``, and ``1.1rc3``. In every
// case the additional spelling should be considered equivalent to their normal
// forms.
//
//
// Implicit pre-release number
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~
//
// Pre releases allow omitting the numeral in which case it is implicitly assumed
// to be ``0``. The normal form for this is to include the ``0`` explicitly. This
// allows versions such as ``1.2a`` which is normalized to ``1.2a0``.
//
//
// Post release separators
// ~~~~~~~~~~~~~~~~~~~~~~~
//
// Post releases allow a ``.``, ``-``, or ``_`` separator as well as omitting the
// separator all together. The normal form of this is with the ``.`` separator.
// This allows versions such as ``1.2-post2`` or ``1.2post2`` which normalize to
// ``1.2.post2``. Like the pre-release separator this also allows an optional
// separator between the post release signifier and the numeral. This allows
// versions like ``1.2.post-2`` which would normalize to ``1.2.post2``.
//
//
// Post release spelling
// ~~~~~~~~~~~~~~~~~~~~~
//
// Post-releases allow the additional spellings of ``rev`` and ``r``. This allows
// versions such as ``1.0-r4`` which normalizes to ``1.0.post4``. As with the
// pre-releases the additional spellings should be considered equivalent to their
// normal forms.
//
//
// Implicit post release number
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~
//
// Post releases allow omitting the numeral in which case it is implicitly assumed
// to be ``0``. The normal form for this is to include the ``0`` explicitly. This
// allows versions such as ``1.2.post`` which is normalized to ``1.2.post0``.
//
//
// Implicit post releases
// ~~~~~~~~~~~~~~~~~~~~~~
//
// Post releases allow omitting the ``post`` signifier all together. When using
// this form the separator MUST be ``-`` and no other form is allowed. This allows
// versions such as ``1.0-1`` to be normalized to ``1.0.post1``. This particular
// normalization MUST NOT be used in conjunction with the implicit post release
// number rule. In other words, ``1.0-`` is *not* a valid version and it does *not*
// normalize to ``1.0.post0``.
//
//
// Development release separators
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
//
// Development releases allow a ``.``, ``-``, or a ``_`` separator as well as
// omitting the separator all together. The normal form of this is with the ``.``
// separator. This allows versions such as ``1.2-dev2`` or ``1.2dev2`` which
// normalize to ``1.2.dev2``.
//
//
// Implicit development release number
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
//
// Development releases allow omitting the numeral in which case it is implicitly
// assumed to be ``0``. The normal form for this is to include the ``0``
// explicitly. This allows versions such as ``1.2.dev`` which is normalized to
// ``1.2.dev0``.
//
//
// Local version segments
// ~~~~~~~~~~~~~~~~~~~~~~
//
// With a local version, in addition to the use of ``.`` as a separator of
// segments, the use of ``-`` and ``_`` is also acceptable. The normal form is
// using the ``.`` character. This allows versions such as ``1.0+ubuntu-1`` to be
// normalized to ``1.0+ubuntu.1``.
//
//
// Preceding v character
// ~~~~~~~~~~~~~~~~~~~~~
//
// In order to support the common version notation of ``v1.0`` versions may be
// preceded by a single literal ``v`` character. This character MUST be ignored
// for all purposes and should be omitted from all normalized forms of the
// version. The same version with and without the ``v`` is considered equivalent.
//
//
// Leading and Trailing Whitespace
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
//
// Leading and trailing whitespace must be silently ignored and removed from all
// normalized forms of a version. This includes ``" "``, ``\t``, ``\n``, ``\r``,
// ``\f``, and ``\v``. This allows accidental whitespace to be handled sensibly,
// such as a version like ``1.0\n`` which normalizes to ``1.0``.
//
//
// Examples of compliant version schemes
// -------------------------------------
//
// The standard version scheme is designed to encompass a wide range of
// identification practices across public and private Python projects. In
// practice, a single project attempting to use the full flexibility offered
// by the scheme would create a situation where human users had difficulty
// figuring out the relative order of versions, even though the rules above
// ensure all compliant tools will order them consistently.
//
// The following examples illustrate a small selection of the different
// approaches projects may choose to identify their releases, while still
// ensuring that the "latest release" and the "latest stable release" can
// be easily determined, both by human users and automated tools.
//
// Simple "major.minor" versioning::
//
//     0.1
//     0.2
//     0.3
//     1.0
//     1.1
//     ...
//
// Simple "major.minor.micro" versioning::
//
//     1.1.0
//     1.1.1
//     1.1.2
//     1.2.0
//     ...
//
// "major.minor" versioning with alpha, beta and candidate
// pre-releases::
//
//     0.9
//     1.0a1
//     1.0a2
//     1.0b1
//     1.0rc1
//     1.0
//     1.1a1
//     ...
//
// "major.minor" versioning with developmental releases, release candidates
// and post-releases for minor corrections::
//
//     0.9
//     1.0.dev1
//     1.0.dev2
//     1.0.dev3
//     1.0.dev4
//     1.0c1
//     1.0c2
//     1.0
//     1.0.post1
//     1.1.dev1
//     ...
//
// Date based releases, using an incrementing serial within each year, skipping
// zero::
//
//     2012.1
//     2012.2
//     2012.3
//     ...
//     2012.15
//     2013.1
//     2013.2
//     ...
//
//
// Summary of permitted suffixes and relative ordering
// ---------------------------------------------------
//
// .. note::
//
//    This section is intended primarily for authors of tools that
//    automatically process distribution metadata, rather than developers
//    of Python distributions deciding on a versioning scheme.
//

// Cmp returns a number < 0 if version 'a' is less than version 'b', > 0 if 'a' is greater than 'b',
// or 0 if they are equal.  This is similar to the C-language strcmp.  You may think of this as
// returning the result of arithmetic subtraction "a-b"; though only the sign is defined; the
// magnitude may be anything.
func (a PublicVersion) Cmp(b PublicVersion) int {
	// The epoch segment of version identifiers MUST be sorted according to the
	// numeric value of the given epoch. If no epoch segment is present, the
	// implicit numeric value is ``0``.
	if d := cmpEpoch(a, b); d != 0 {
		return d
	}
	//
	// The release segment of version identifiers MUST be sorted in
	// the same order as Python's tuple sorting when the normalized release segment is
	// parsed as follows::
	//
	//     tuple(map(int, release_segment.split(".")))
	//
	// All release segments involved in the comparison MUST be converted to a
	// consistent length by padding shorter segments with zeros as needed.
	if d := cmpRelease(a, b); d != 0 {
		return d
	}
	//
	// Within a numeric release (``1.0``, ``2.7.3``), the following suffixes
	// are permitted and MUST be ordered as shown::
	//
	//    .devN, aN, bN, rcN, <no suffix>, .postN
	//
	// Note that `c` is considered to be semantically equivalent to `rc` and must be
	// sorted as if it were `rc`. Tools MAY reject the case of having the same ``N``
	// for both a ``c`` and a ``rc`` in the same release segment as ambiguous and
	// remain in compliance with the PEP.
	if d := cmpPreRelease(a, b); d != 0 {
		return d
	}
	//
	// Within an alpha (``1.0a1``), beta (``1.0b1``), or release candidate
	// (``1.0rc1``, ``1.0c1``), the following suffixes are permitted and MUST be
	// ordered as shown::
	//
	//    .devN, <no suffix>, .postN
	if d := cmpPostRelease(a, b); d != 0 {
		return d
	}
	//
	// Within a post-release (``1.0.post1``), the following suffixes are permitted
	// and MUST be ordered as shown::
	//
	//     .devN, <no suffix>
	if d := cmpDevRelease(a, b); d != 0 {
		return d
	}
	//
	// Note that ``devN`` and ``postN`` MUST always be preceded by a dot, even
	// when used immediately following a numeric version (e.g. ``1.0.dev456``,
	// ``1.0.post1``).
	//
	// Within a pre-release, post-release or development release segment with a
	// shared prefix, ordering MUST be by the value of the numeric component.
	return 0
}

//
// The following example covers many of the possible combinations::
//
//     1.0.dev456
//     1.0a1
//     1.0a2.dev456
//     1.0a12.dev456
//     1.0a12
//     1.0b1.dev456
//     1.0b2
//     1.0b2.post345.dev456
//     1.0b2.post345
//     1.0rc1.dev456
//     1.0rc1
//     1.0
//     1.0+abc.5
//     1.0+abc.7
//     1.0+5
//     1.0.post456.dev34
//     1.0.post456
//     1.1.dev1
//
//
// Version ordering across different metadata versions
// ---------------------------------------------------
//
// Metadata v1.0 (PEP 241) and metadata v1.1 (PEP 314) do not specify a standard
// version identification or ordering scheme. However metadata v1.2 (PEP 345)
// does specify a scheme which is defined in PEP 386.
//
// Due to the nature of the simple installer API it is not possible for an
// installer to be aware of which metadata version a particular distribution was
// using. Additionally installers required the ability to create a reasonably
// prioritized list that includes all, or as many as possible, versions of
// a project to determine which versions it should install. These requirements
// necessitate a standardization across one parsing mechanism to be used for all
// versions of a project.
//
// Due to the above, this PEP MUST be used for all versions of metadata and
// supersedes PEP 386 even for metadata v1.2. Tools SHOULD ignore any versions
// which cannot be parsed by the rules in this PEP, but MAY fall back to
// implementation defined version parsing and ordering schemes if no versions
// complying with this PEP are available.
//
// Distribution users may wish to explicitly remove non-compliant versions from
// any private package indexes they control.
//
//
// Compatibility with other version schemes
// ----------------------------------------
//
// Some projects may choose to use a version scheme which requires
// translation in order to comply with the public version scheme defined in
// this PEP. In such cases, the project specific version can be stored in the
// metadata while the translated public version is published in the version field.
//
// This allows automated distribution tools to provide consistently correct
// ordering of published releases, while still allowing developers to use
// the internal versioning scheme they prefer for their projects.
//
//
// Semantic versioning
// ~~~~~~~~~~~~~~~~~~~
//
// `Semantic versioning`_ is a popular version identification scheme that is
// more prescriptive than this PEP regarding the significance of different
// elements of a release number. Even if a project chooses not to abide by
// the details of semantic versioning, the scheme is worth understanding as
// it covers many of the issues that can arise when depending on other
// distributions, and when publishing a distribution that others rely on.
//
// The "Major.Minor.Patch" (described in this PEP as "major.minor.micro")
// aspects of semantic versioning (clauses 1-8 in the 2.0.0 specification)
// are fully compatible with the version scheme defined in this PEP, and abiding
// by these aspects is encouraged.
//
// Semantic versions containing a hyphen (pre-releases - clause 10) or a
// plus sign (builds - clause 11) are *not* compatible with this PEP
// and are not permitted in the public version field.
//
// One possible mechanism to translate such semantic versioning based source
// labels to compatible public versions is to use the ``.devN`` suffix to
// specify the appropriate version order.
//
// Specific build information may also be included in local version labels.
//
// .. _Semantic versioning: http://semver.org/
//
//
// DVCS based version labels
// ~~~~~~~~~~~~~~~~~~~~~~~~~
//
// Many build tools integrate with distributed version control systems like
// Git and Mercurial in order to add an identifying hash to the version
// identifier. As hashes cannot be ordered reliably such versions are not
// permitted in the public version field.
//
// As with semantic versioning, the public ``.devN`` suffix may be used to
// uniquely identify such releases for publication, while the original DVCS based
// label can be stored in the project metadata.
//
// Identifying hash information may also be included in local version labels.
//
//
// Olson database versioning
// ~~~~~~~~~~~~~~~~~~~~~~~~~
//
// The ``pytz`` project inherits its versioning scheme from the corresponding
// Olson timezone database versioning scheme: the year followed by a lowercase
// character indicating the version of the database within that year.
//
// This can be translated to a compliant public version identifier as
// ``<year>.<serial>``, where the serial starts at zero or one (for the
// '<year>a' release) and is incremented with each subsequent database
// update within the year.
//
// As with other translated version identifiers, the corresponding Olson
// database version could be recorded in the project metadata.
