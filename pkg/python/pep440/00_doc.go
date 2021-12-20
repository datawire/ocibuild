// Package pep440 implements PEP 440 -- Version Identification and Dependency Specification.
//
// https://www.python.org/dev/peps/pep-0440/
package pep440

// This package contains as comments the full text of
// https://github.com/python/peps/blob/master/pep-0440.txt which has been placed in to the public
// domain.

// PEP: 440
// Title: Version Identification and Dependency Specification
// Version: $Revision$
// Last-Modified: $Date$
// Author: Nick Coghlan <ncoghlan@gmail.com>,
//         Donald Stufft <donald@stufft.io>
// BDFL-Delegate: Nick Coghlan <ncoghlan@gmail.com>
// Discussions-To: Distutils SIG <distutils-sig@python.org>
// Status: Active
// Type: Informational
// Content-Type: text/x-rst
// Created: 18-Mar-2013
// Post-History: 30 Mar 2013, 27 May 2013, 20 Jun 2013,
//               21 Dec 2013, 28 Jan 2014, 08 Aug 2014
//               22 Aug 2014
// Replaces: 386
// Resolution: https://mail.python.org/pipermail/distutils-sig/2014-August/024673.html
//
//
// Abstract
// ========
//
// This PEP describes a scheme for identifying versions of Python software
// distributions, and declaring dependencies on particular versions.
//
// This document addresses several limitations of the previous attempt at a
// standardized approach to versioning, as described in PEP 345 and PEP 386.
//
//
// Definitions
// ===========
//
// The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT",
// "SHOULD", "SHOULD NOT", "RECOMMENDED",  "MAY", and "OPTIONAL" in this
// document are to be interpreted as described in RFC 2119.
//
// "Projects" are software components that are made available for integration.
// Projects include Python libraries, frameworks, scripts, plugins,
// applications, collections of data or other resources, and various
// combinations thereof. Public Python projects are typically registered on
// the `Python Package Index <https://pypi.python.org>`__.
//
// "Releases" are uniquely identified snapshots of a project.
//
// "Distributions" are the packaged files which are used to publish
// and distribute a release.
//
// "Build tools" are automated tools intended to run on development systems,
// producing source and binary distribution archives. Build tools may also be
// invoked by integration tools in order to build software distributed as
// sdists rather than prebuilt binary archives.
//
// "Index servers" are active distribution registries which publish version and
// dependency metadata and place constraints on the permitted metadata.
//
// "Publication tools" are automated tools intended to run on development
// systems and upload source and binary distribution archives to index servers.
//
// "Installation tools" are integration tools specifically intended to run on
// deployment targets, consuming source and binary distribution archives from
// an index server or other designated location and deploying them to the target
// system.
//
// "Automated tools" is a collective term covering build tools, index servers,
// publication tools, integration tools and any other software that produces
// or consumes distribution version and dependency metadata.
// This file contains as comments the full text of
// https://github.com/python/peps/blob/master/pep-0440.txt
// which has been placed in to the public domain.

// PEP: 440
// Title: Version Identification and Dependency Specification
// Version: $Revision$
// Last-Modified: $Date$
// Author: Nick Coghlan <ncoghlan@gmail.com>,
//         Donald Stufft <donald@stufft.io>
// BDFL-Delegate: Nick Coghlan <ncoghlan@gmail.com>
// Discussions-To: Distutils SIG <distutils-sig@python.org>
// Status: Active
// Type: Informational
// Content-Type: text/x-rst
// Created: 18-Mar-2013
// Post-History: 30 Mar 2013, 27 May 2013, 20 Jun 2013,
//               21 Dec 2013, 28 Jan 2014, 08 Aug 2014
//               22 Aug 2014
// Replaces: 386
// Resolution: https://mail.python.org/pipermail/distutils-sig/2014-August/024673.html
//
//
// Abstract
// ========
//
// This PEP describes a scheme for identifying versions of Python software
// distributions, and declaring dependencies on particular versions.
//
// This document addresses several limitations of the previous attempt at a
// standardized approach to versioning, as described in PEP 345 and PEP 386.
//
//
// Definitions
// ===========
//
// The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT",
// "SHOULD", "SHOULD NOT", "RECOMMENDED",  "MAY", and "OPTIONAL" in this
// document are to be interpreted as described in RFC 2119.
//
// "Projects" are software components that are made available for integration.
// Projects include Python libraries, frameworks, scripts, plugins,
// applications, collections of data or other resources, and various
// combinations thereof. Public Python projects are typically registered on
// the `Python Package Index <https://pypi.python.org>`__.
//
// "Releases" are uniquely identified snapshots of a project.
//
// "Distributions" are the packaged files which are used to publish
// and distribute a release.
//
// "Build tools" are automated tools intended to run on development systems,
// producing source and binary distribution archives. Build tools may also be
// invoked by integration tools in order to build software distributed as
// sdists rather than prebuilt binary archives.
//
// "Index servers" are active distribution registries which publish version and
// dependency metadata and place constraints on the permitted metadata.
//
// "Publication tools" are automated tools intended to run on development
// systems and upload source and binary distribution archives to index servers.
//
// "Installation tools" are integration tools specifically intended to run on
// deployment targets, consuming source and binary distribution archives from
// an index server or other designated location and deploying them to the target
// system.
//
// "Automated tools" is a collective term covering build tools, index servers,
// publication tools, integration tools and any other software that produces
// or consumes distribution version and dependency metadata.
