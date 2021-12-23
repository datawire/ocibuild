package bdist

import (
	"crypto/sha256"
	"encoding/base64"
	"io"

	"github.com/datawire/ocibuild/pkg/fsutil"
)

type Recordable interface {
	fsutil.FileReference
	Record() (hash string, size int64)
}

type withRecord struct {
	fsutil.FileReference
	RecordHash string
	RecordSize int64
}

func (f *withRecord) Record() (string, int64) {
	return f.RecordHash, f.RecordSize
}

var _ Recordable = (*withRecord)(nil)

func genRecord(open func() (io.ReadCloser, error)) (string, int64, error) {
	reader, err := open()
	if err != nil {
		return "", 0, err
	}
	defer func() {
		_ = reader.Close()
	}()

	hasher := sha256.New()
	size, err := io.Copy(hasher, reader)
	if err != nil {
		return "", 0, err
	}
	hash := "sha256=" + base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
	return hash, size, nil
}
