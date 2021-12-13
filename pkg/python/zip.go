// This file mimics `zipfile.py`.

package python

// A DOSAttribute represents an "MS-DOS directory attribute byte" (as referenced by the ZIP file
// format specification[1]).
//
// [1]: https://www.pkware.com/appnote
type DOSAttribute uint8

// https://docs.microsoft.com/en-us/windows/win32/fileio/file-attribute-constants
const (
	// The letter indicates the flag letter used in MS-DOS commands.
	DOSReadOnly  DOSAttribute = 1 << iota // R
	DOSHidden                             // H
	DOSSystem                             // S
	_DOSUnused3                           // 1<<3
	DOSDirectory                          // D
	DOSArchive                            // A
	_DOSUnused6                           // 1<<6
	_DOSUnused7                           // 1<<7
)

// A ZIPExternalAttributes represents Python's view of a ZIP file's "external file attributes"
// field.
//
// The ZIP file format specification[1] specifies a 4-byte "external file attributes" field for each
// file, the meaning of which depends on the platform on which the ZIP file was created (or at least
// claims to be created in the "version made by" field).
//
// On the "MS-DOS" (0x00) platform, it is specified that only the low byte is used.
//
// On the "UNIX" (0x03) platform, the meaning isn't specified in the main ZIP spec, I'm not sure
// where it's specified, but only the upper 2 bytes are used.
//
// Now, Python's `zipfile` doesn't actually check the "version made by" field, and just parses it
// both ways (and assumes that all the world is either UNIX or Microsoft DOS/Windows-32).
//
// [1]: https://www.pkware.com/appnote
type ZIPExternalAttributes struct {
	UNIX   StatMode
	Unused uint8
	MSDOS  DOSAttribute
}

// Raw turns a 32-bit ZIPExternalAttributes struct in to an unstructured 32-bit unsigned integer.
func (ea ZIPExternalAttributes) Raw() uint32 {
	return uint32(ea.UNIX)<<16 | uint32(ea.Unused)<<8 | uint32(ea.MSDOS)
}

// ParseZIPExternalAttributes turns an unstructured 32-bit unsigned integer in to a 32-bit
// ZIPExternalAttributes struct.
func ParseZIPExternalAttributes(raw uint32) ZIPExternalAttributes {
	return ZIPExternalAttributes{
		UNIX:   StatMode(raw >> 16),
		Unused: uint8(raw >> 8),
		MSDOS:  DOSAttribute(raw),
	}
}
