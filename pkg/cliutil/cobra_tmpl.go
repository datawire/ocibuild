package cliutil

import (
	"github.com/spf13/cobra"
)

func init() {
	cobra.AddTemplateFunc("getTerminalWidth", GetTerminalWidth)
	cobra.AddTemplateFunc("wrap", Wrap)
	cobra.AddTemplateFunc("wrapIndent", WrapIndent)
	cobra.AddTemplateFunc("add", func(args ...int) int {
		ret := 0
		for _, arg := range args {
			ret += arg
		}
		return ret
	})
}

const HelpTemplate = `Usage: {{ .UseLine }}

{{- /* Short help text ---------------------------------------------------- */}}
{{- if .Short }}
{{ .Short }}
{{- end }}

{{- /* Long help text ----------------------------------------------------- */}}
{{- if .Long }}

{{ .Long | wrap getTerminalWidth | trimTrailingWhitespaces }}
{{- end }}

{{- /* Aliases ------------------------------------------------------------ */}}
{{- if .Aliases }}

Aliases:
  {{ .NameAndAliases }}
{{- end }}

{{- /* Aliases ------------------------------------------------------------ */}}
{{- if .HasExample }}

Examples:
{{ .Example }}
{{- end }}

{{- /* Subcommands -------------------------------------------------------- */}}
{{- if .HasAvailableSubCommands }}

Available Commands:
{{- range .Commands}}
  {{- if (or .IsAvailableCommand (eq .Name "help")) }}
    {{- "\n" }}  {{ rpad .Name .NamePadding }}   {{ .Short | wrapIndent (add .NamePadding 5) getTerminalWidth }}
  {{- end }}
{{- end }}
{{- end }}

{{- /* Local Flags -------------------------------------------------------- */}}
{{- if .HasAvailableLocalFlags }}

Flags:
{{ getTerminalWidth | .LocalFlags.FlagUsagesWrapped | trimTrailingWhitespaces }}
{{- end }}

{{- /* Global flags ------------------------------------------------------- */}}
{{- if .HasAvailableInheritedFlags }}

Global Flags:
{{ getTerminalWidth | .InheritedFlags.FlagUsagesWrapped | trimTrailingWhitespaces }}
{{- end }}

{{- /* Help topics -------------------------------------------------------- */}}
{{- if .HasHelpSubCommands }}

Additional help topics:
{{- range .Commands }}
  {{- if .IsAdditionalHelpTopicCommand }}
    {{- "\n" }}  {{ rpad .CommandPath .CommandPathPadding }}   {{ .Short | wrapIndent (add .NamePadding 5) getTerminalWidth }}
  {{- end }}
{{- end }}
{{- end }}

{{- /* Help footer -------------------------------------------------------- */}}
{{- if .HasAvailableSubCommands }}

Use "{{ .CommandPath }} [command] --help" for more information about a command.
{{- end}}
`
