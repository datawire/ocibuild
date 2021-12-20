package python

import (
	"io"
)

type Config map[string]ConfigSection

type ConfigSection map[string]string

type ConfigParser struct {
	// TODO
}

func (p *ConfigParser) Parse(io.Reader) (Config, error) {
	// TODO
	return nil, nil
}
