package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"text/template"
)

var ErrHelp = fmt.Errorf("cli: help requested")

var helpTemplateString = `
{{- if 0}}{{end -}}
USAGE:
    {{.FullName}}{{if .Fields}} [OPTIONS]{{end}}{{if .Commands}} <COMMAND>{{end}}{{if .Args}} [ARGS]{{end}}

{{- if .Fields}}

OPTIONS:
{{- range .Fields}}{{if not .Hidden}}
\t    \t
{{- if .ShortName}}-{{.ShortName}}, {{end}}--{{.Name}}
{{- if .HasArg}} <{{if .Placeholder}}{{.Placeholder}}{{else}}VALUE{{end}}>{{end}}\t
{{- if .EnvVarName}}  {{.EnvVarName}}{{end}}\t
{{- if .Help}}  {{.Help}}{{end}}
{{- if and .HasArg }}{{if and .Default (not .Required)}}  (default: {{.Default}}){{else if .Required}}  (required){{end}}{{end}}
{{- end}}

{{- end}}{{end}}

{{- if .Commands}}

COMMANDS:
{{- range .Commands}}
\t    \t{{.Name}}\t{{ if .Help}}  {{.Help}}{{end}}
{{- end}}

{{- end}}

{{- if .Description}}

DESCRIPTION:
    {{.Description}}
{{- end}}

`

var helpTemplate *template.Template

func init() {
	helpTemplate = template.Must(
		template.New("help").Parse(helpTemplateString),
	)
}

func (cmd *Command) fullName() string {
	sb := strings.Builder{}
	if cmd.parent != nil {
		sb.WriteString(cmd.parent.fullName())
		sb.WriteString(" ")
	}
	sb.WriteString(cmd.name)
	return sb.String()
}

func (cmd *Command) HelpString() string {
	sb := strings.Builder{}
	cmd.WriteHelp(&sb)
	return sb.String()
}

func (cmd *Command) WriteHelp(w io.Writer) {
	type subcommandData struct {
		Name string
		Help string
	}
	data := struct {
		FullName    string
		Description string
		Fields      []field
		Commands    []subcommandData
		Args        bool
	}{
		FullName:    cmd.fullName(),
		Description: strings.ReplaceAll(strings.TrimSpace(cmd.description), "\n", "\n    "),
		Fields:      cmd.fields,
		Commands:    []subcommandData{},
		Args:        cmd.argsField != nil,
	}
	for _, cmd := range cmd.commands {
		data.Commands = append(data.Commands, subcommandData{
			Name: cmd.name,
			Help: cmd.help,
		})
	}

	tw := newEscapedTabWriter(w)
	err := helpTemplate.Execute(tw, data)
	if err != nil {
		panic(fmt.Sprintf("cli: error executing help template: %s", err))
	}
	tw.Flush()
}

type escapedTabWriter struct {
	replacer  *strings.Replacer
	tabWriter *tabwriter.Writer
}

func newEscapedTabWriter(w io.Writer) escapedTabWriter {
	return escapedTabWriter{
		replacer:  strings.NewReplacer(`\t`, "\t", `\f`, "\f"),
		tabWriter: tabwriter.NewWriter(w, 0, 0, 0, ' ', 0),
	}
}

func (w escapedTabWriter) Write(p []byte) (int, error) {
	return w.replacer.WriteString(w.tabWriter, string(p))
}

func (w escapedTabWriter) Flush() error {
	return w.tabWriter.Flush()
}
