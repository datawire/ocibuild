package direct_url

import (
	"bytes"
	"encoding/json"
	"io"
)

// jsonDumps is like `json.Marshal`, but mimics the formatting of Python stdlib `json.dumps`.
func jsonDumps(typedObj interface{}) ([]byte, error) {
	// First, round-trip through JSON first to discard any type information.
	src, err := json.Marshal(typedObj)
	if err != nil {
		return nil, err
	}
	var untypedObj interface{}
	if err := json.Unmarshal(src, &untypedObj); err != nil {
		return nil, err
	}
	// Second, marshal it "for real".
	src, err = json.Marshal(untypedObj)
	if err != nil {
		return nil, err
	}
	// Third, post-process that marshaling to match json.dumps whitespace.
	var dst bytes.Buffer
	decoder := json.NewDecoder(bytes.NewReader(src))
	mapStack := []int{-1}
	completeObj := func() {
		depth := len(mapStack) - 1
		if mapStack[depth] < 0 {
			// we're inside an array
			mapStack[depth] = mapStack[depth] - 1
		} else {
			// we're inside a map
			if mapStack[depth]%2 == 1 {
				_, _ = dst.WriteString(": ")
			}
			mapStack[depth] = mapStack[depth] + 1
		}
	}
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return dst.Bytes(), nil
			}
			return nil, err
		}

		switch tok := tok.(type) {
		case json.Delim:
			switch tok {
			case '[':
				mapStack = append(mapStack, -1)
			case '{':
				mapStack = append(mapStack, 1)
			case '}', ']':
				mapStack = mapStack[:len(mapStack)-1]
				completeObj()
			}
			dst.WriteRune(rune(tok))
		default:
			if depth := len(mapStack) - 1; mapStack[depth] < -1 {
				// we're inside an array
				_, _ = dst.WriteString(", ")
			} else if mapStack[depth] > 1 && mapStack[depth]%2 == 1 {
				// we're inside an map
				_, _ = dst.WriteString(", ")
			}

			bs, err := json.Marshal(tok)
			if err != nil {
				return nil, err
			}

			_, _ = dst.Write(bs)

			completeObj()
		}
	}
}
