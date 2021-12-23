//Package entry_points implementes the PyPA Entry points specification.
//
// https://packaging.python.org/en/latest/specifications/entry-points/
package entry_points

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/python"
	"github.com/datawire/ocibuild/pkg/python/pypa/bdist"
)

var (
	scriptTmpl = template.Must(template.
			New("entry_point.py").
			Parse(`#!{{ .Shebang }}
# -*- coding: utf-8 -*-
import re
import sys
from {{ .Module }} import {{ .Func }}
if __name__ == '__main__':
    sys.argv[0] = re.sub(r'(-script\.pyw|\.exe)?$', '', sys.argv[0])
    sys.exit({{ .Func }}())
`))

	configParser = func() *python.ConfigParser {
		configParser := python.NewConfigParser()
		configParser.OptionTransform = func(str string) string { return str }
		configParser.Delimiters = []string{"="}
		return configParser
	}()

	// This is lax on validation of the [extras] part, but that's OK; we don't care about that
	// part.
	reFuncRef = regexp.MustCompile(`^(?P<callable>\w+([:.]\w+)*)(?:\s*\[.*\])?$`)
)

func CreateScripts(plat python.Platform) bdist.PostInstallHook {
	return func(ctx context.Context, clampTime time.Time, vfs map[string]fsutil.FileReference, installedDistInfoDir string) error {
		if err := plat.Init(); err != nil {
			return err
		}
		configFile, ok := vfs[path.Join(installedDistInfoDir, "entry_points.txt")]
		if !ok {
			return nil
		}
		configReader, err := configFile.Open()
		if err != nil {
			return err
		}

		configData, err := configParser.Parse(configReader)
		if err != nil {
			return err
		}

		interesting := map[string]string{
			"console_scripts": plat.ConsoleShebang,
			"gui_scripts":     plat.GraphicalShebang,
		}

		for sectionName, shebang := range interesting {
			sectionData, ok := configData[sectionName]
			if !ok {
				continue
			}
			for k, v := range sectionData {
				m := reFuncRef.FindStringSubmatch(v)
				if m == nil {
					return fmt.Errorf("entry_points.txt: %q: %q: not a function reference: %q", sectionName, k, v)
				}
				funcRef := m[reFuncRef.SubexpIndex("callable")]
				parts := strings.Split(funcRef, ":")
				if len(parts) != 2 {
					return fmt.Errorf("entry_points.txt: %q: %q: not a function reference: %q", sectionName, k, v)
				}
				var buf bytes.Buffer
				if err := scriptTmpl.Execute(&buf, map[string]string{
					"Shebang":    shebang,
					"Module":     parts[0],
					"ImportName": strings.SplitN(parts[1], ".", 2)[0],
					"Func":       parts[1],
				}); err != nil {
					return fmt.Errorf("%s: %s: %w", sectionName, k, err)
				}
				header := &tar.Header{
					Typeflag: tar.TypeReg,
					Name:     path.Join(plat.Scheme.Scripts[1:], k),
					Mode:     0o755,
					Size:     int64(buf.Len()),
					ModTime:  clampTime,
				}
				vfs[header.Name] = &fsutil.InMemFileReference{
					FileInfo:  header.FileInfo(),
					MFullName: header.Name,
					MContent:  buf.Bytes(),
				}
			}
		}
		return nil
	}
}
