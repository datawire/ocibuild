package pep427

import (
	"io"
)

type skipReader struct {
	skip  int
	inner io.Reader
}

func (r *skipReader) Read(p []byte) (int, error) {
	if r.skip > 0 {
		buff := make([]byte, r.skip)
		n, err := io.ReadFull(r.inner, buff)
		r.skip -= n
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return 0, err
		}
	}
	return r.inner.Read(p)
}

type readCloser struct {
	io.Reader
	io.Closer
}
