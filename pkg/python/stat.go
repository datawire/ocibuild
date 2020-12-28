// This file mimics `stat.py`.

package python

import (
	"os"
)

// A StatMode represents a file's mode and permission bits, as represented in Python
// (i.e. `os.stat()`'s `st_mode` member).  Similar to how Go's `os.FileMode` assigns bits to have
// the same definition on all systems for portability, Python's `stat` assigns bits to have the same
// definition on all systems for portability.  And it just so happens that Go's bits match those of
// Plan 9, and Python's bits match those of the Linux kernel.
type StatMode uint16

const (
	// 16 bits = 5⅓ octal characters

	ModeFmt StatMode = 017_0000 // mask for the type bits

	ModeFmtNamedPipe   StatMode = 001_0000 // type: named pipe (FIFO)
	ModeFmtCharDevice  StatMode = 002_0000 // type: character device
	ModeFmtDir         StatMode = 004_0000 // type: directory
	ModeFmtBlockDevice StatMode = 006_0000 // type: block device
	ModeFmtRegular     StatMode = 010_0000 // type: regular file
	ModeFmtSymlink     StatMode = 012_0000 // type: symbolic link
	ModeFmtSocket      StatMode = 014_0000 // type: socket file

	ModePerm StatMode = 000_7777 // mask for permission bits

	ModePermSetUID StatMode = 000_4000 // permission: set user id
	ModePermSetGID StatMode = 000_2000 // permission: set group ID
	ModePermSticky StatMode = 000_1000 // permission: sticky bit

	ModePermUsrR StatMode = 000_0400 // permission: user: read
	ModePermUsrW StatMode = 000_0200 // permission: user: write
	ModePermUsrX StatMode = 000_0100 // permission: user: execute

	ModePermGrpR StatMode = 000_0040 // permission: group: read
	ModePermGrpW StatMode = 000_0020 // permission: group: write
	ModePermGrpX StatMode = 000_0010 // permission: group: execute

	ModePermOthR StatMode = 000_0004 // permission: other: read
	ModePermOthW StatMode = 000_0002 // permission: other: write
	ModePermOthX StatMode = 000_0001 // permission: other: execute
)

func (pm StatMode) ToGo() os.FileMode {
	// permissions: base
	gm := os.FileMode(pm & 0777)

	// permissions: extended
	if pm&ModePermSetGID != 0 {
		gm |= os.ModeSetgid
	}
	if pm&ModePermSetUID != 0 {
		gm |= os.ModeSetuid
	}
	if pm&ModePermSticky != 0 {
		gm |= os.ModeSticky
	}

	// type
	switch pm & ModeFmt {
	case ModeFmtBlockDevice:
		gm |= os.ModeDevice
	case ModeFmtCharDevice:
		gm |= os.ModeDevice | os.ModeCharDevice
	case ModeFmtDir:
		gm |= os.ModeDir
	case ModeFmtNamedPipe:
		gm |= os.ModeNamedPipe
	case ModeFmtSymlink:
		gm |= os.ModeSymlink
	case ModeFmtRegular:
		// nothing to do
	case ModeFmtSocket:
		gm |= os.ModeSocket
	}

	return gm
}

func ModeFromGo(gm os.FileMode) StatMode {
	// permissions: base
	pm := StatMode(gm & 0777)

	// permissions: extended
	if gm&os.ModeSetgid != 0 {
		pm |= ModePermSetGID
	}
	if gm&os.ModeSetuid != 0 {
		pm |= ModePermSetUID
	}
	if gm&os.ModeSticky != 0 {
		pm |= ModePermSticky
	}

	// type
	switch gm & os.ModeType {
	case os.ModeDevice:
		pm |= ModeFmtBlockDevice
	case os.ModeDevice | os.ModeCharDevice:
		pm |= ModeFmtCharDevice
	case os.ModeDir:
		pm |= ModeFmtDir
	case os.ModeNamedPipe:
		pm |= ModeFmtNamedPipe
	case os.ModeSymlink:
		pm |= ModeFmtSymlink
	case 0:
		pm |= ModeFmtRegular
	case os.ModeSocket:
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
// command.  POSIX does not specify the character to use to indicate a ModeFmtFocket file; this
// method uses 's' (both Python `stat.filemode()` behavior and GNU `ls` behavior; though POSIX notes
// that many implementations use '=' for sockets).
func (pm StatMode) String() string {
	buf := [10]byte{
		// type
		"?pc?d?b?-?l?s???"[pm>>12],

		// owner
		"-r"[(pm>>8)&01],
		"-w"[(pm>>7)&01],
		"-xSs"[((pm>>6)&01)|((pm>>10)&02)],

		// group
		"-r"[(pm>>5)&01],
		"-w"[(pm>>4)&01],
		"-xSs"[((pm>>3)&01)|((pm>>9)&02)],

		// group
		"-r"[(pm>>2)&01],
		"-w"[(pm>>1)&01],
		"-xTt"[((pm>>0)&01)|((pm>>8)&02)],
	}

	return string(buf[:])
}

// A StatFileAttribute represents MS Windows file attributes, as seen in Python `os.stat()`'s
// `st_file_attributes` member.
type StatFileAttribute uint32

// https://docs.microsoft.com/en-us/windows/win32/fileio/file-attribute-constants
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

func (fa StatFileAttribute) ToDOS() DOSAttribute {
	return DOSAttribute(fa & 0b00110111)
}
