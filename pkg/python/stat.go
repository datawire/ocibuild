// This file mimics `stat.py`.

package python

import (
	"io/fs"
)

// A StatMode represents a file's mode and permission bits, as represented in Python
// (i.e. `os.stat()`'s `st_mode` member).  Similar to how Go's `io/fs.FileMode` assigns bits to have
// the same definition on all systems for portability, Python's `stat` assigns bits to have the same
// definition on all systems for portability.  And it just so happens that Go's bits match those of
// Plan 9, and Python's bits match those of the Linux kernel.
type StatMode uint16

//nolint:deadcode,varcheck // not all of these modes will be used
const (
	// 16 bits = 5⅓ octal characters

	ModeFmt StatMode = 0o17_0000 // mask for the type bits

	_ModeFmtUnused000  StatMode = 0o00_0000
	ModeFmtNamedPipe   StatMode = 0o01_0000 // type: named pipe (FIFO)
	ModeFmtCharDevice  StatMode = 0o02_0000 // type: character device
	_ModeFmtUnused003  StatMode = 0o03_0000
	ModeFmtDir         StatMode = 0o04_0000 // type: directory
	_ModeFmtUnused005  StatMode = 0o05_0000
	ModeFmtBlockDevice StatMode = 0o06_0000 // type: block device
	_ModeFmtUnused007  StatMode = 0o07_0000
	ModeFmtRegular     StatMode = 0o10_0000 // type: regular file
	_ModeFmtUnused011  StatMode = 0o11_0000
	ModeFmtSymlink     StatMode = 0o12_0000 // type: symbolic link
	_ModeFmtUnused013  StatMode = 0o13_0000
	ModeFmtSocket      StatMode = 0o14_0000 // type: socket file
	_ModeFmtUnused015  StatMode = 0o15_0000
	ModeFmtWhiteout    StatMode = 0o16_0000 // type: whiteout (non-Linux)
	_ModeFmtUnused017  StatMode = 0o17_0000

	ModePerm StatMode = 0o00_7777 // mask for permission bits

	ModePermSetUID StatMode = 0o00_4000 // permission: set user id
	ModePermSetGID StatMode = 0o00_2000 // permission: set group ID
	ModePermSticky StatMode = 0o00_1000 // permission: sticky bit

	ModePermUsrR StatMode = 0o00_0400 // permission: user: read
	ModePermUsrW StatMode = 0o00_0200 // permission: user: write
	ModePermUsrX StatMode = 0o00_0100 // permission: user: execute

	ModePermGrpR StatMode = 0o00_0040 // permission: group: read
	ModePermGrpW StatMode = 0o00_0020 // permission: group: write
	ModePermGrpX StatMode = 0o00_0010 // permission: group: execute

	ModePermOthR StatMode = 0o00_0004 // permission: other: read
	ModePermOthW StatMode = 0o00_0002 // permission: other: write
	ModePermOthX StatMode = 0o00_0001 // permission: other: execute
)

// ToGo translates pm from a Statmode to an fs.FileMode.
func (pm StatMode) ToGo() fs.FileMode {
	// permissions: base
	gm := fs.FileMode(pm & 0o777)

	// permissions: extended
	if pm&ModePermSetGID != 0 {
		gm |= fs.ModeSetgid
	}
	if pm&ModePermSetUID != 0 {
		gm |= fs.ModeSetuid
	}
	if pm&ModePermSticky != 0 {
		gm |= fs.ModeSticky
	}

	// type
	switch pm & ModeFmt {
	case ModeFmtBlockDevice:
		gm |= fs.ModeDevice
	case ModeFmtCharDevice:
		gm |= fs.ModeDevice | fs.ModeCharDevice
	case ModeFmtDir:
		gm |= fs.ModeDir
	case ModeFmtNamedPipe:
		gm |= fs.ModeNamedPipe
	case ModeFmtSymlink:
		gm |= fs.ModeSymlink
	case ModeFmtRegular:
		// nothing to do
	case ModeFmtSocket:
		gm |= fs.ModeSocket
	}

	return gm
}

// ModeFromGo translates an fs.FileMode to a StatMode.
func ModeFromGo(gm fs.FileMode) StatMode {
	// permissions: base
	pm := StatMode(gm & 0o777)

	// permissions: extended
	if gm&fs.ModeSetgid != 0 {
		pm |= ModePermSetGID
	}
	if gm&fs.ModeSetuid != 0 {
		pm |= ModePermSetUID
	}
	if gm&fs.ModeSticky != 0 {
		pm |= ModePermSticky
	}

	// type
	switch gm & fs.ModeType {
	case fs.ModeDevice:
		pm |= ModeFmtBlockDevice
	case fs.ModeDevice | fs.ModeCharDevice:
		pm |= ModeFmtCharDevice
	case fs.ModeDir:
		pm |= ModeFmtDir
	case fs.ModeNamedPipe:
		pm |= ModeFmtNamedPipe
	case fs.ModeSymlink:
		pm |= ModeFmtSymlink
	case 0:
		pm |= ModeFmtRegular
	case fs.ModeSocket:
		pm |= ModeFmtSocket
	}

	return pm
}

// IsDir reports whether pm describes a directory.
//
// That is, it tests that the ModeFmt bits are set to ModeFmtDir.
func (pm StatMode) IsDir() bool {
	return pm&ModeFmt == ModeFmtDir
}

// IsRegular reports whether m describes a regular file.
//
// That is, it tests that the ModeFmt bits are set to ModeFmtRegular.
func (pm StatMode) IsRegular() bool {
	return pm&ModeFmt == ModeFmtRegular
}

// String returns a textual representation of the mode.
//
// This is the format that POSIX specifies for showing the mode in the output of the `ls -l`
// command.  POSIX does not specify the character to use to indicate a ModeFmtSocket file; this
// method uses 's' (both Python `stat.filemode()` behavior and GNU `ls` behavior; though POSIX notes
// that many implementations use '=' for sockets).  POSIX does not specify the character to use to
// indicate a ModeFmtWhiteout file; the method uses 'w' (the behavior of Pyton `stat.filemode()`).
func (pm StatMode) String() string {
	buf := [10]byte{
		// type: This string is easy; it directly pairs with the above ModeFmtXXX list
		// above; the character in the string left-to-right corresponds with the constant in
		// the list top-to-bottom.
		"?pc?d?b?-?l?s?w?"[pm>>12],

		// owner
		"-r"[(pm>>8)&0o1],
		"-w"[(pm>>7)&0o1],
		"-xSs"[((pm>>6)&0o1)|((pm>>10)&0o2)],

		// group
		"-r"[(pm>>5)&0o1],
		"-w"[(pm>>4)&0o1],
		"-xSs"[((pm>>3)&0o1)|((pm>>9)&0o2)],

		// group
		"-r"[(pm>>2)&0o1],
		"-w"[(pm>>1)&0o1],
		"-xTt"[((pm>>0)&0o1)|((pm>>8)&0o2)],
	}

	return string(buf[:])
}

// A StatFileAttribute represents MS Windows file attributes, as seen in Python `os.stat()`'s
// `st_file_attributes` member.
type StatFileAttribute uint32

// https://docs.microsoft.com/en-us/windows/win32/fileio/file-attribute-constants
//
//nolint:deadcode,varcheck // not all of these modes will be used
const (
	// 1<<0 through 1<<7
	FileAttributeReadonly  StatFileAttribute = StatFileAttribute(DOSReadOnly)
	FileAttributeHidden    StatFileAttribute = StatFileAttribute(DOSHidden)
	FileAttributeSystem    StatFileAttribute = StatFileAttribute(DOSSystem)
	_FileAttributeUnused3  StatFileAttribute = StatFileAttribute(_DOSUnused3)
	FileAttributeDirectory StatFileAttribute = StatFileAttribute(DOSDirectory)
	FileAttributeArchive   StatFileAttribute = StatFileAttribute(DOSArchive)
	FileAttributeDevice    StatFileAttribute = StatFileAttribute(_DOSUnused6)
	FileAttributeNormal    StatFileAttribute = StatFileAttribute(_DOSUnused7)

	// 1<<8 through 1<<15
	FileAttributeTemporary StatFileAttribute = 1 << (8 + iota)
	FileAttributeSparseFile
	FileAttributeReparsePoint
	FileAttributeCompressed
	FileAttributeOffline
	FileAttributeNotContentIndexed
	FileAttributeEncrypted
	FileAttributeIntegrityStream

	// 1<<16 through 1<<23
	FileAttributeVirtual
	FileAttributeNoScrubData
	FileAttributeRecallOnOpen // not in Python 3.9.1 stat.py
	_FileattributeUnused19
	FileAttributeRecallOnDataAccess // not in Python 3.9.1 stat.py
)

// ToDOS returns just the bits that are valid DOSAttribute bits.
func (fa StatFileAttribute) ToDOS() DOSAttribute {
	return DOSAttribute(fa & 0b00110111)
}
