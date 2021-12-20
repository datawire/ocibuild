// This file mimics `configparser.py`.

package python

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type Config map[string]ConfigSection

type ConfigSection map[string]string

type ConfigParser struct {
	Delimiters            []string
	CommentPrefixes       []string
	InlineCommentPrefixes []string

	Strict             bool
	EmptyLinesInValues bool

	// Transform keys
	OptionTransform func(string) string
	// Transform values
	Interpolate func(Config, string) (string, error)
}

func NewConfigParser() *ConfigParser {
	return &ConfigParser{
		Delimiters:            []string{"=", ":"},
		CommentPrefixes:       []string{"#", ";"},
		InlineCommentPrefixes: []string{},

		Strict:             true,
		EmptyLinesInValues: true,

		OptionTransform: strings.ToLower,
		Interpolate:     NoInterpolation, // TODO(lukeshu): Implement BasicInterpolation.
	}
}

func (p *ConfigParser) Parse(fp io.Reader) (Config, error) {
	config := make(Config)

	var (
		curIndentLevel int
		curSection     ConfigSection
		curKey         string
		curVal         []string
	)

	flushKV := func() {
		if curVal != nil {
			curSection[curKey] = strings.TrimRight(strings.Join(curVal, "\n"), "\n")
			curKey = ""
			curVal = nil
		}
	}

	fpLines := bufio.NewReader(fp)
	lineno := 0
	keepGoing := true
	for keepGoing {
		line, err := fpLines.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			keepGoing = false
		}
		lineno++
		// strip comments and whitespace
		commentStart := len(line)
		for _, commentPrefix := range p.InlineCommentPrefixes {
			index := strings.Index(line, commentPrefix)
			if index > 0 && index < commentStart {
				commentStart = index
			}
		}
		for _, commentPrefix := range p.CommentPrefixes {
			if strings.HasPrefix(strings.TrimSpace(line), commentPrefix) {
				commentStart = 0
				break
			}
		}
		value := strings.TrimSpace(line[:commentStart])
		// handle empty lines
		if value == "" {
			if p.EmptyLinesInValues {
				// append empty line to the value (if there is one!), but only if
				// there was no comment.
				if curVal != nil && commentStart == len(line) {
					curVal = append(curVal, value)
				}
			} else {
				curIndentLevel = 0
			}
			continue
		}

		lineIndentLevel := 0
		for i, r := range line {
			if !unicode.IsSpace(r) {
				lineIndentLevel = i
				break
			}
		}
		if curVal != nil && lineIndentLevel > 0 && lineIndentLevel > curIndentLevel {
			// continuation line
			curVal = append(curVal, value)
		} else if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			// section header
			flushKV()
			curIndentLevel = lineIndentLevel
			sectName := strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
			if _, exists := config[sectName]; !exists {
				config[sectName] = make(ConfigSection)
			} else if p.Strict {
				return nil, fmt.Errorf("line %d: duplicate section name %q", lineno, sectName)
			}
			curSection = config[sectName]
		} else {
			// start of a k/v pair
			flushKV()
			curIndentLevel = lineIndentLevel
			if curSection == nil {
				return nil, fmt.Errorf("line %d: no section header", lineno)
			}
			sepPos := len(value)
			sepLen := 0
			for _, sep := range p.Delimiters {
				if index := strings.Index(value, sep); index >= 0 {
					if index < sepPos {
						sepPos = index
						sepLen = len(sep)
					}
				}
			}
			if sepPos == len(value) {
				return nil, fmt.Errorf("line %d: invalid line: %q", lineno, value)
			}
			curKey = p.OptionTransform(strings.TrimSpace(value[:sepPos]))
			curVal = []string{
				strings.TrimSpace(value[sepPos+sepLen:]),
			}
			if _, exists := curSection[curKey]; p.Strict && exists {
				return nil, fmt.Errorf("line %d: duplicate option name %q", lineno, curKey)
			}
		}
	}
	flushKV()

	for sect := range config {
		for key, val := range config[sect] {
			var err error
			config[sect][key], err = p.Interpolate(config, val)
			if err != nil {
				return nil, err
			}
		}
	}

	return config, nil
}

func NoInterpolation(_ Config, val string) (string, error) {
	return val, nil
}
