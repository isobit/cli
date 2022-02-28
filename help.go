package opts

import (
	"io"
	"strings"
	"text/tabwriter"
	"text/template"
)

var usageTemplateString = `{{.Name}}{{if .Flags}} [OPTIONS]{{end}}{{if and .Commands (not .Command)}} <COMMAND>{{end}}`
var helpTemplateString = `
{{- if .Help -}}
{{.Help}}

{{end -}}
USAGE:
    {{.Usage}}

{{- if .Flags}}

OPTIONS:
{{- range .Flags}}
\t    \t
{{- if .ShortName}}-{{.ShortName}}, {{end}}--{{.Name}}
{{- if .HasArg}} <{{if .Placeholder}}{{.Placeholder}}{{else}}VALUE{{end}}>{{end}}\t
{{- if .Help}}  {{.Help}}{{end}}
{{- if and .HasArg .Default (not .Required)}}  (default: {{.Default}}){{end}}
{{- end}}

{{- end}}

{{- if .Commands}}

COMMANDS:
{{- range .Commands}}
\t    \t{{.Name}}\t
{{- if .ShortHelp}}  {{.ShortHelp}}{{end}}
{{- end}}

{{- end}}

`
var helpTemplate *template.Template

var usageTemplate *template.Template

func init() {
	helpTemplate = template.Must(
		template.New("help").Parse(helpTemplateString),
	)
	usageTemplate = template.Must(
		template.New("usage").Parse(usageTemplateString),
	)
}

func (opts *Opts) usage(command string) string {
	commands := []*Opts{}
	for _, cmd := range opts.commands {
		commands = append(commands, cmd)
	}
	data := struct {
		Name     string
		Flags    []field
		Commands []*Opts
		Command  string
	}{
		Name:     opts.Name,
		Flags:    opts.fields,
		Commands: commands,
		Command:  command,
	}

	sb := strings.Builder{}
	if opts.parent != nil {
		sb.WriteString(opts.parent.usage(opts.Name))
		sb.WriteString(" ")
	}
	usageTemplate.Execute(&sb, data)
	return sb.String()
}

func (opts *Opts) HelpString() string {
	sb := strings.Builder{}
	opts.WriteHelp(&sb)
	return sb.String()
}

func (opts *Opts) WriteHelp(w io.Writer) {
	flags := []field{}
	for _, f := range opts.fields {
		flags = append(flags, f)
	}
	commands := []*Opts{}
	for _, cmd := range opts.commands {
		commands = append(commands, cmd)
	}
	data := struct {
		Usage    string
		Help     string
		Flags    []field
		Commands []*Opts
	}{
		Usage:    opts.usage(""),
		Help:     opts.Help,
		Flags:    flags,
		Commands: commands,
	}

	tw := newEscapedTabWriter(w)
	err := helpTemplate.Execute(tw, data)
	if err != nil {
		panic(err)
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
