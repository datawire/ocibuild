// Copyright (C) 2021  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package pep440

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// Direct references
// =================
//
// Some automated tools may permit the use of a direct reference as an
// alternative to a normal version specifier. A direct reference consists of
// the specifier ``@`` and an explicit URL.
//
// Whether or not direct references are appropriate depends on the specific
// use case for the version specifier. Automated tools SHOULD at least issue
// warnings and MAY reject them entirely when direct references are used
// inappropriately.
//
// Public index servers SHOULD NOT allow the use of direct references in
// uploaded distributions. Direct references are intended as a tool for
// software integrators rather than publishers.
//
// Depending on the use case, some appropriate targets for a direct URL
// reference may be an sdist or a wheel binary archive. The exact URLs and
// targets supported will be tool dependent.
//
// For example, a local source archive may be referenced directly::
//
//     pip @ file:///localbuilds/pip-1.3.1.zip
//
// Alternatively, a prebuilt archive may also be referenced::
//
//     pip @ file:///localbuilds/pip-1.3.1-py33-none-any.whl
//
// All direct references that do not refer to a local file URL SHOULD specify
// a secure transport mechanism (such as ``https``) AND include an expected
// hash value in the URL for verification purposes. If a direct reference is
// specified without any hash information, with hash information that the
// tool doesn't understand, or with a selected hash algorithm that the tool
// considers too weak to trust, automated tools SHOULD at least emit a warning
// and MAY refuse to rely on the URL. If such a direct reference also uses an
// insecure transport, automated tools SHOULD NOT rely on the URL.
//
// It is RECOMMENDED that only hashes which are unconditionally provided by
// the latest version of the standard library's ``hashlib`` module be used
// for source archive hashes. At time of writing, that list consists of
// ``'md5'``, ``'sha1'``, ``'sha224'``, ``'sha256'``, ``'sha384'``, and
// ``'sha512'``.
//
// For source archive and wheel references, an expected hash value may be
// specified by including a ``<hash-algorithm>=<expected-hash>`` entry as
// part of the URL fragment.
//
// For version control references, the ``VCS+protocol`` scheme SHOULD be
// used to identify both the version control system and the secure transport,
// and a version control system with hash based commit identifiers SHOULD be
// used. Automated tools MAY omit warnings about missing hashes for version
// control systems that do not provide hash based commit identifiers.
//
// To handle version control systems that do not support including commit or
// tag references directly in the URL, that information may be appended to the
// end of the URL using the ``@<commit-hash>`` or the ``@<tag>#<commit-hash>``
// notation.
//
// .. note::
//
//    This isn't *quite* the same as the existing VCS reference notation
//    supported by pip. Firstly, the distribution name is moved in front rather
//    than embedded as part of the URL. Secondly, the commit hash is included
//    even when retrieving based on a tag, in order to meet the requirement
//    above that *every* link should include a hash to make things harder to
//    forge (creating a malicious repo with a particular tag is easy, creating
//    one with a specific *hash*, less so).
//
// Remote URL examples::
//
//     pip @ https://github.com/pypa/pip/archive/1.3.1.zip#sha1=da9234ee9982d4bbb3c72346a6de940a148ea686
//     pip @ git+https://github.com/pypa/pip.git@7921be1537eac1e97bc40179a57f0349c2aee67d
//     pip @ git+https://github.com/pypa/pip.git@1.3.1#7921be1537eac1e97bc40179a57f0349c2aee67d
//
//
// File URLs
// ---------
//
// File URLs take the form of ``file://<host>/<path>``. If the ``<host>`` is
// omitted it is assumed to be ``localhost`` and even if the ``<host>`` is omitted
// the third slash MUST still exist. The ``<path>`` defines what the file path on
// the filesystem that is to be accessed.
//
// On the various \*nix operating systems the only allowed values for ``<host>``
// is for it to be omitted, ``localhost``, or another FQDN that the current
// machine believes matches its own host. In other words, on \*nix the ``file://``
// scheme can only be used to access paths on the local machine.
//
// On Windows the file format should include the drive letter if applicable as
// part of the ``<path>`` (e.g. ``file:///c:/path/to/a/file``). Unlike \*nix on
// Windows the ``<host>`` parameter may be used to specify a file residing on a
// network share. In other words, in order to translate ``\\machine\volume\file``
// to a ``file://`` url, it would end up as ``file://machine/volume/file``. For
// more information on ``file://`` URLs on Windows see MSDN [4]_.
//
//
// Updating the versioning specification
// =====================================
//
// The versioning specification may be updated with clarifications without
// requiring a new PEP or a change to the metadata version.
//
// Any technical changes that impact the version identification and comparison
// syntax and semantics would require an updated versioning scheme to be
// defined in a new PEP.
//
//
// Summary of differences from pkg_resources.parse_version
// =======================================================
//
// * Local versions sort differently, this PEP requires that they sort as greater
//   than the same version without a local version, whereas
//   ``pkg_resources.parse_version`` considers it a pre-release marker.
//
// * This PEP purposely restricts the syntax which constitutes a valid version
//   while ``pkg_resources.parse_version`` attempts to provide some meaning from
//   *any* arbitrary string.
//
// * ``pkg_resources.parse_version`` allows arbitrarily deeply nested version
//   signifiers like ``1.0.dev1.post1.dev5``. This PEP however allows only a
//   single use of each type and they must exist in a certain order.
//
//
// Summary of differences from \PEP 386
// ====================================
//
// * Moved the description of version specifiers into the versioning PEP
//
// * Added the "direct reference" concept as a standard notation for direct
//   references to resources (rather than each tool needing to invent its own)
//
// * Added the "local version identifier" and "local version label" concepts to
//   allow system integrators to indicate patched builds in a way that is
//   supported by the upstream tools, as well as to allow the incorporation of
//   build tags into the versioning of binary distributions.
//
// * Added the "compatible release" clause
//
// * Added the trailing wildcard syntax for prefix based version matching
//   and exclusion
//
// * Changed the top level sort position of the ``.devN`` suffix
//
// * Allowed single value version numbers
//
// * Explicit exclusion of leading or trailing whitespace
//
// * Explicit support for date based versions
//
// * Explicit normalisation rules to improve compatibility with
//   existing version metadata on PyPI where it doesn't introduce
//   ambiguity
//
// * Implicitly exclude pre-releases unless they're already present or
//   needed to satisfy a dependency
//
// * Treat post releases the same way as unqualified releases
//
// * Discuss ordering and dependencies across metadata versions
//
// * Switch from preferring ``c`` to ``rc``.
//
// The rationale for major changes is given in the following sections.
//
//
// Changing the version scheme
// ---------------------------
//
// One key change in the version scheme in this PEP relative to that in
// PEP 386 is to sort top level developmental releases like ``X.Y.devN`` ahead
// of alpha releases like ``X.Ya1``. This is a far more logical sort order, as
// projects already using both development releases and alphas/betas/release
// candidates do not want their developmental releases sorted in
// between their release candidates and their final releases. There is no
// rationale for using ``dev`` releases in that position rather than
// merely creating additional release candidates.
//
// The updated sort order also means the sorting of ``dev`` versions is now
// consistent between the metadata standard and the pre-existing behaviour
// of ``pkg_resources`` (and hence the behaviour of current installation
// tools).
//
// Making this change should make it easier for affected existing projects to
// migrate to the latest version of the metadata standard.
//
// Another change to the version scheme is to allow single number
// versions, similar to those used by non-Python projects like Mozilla
// Firefox, Google Chrome and the Fedora Linux distribution. This is actually
// expected to be more useful for version specifiers, but it is easier to
// allow it for both version specifiers and release numbers, rather than
// splitting the two definitions.
//
// The exclusion of leading and trailing whitespace was made explicit after
// a couple of projects with version identifiers differing only in a
// trailing ``\n`` character were found on PyPI.
//
// Various other normalisation rules were also added as described in the
// separate section on version normalisation below.
//
// `Appendix A` shows detailed results of an analysis of PyPI distribution
// version information, as collected on 8th August, 2014. This analysis
// compares the behavior of the explicitly ordered version scheme defined in
// this PEP with the de facto standard defined by the behavior of setuptools.
// These metrics are useful, as the intent of this PEP is to follow existing
// setuptools behavior as closely as is feasible, while still throwing
// exceptions for unorderable versions (rather than trying to guess an
// appropriate order as setuptools does).
//
//
// A more opinionated description of the versioning scheme
// -------------------------------------------------------
//
// As in PEP 386, the primary focus is on codifying existing practices to make
// them more amenable to automation, rather than demanding that existing
// projects make non-trivial changes to their workflow. However, the
// standard scheme allows significantly more flexibility than is needed
// for the vast majority of simple Python packages (which often don't even
// need maintenance releases - many users are happy with needing to upgrade to a
// new feature release to get bug fixes).
//
// For the benefit of novice developers, and for experienced developers
// wishing to better understand the various use cases, the specification
// now goes into much greater detail on the components of the defined
// version scheme, including examples of how each component may be used
// in practice.
//
// The PEP also explicitly guides developers in the direction of
// semantic versioning (without requiring it), and discourages the use of
// several aspects of the full versioning scheme that have largely been
// included in order to cover esoteric corner cases in the practices of
// existing projects and in repackaging software for Linux distributions.
//
//
// Describing version specifiers alongside the versioning scheme
// -------------------------------------------------------------
//
// The main reason to even have a standardised version scheme in the first place
// is to make it easier to do reliable automated dependency analysis. It makes
// more sense to describe the primary use case for version identifiers alongside
// their definition.
//
//
// Changing the interpretation of version specifiers
// -------------------------------------------------
//
// The previous interpretation of version specifiers made it very easy to
// accidentally download a pre-release version of a dependency. This in
// turn made it difficult for developers to publish pre-release versions
// of software to the Python Package Index, as even marking the package as
// hidden wasn't enough to keep automated tools from downloading it, and also
// made it harder for users to obtain the test release manually through the
// main PyPI web interface.
//
// The previous interpretation also excluded post-releases from some version
// specifiers for no adequately justified reason.
//
// The updated interpretation is intended to make it difficult to accidentally
// accept a pre-release version as satisfying a dependency, while still
// allowing pre-release versions to be retrieved automatically when that's the
// only way to satisfy a dependency.
//
// The "some forward compatibility assumed" version constraint is derived from the
// Ruby community's "pessimistic version constraint" operator [2]_ to allow
// projects to take a cautious approach to forward compatibility promises, while
// still easily setting a minimum required version for their dependencies. The
// spelling of the compatible release clause (``~=``) is inspired by the Ruby
// (``~>``) and PHP (``~``) equivalents.
//
// Further improvements are also planned to the handling of parallel
// installation of multiple versions of the same library, but these will
// depend on updates to the installation database definition along with
// improved tools for dynamic path manipulation.
//
// The trailing wildcard syntax to request prefix based version matching was
// added to make it possible to sensibly define compatible release clauses.
//
//
// Support for date based version identifiers
// ------------------------------------------
//
// Excluding date based versions caused significant problems in migrating
// ``pytz`` to the new metadata standards. It also caused concerns for the
// OpenStack developers, as they use a date based versioning scheme and would
// like to be able to migrate to the new metadata standards without changing
// it.
//
//
// Adding version epochs
// ---------------------
//
// Version epochs are added for the same reason they are part of other
// versioning schemes, such as those of the Fedora and Debian Linux
// distributions: to allow projects to gracefully change their approach to
// numbering releases, without having a new release appear to have a lower
// version number than previous releases and without having to change the name
// of the project.
//
// In particular, supporting version epochs allows a project that was previously
// using date based versioning to switch to semantic versioning by specifying
// a new version epoch.
//
// The ``!`` character was chosen to delimit an epoch version rather than the
// ``:`` character, which is commonly used in other systems, due to the fact that
// ``:`` is not a valid character in a Windows directory name.
//
//
// Adding direct references
// ------------------------
//
// Direct references are added as an "escape clause" to handle messy real
// world situations that don't map neatly to the standard distribution model.
// This includes dependencies on unpublished software for internal use, as well
// as handling the more complex compatibility issues that may arise when
// wrapping third party libraries as C extensions (this is of especial concern
// to the scientific community).
//
// Index servers are deliberately given a lot of freedom to disallow direct
// references, since they're intended primarily as a tool for integrators
// rather than publishers. PyPI in particular is currently going through the
// process of *eliminating* dependencies on external references, as unreliable
// external services have the effect of slowing down installation operations,
// as well as reducing PyPI's own apparent reliability.
//
//
// Adding arbitrary equality
// -------------------------
//
// Arbitrary equality is added as an "escape clause" to handle the case where
// someone needs to install a project which uses a non compliant version. Although
// this PEP is able to attain ~97% compatibility with the versions that are
// already on PyPI there are still ~3% of versions which cannot be parsed. This
// operator gives a simple and effective way to still depend on them without
// having to "guess" at the semantics of what they mean (which would be required
// if anything other than strict string based equality was supported).
//
//
// Adding local version identifiers
// --------------------------------
//
// It's a fact of life that downstream integrators often need to backport
// upstream bug fixes to older versions. It's one of the services that gets
// Linux distro vendors paid, and application developers may also apply patches
// they need to bundled dependencies.
//
// Historically, this practice has been invisible to cross-platform language
// specific distribution tools - the reported "version" in the upstream
// metadata is the same as for the unmodified code. This inaccuracy can then
// cause problems when attempting to work with a mixture of integrator
// provided code and unmodified upstream code, or even just attempting to
// identify exactly which version of the software is installed.
//
// The introduction of local version identifiers and "local version labels"
// into the versioning scheme, with the corresponding ``python.integrator``
// metadata extension allows this kind of activity to be represented
// accurately, which should improve interoperability between the upstream
// tools and various integrated platforms.
//
// The exact scheme chosen is largely modeled on the existing behavior of
// ``pkg_resources.parse_version`` and ``pkg_resources.parse_requirements``,
// with the main distinction being that where ``pkg_resources`` currently always
// takes the suffix into account when comparing versions for exact matches,
// the PEP requires that the local version label of the candidate version be
// ignored when no local version label is present in the version specifier
// clause. Furthermore, the PEP does not attempt to impose any structure on
// the local version labels (aside from limiting the set of permitted
// characters and defining their ordering).
//
// This change is designed to ensure that an integrator provided version like
// ``pip 1.5+1`` or ``pip 1.5+1.git.abc123de`` will still satisfy a version
// specifier like ``pip>=1.5``.
//
// The plus is chosen primarily for readability of local version identifiers.
// It was chosen instead of the hyphen to prevent
// ``pkg_resources.parse_version`` from parsing it as a prerelease, which is
// important for enabling a successful migration to the new, more structured,
// versioning scheme. The plus was chosen instead of a tilde because of the
// significance of the tilde in Debian's version ordering algorithm.
//
//
// Providing explicit version normalization rules
// ----------------------------------------------
//
// Historically, the de facto standard for parsing versions in Python has been the
// ``pkg_resources.parse_version`` command from the setuptools project. This does
// not attempt to reject *any* version and instead tries to make something
// meaningful, with varying levels of success, out of whatever it is given. It has
// a few simple rules but otherwise it more or less relies largely on string
// comparison.
//
// The normalization rules provided in this PEP exist primarily to either increase
// the compatibility with ``pkg_resources.parse_version``, particularly in
// documented use cases such as ``rev``, ``r``, ``pre``, etc or to do something
// more reasonable with versions that already exist on PyPI.
//
// All possible normalization rules were weighed against whether or not they were
// *likely* to cause any ambiguity (e.g. while someone might devise a scheme where
// ``v1.0`` and ``1.0`` are considered distinct releases, the likelihood of anyone
// actually doing that, much less on any scale that is noticeable, is fairly low).
// They were also weighed against how ``pkg_resources.parse_version`` treated a
// particular version string, especially with regards to how it was sorted. Finally
// each rule was weighed against the kinds of additional versions it allowed, how
// "ugly" those versions looked, how hard there were to parse (both mentally and
// mechanically) and how much additional compatibility it would bring.
//
// The breadth of possible normalizations were kept to things that could easily
// be implemented as part of the parsing of the version and not pre-parsing
// transformations applied to the versions. This was done to limit the side
// effects of each transformation as simple search and replace style transforms
// increase the likelihood of ambiguous or "junk" versions.
//
// For an extended discussion on the various types of normalizations that were
// considered, please see the proof of concept for PEP 440 within pip [5]_.
//
//
// Allowing Underscore in Normalization
// ------------------------------------
//
// There are not a lot of projects on PyPI which utilize a ``_`` in the version
// string. However this PEP allows its use anywhere that ``-`` is acceptable. The
// reason for this is that the Wheel normalization scheme specifies that ``-``
// gets normalized to a ``_`` to enable easier parsing of the filename.
//
//
// Summary of changes to \PEP 440
// ==============================
//
// The following changes were made to this PEP based on feedback received after
// the initial reference implementation was released in setuptools 8.0 and pip
// 6.0:
//
// * The exclusive ordered comparisons were updated to no longer imply a ``!=V.*``
//   which was deemed to be surprising behavior which was too hard to accurately
//   describe. Instead the exclusive ordered comparisons will simply disallow
//   matching pre-releases, post-releases, and local versions of the specified
//   version (unless the specified version is itself a pre-release, post-release
//   or local version). For an extended discussion see the threads on
//   distutils-sig [6]_ [7]_.
//
// * The normalized form for release candidates was updated from 'c' to 'rc'.
//   This change was based on user feedback received when setuptools 8.0
//   started applying normalisation to the release metadata generated when
//   preparing packages for publication on PyPI [8]_.
//
// * The PEP text and the ``is_canonical`` regex were updated to be explicit
//   that numeric components are specifically required to be represented as
//   sequences of ASCII digits, not arbitrary Unicode [Nd] code points. This
//   was previously implied by the version parsing regex in Appendix B, but
//   not stated explicitly [10]_.
//
//
//
// References
// ==========
//
// The initial attempt at a standardised version scheme, along with the
// justifications for needing such a standard can be found in PEP 386.
//
// .. [1] Reference Implementation of PEP 440 Versions and Specifiers
//    https://github.com/pypa/packaging/pull/1
//
// .. [2] Version compatibility analysis script:
//    https://github.com/pypa/packaging/blob/master/tasks/check.py
//
// .. [3] Pessimistic version constraint
//    http://docs.rubygems.org/read/chapter/16
//
// .. [4] File URIs in Windows
//    http://blogs.msdn.com/b/ie/archive/2006/12/06/file-uris-in-windows.aspx
//
// .. [5] Proof of Concept: PEP 440 within pip
//     https://github.com/pypa/pip/pull/1894
//
// .. [6] PEP440: foo-X.Y.Z does not satisfy "foo>X.Y"
//     https://mail.python.org/pipermail/distutils-sig/2014-December/025451.html
//
// .. [7] PEP440: >1.7 vs >=1.7
//     https://mail.python.org/pipermail/distutils-sig/2014-December/025507.html
//
// .. [8] Amend PEP 440 with Wider Feedback on Release Candidates
//    https://mail.python.org/pipermail/distutils-sig/2014-December/025409.html
//
// .. [9] Changing the status of PEP 440 to Provisional
//    https://mail.python.org/pipermail/distutils-sig/2014-December/025412.html
//
// .. [10] PEP 440: regex should not permit Unicode [Nd] characters
//    https://github.com/python/peps/pull/966
//
// Appendix A
// ==========
//
// Metadata v2.0 guidelines versus setuptools::
//
//     $ invoke check.pep440
//     Total Version Compatibility:              245806/250521 (98.12%)
//     Total Sorting Compatibility (Unfiltered): 45441/47114 (96.45%)
//     Total Sorting Compatibility (Filtered):   47057/47114 (99.88%)
//     Projects with No Compatible Versions:     498/47114 (1.06%)
//     Projects with Differing Latest Version:   688/47114 (1.46%)
//

// Appendix B : Parsing version strings with regular expressions
// =============================================================
//
// As noted earlier in the `Public version identifiers` section, published
// version identifiers SHOULD use the canonical format. This section provides
// regular expressions that can be used to test whether a version is already
// in that form, and if it's not, extract the various components for subsequent
// normalization.
//
// To test whether a version identifier is in the canonical format, you can use
// the following function::
//

//nolint:lll // long regexp in source specification
//
//     import re
//     def is_canonical(version):
//         return re.match(r'^([1-9][0-9]*!)?(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*))*((a|b|rc)(0|[1-9][0-9]*))?(\.post(0|[1-9][0-9]*))?(\.dev(0|[1-9][0-9]*))?$', version) is not None

//
// To extract the components of a version identifier, use the following regular
// expression (as defined by the `packaging <https://github.com/pypa/packaging>`_
// project)::
//
//     VERSION_PATTERN = r"""
//         v?
//         (?:
//             (?:(?P<epoch>[0-9]+)!)?                           # epoch
//             (?P<release>[0-9]+(?:\.[0-9]+)*)                  # release segment
//             (?P<pre>                                          # pre-release
//                 [-_\.]?
//                 (?P<pre_l>(a|b|c|rc|alpha|beta|pre|preview))
//                 [-_\.]?
//                 (?P<pre_n>[0-9]+)?
//             )?
//             (?P<post>                                         # post release
//                 (?:-(?P<post_n1>[0-9]+))
//                 |
//                 (?:
//                     [-_\.]?
//                     (?P<post_l>post|rev|r)
//                     [-_\.]?
//                     (?P<post_n2>[0-9]+)?
//                 )
//             )?
//             (?P<dev>                                          # dev release
//                 [-_\.]?
//                 (?P<dev_l>dev)
//                 [-_\.]?
//                 (?P<dev_n>[0-9]+)?
//             )?
//         )
//         (?:\+(?P<local>[a-z0-9]+(?:[-_\.][a-z0-9]+)*))?       # local version
//     """
//
//     _regex = re.compile(
//         r"^\s*" + VERSION_PATTERN + r"\s*$",
//         re.VERBOSE | re.IGNORECASE,
//     )
var reVersion = regexp.MustCompile(`(?i)^\s*` + regexp.MustCompile(`(?:\s+|#.*)`).ReplaceAllString(`
		v?
		(?:
		    (?:(?P<epoch>[0-9]+)!)?                           # epoch
		    (?P<release>[0-9]+(?:\.[0-9]+)*)                  # release segment
		    (?P<pre>                                          # pre-release
		        [-_\.]?
		        (?P<pre_l>(a|b|c|rc|alpha|beta|pre|preview))
		        [-_\.]?
		        (?P<pre_n>[0-9]+)?
		    )?
		    (?P<post>                                         # post release
		        (?:-(?P<post_n1>[0-9]+))
		        |
		        (?:
		            [-_\.]?
		            (?P<post_l>post|rev|r)
		            [-_\.]?
		            (?P<post_n2>[0-9]+)?
		        )
		    )?
		    (?P<dev>                                          # dev release
		        [-_\.]?
		        (?P<dev_l>dev)
		        [-_\.]?
		        (?P<dev_n>[0-9]+)?
		    )?
		)
		(?:\+(?P<local>[a-z0-9]+(?:[-_\.][a-z0-9]+)*))?       # local version
	`, ``) + `\s*$`)

func parseVersion(str string) (*Version, error) {
	match := reVersion.FindStringSubmatch(str)
	if match == nil {
		return nil, fmt.Errorf("invalid version: %q", str)
	}

	var ver Version
	var err error

	if epoch := match[reVersion.SubexpIndex("epoch")]; epoch != "" {
		ver.Epoch, err = strconv.Atoi(epoch)
		if err != nil {
			return nil, err
		}
	}

	for _, segStr := range strings.Split(match[reVersion.SubexpIndex("release")], ".") {
		segInt, err := strconv.Atoi(segStr)
		if err != nil {
			return nil, err
		}
		ver.Release = append(ver.Release, segInt)
	}

	type letterNumber struct {
		L string
		N int
	}

	parseLetterNumber := func(letter, number string, acceptableLetters map[string][]string) (*letterNumber, error) {
		if letter == "" && number == "" {
			//nolint:nilnil // weird semantic
			return nil, nil
		}
		letter = strings.ToLower(letter)
		if letter != "" && number == "" {
			number = "0"
		}
		var ret letterNumber

		if _, ok := acceptableLetters[letter]; ok {
			ret.L = letter
		} else {
			found := false
		outer:
			for canonical, others := range acceptableLetters {
				for _, other := range others {
					if letter == other {
						ret.L = canonical
						found = true
						break outer
					}
				}
			}
			if !found {
				return nil, fmt.Errorf("invalid string-part: %q", letter)
			}
		}

		if number != "" {
			ret.N, err = strconv.Atoi(number)
			if err != nil {
				return nil, err
			}
		}
		return &ret, nil
	}

	pre, err := parseLetterNumber(
		match[reVersion.SubexpIndex("pre_l")],
		match[reVersion.SubexpIndex("pre_n")],
		map[string][]string{
			"a":  {"alpha"},
			"b":  {"beta"},
			"rc": {"c", "pre", "preview"},
		})
	if err != nil {
		return nil, fmt.Errorf("pre-release: %w", err)
	}
	if pre != nil {
		ver.Pre = &PreRelease{
			L: pre.L,
			N: pre.N,
		}
	}

	post, err := parseLetterNumber(
		match[reVersion.SubexpIndex("post_l")],
		match[reVersion.SubexpIndex("post_n1")]+match[reVersion.SubexpIndex("post_n2")],
		map[string][]string{
			"post": {"", "rev", "r"},
		})
	if err != nil {
		return nil, fmt.Errorf("post-release: %w", err)
	}
	if post != nil {
		ver.Post = &post.N
	}

	dev, err := parseLetterNumber(
		match[reVersion.SubexpIndex("dev_l")],
		match[reVersion.SubexpIndex("dev_n")],
		map[string][]string{
			"dev": nil,
		})
	if err != nil {
		return nil, fmt.Errorf("dev: %w", err)
	}
	if dev != nil {
		ver.Dev = &dev.N
	}

	localParts := strings.FieldsFunc(match[reVersion.SubexpIndex("local")], func(r rune) bool {
		return strings.ContainsRune("-_.", r)
	})
	for _, part := range localParts {
		ver.Local = append(ver.Local, intstr.Parse(strings.ToLower(part)))
	}

	return &ver, nil
}

//
//
// Copyright
// =========
//
// This document has been placed in the public domain.
