package fsutil

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"reflect"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
)

func readersEqual(a, b io.Reader) (equal bool, err error) {
	const chunkSize = 1024

	var aBuf, bBuf [chunkSize]byte
	for {
		aLen, err := io.ReadFull(a, aBuf[:])
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return false, err
		}
		bLen, err := io.ReadFull(b, bBuf[:])
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return false, err
		}
		if !bytes.Equal(aBuf[:aLen], bBuf[:bLen]) {
			return false, nil
		}
		if aLen < chunkSize {
			// EOF
			break
		}
	}

	return true, nil
}

func headersEqualExceptTimestamps(a, b tar.Header) bool {
	b.ModTime = a.ModTime
	b.AccessTime = a.AccessTime
	b.ModTime = a.ModTime

	return reflect.DeepEqual(a, b)
}

func LayersEqualExceptTimestamps(aLayer, bLayer ociv1.Layer) (equal bool, err error) {
	maybeSetErr := func(_err error) {
		if _err != nil && err == nil {
			equal = false
			err = _err
		}
	}

	aIOReader, err := aLayer.Uncompressed()
	if err != nil {
		return false, err
	}
	defer func() {
		maybeSetErr(aIOReader.Close())
	}()
	aTarReader := tar.NewReader(aIOReader)

	bIOReader, err := bLayer.Uncompressed()
	if err != nil {
		return false, err
	}
	defer func() {
		maybeSetErr(bIOReader.Close())
	}()
	bTarReader := tar.NewReader(bIOReader)

	for {
		aHeader, err := aTarReader.Next()
		if err != nil && !errors.Is(err, io.EOF) {
			return false, err
		}
		bHeader, err := bTarReader.Next()
		if err != nil && !errors.Is(err, io.EOF) {
			return false, err
		}
		if aHeader == nil && bHeader == nil {
			// got EOF from both
			break
		}
		if aHeader == nil || bHeader == nil {
			// one got EOF before the other
			return false, nil
		}

		if !headersEqualExceptTimestamps(*aHeader, *bHeader) {
			return false, nil
		}

		if equal, err := readersEqual(aTarReader, bTarReader); err != nil {
			return false, err
		} else if !equal {
			return false, nil
		}
	}

	return true, nil
}
