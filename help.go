package opts

import (
	"strings"
	"text/tabwriter"
	"text/template"
)

var usageTemplateString = `{{.Name}}{{if .Flags}} [OPTIONS]{{end}}{{if and .Commands (not .Command)}} <COMMAND>{{end}}`
var helpTemplateString = `USAGE:
    {{.Usage}}
{{- if .Flags}}

OPTIONS:
{{- range .Flags}}
\t    \t{{if .ShortName}}-{{.ShortName}}, {{end}}--{{.Name}}{{if .HasArg}} <{{if .Placeholder}}{{.Placeholder}}{{else}}VALUE{{end}}>{{end}}\t{{if and .HasArg .Default}}  (default: {{.Default}}){{else if .Required}}  (required){{end}}\t{{if .Help}}  {{.Help}}{{end}}
{{- end}}
{{- if .Args}}

ARGUMENTS:
{{- range .Args}}
\t    \t<{{if .Placeholder}}{{.Placeholder}}{{else}}VALUE{{end}}>{{end}}\t  {{.Help}}
{{- end}}
{{- end}}
{{- if .Commands}}

COMMANDS:
{{- range .Commands}}
\t    \t{{.Name}}
{{- end}}
{{- end}}

`
var helpTemplate *template.Template

var usageTemplate *template.Template

func init() {
	replacer := strings.NewReplacer(`\t`, "\t", `\f`, "\f")
	preRenderHelpTemplateString := replacer.Replace(helpTemplateString)
	helpTemplate = template.Must(
		template.New("help").Parse(preRenderHelpTemplateString),
	)
	usageTemplate = template.Must(
		template.New("usage").Parse(usageTemplateString),
	)
}

func (opts *Opts) usage(command string) string {
	flags := []field{}
	args := []field{}
	for _, f := range opts.fields {
		flags = append(flags, f)
	}
	commands := []*Opts{}
	for _, cmd := range opts.commands {
		commands = append(commands, cmd)
	}
	data := struct {
		Name     string
		Flags    []field
		Args     []field
		Commands []*Opts
		Command  string
	}{
		Name:     opts.Name,
		Flags:    flags,
		Args:     args,
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

func (opts *Opts) Help() string {
	flags := []field{}
	args := []field{}
	for _, f := range opts.fields {
		flags = append(flags, f)
	}
	commands := []*Opts{}
	for _, cmd := range opts.commands {
		commands = append(commands, cmd)
	}
	data := struct {
		Usage    string
		Flags    []field
		Args     []field
		Commands []*Opts
	}{
		Usage:    opts.usage(""),
		Flags:    flags,
		Args:     args,
		Commands: commands,
	}

	sb := strings.Builder{}
	w := tabwriter.NewWriter(&sb, 0, 0, 0, ' ', 0)
	err := helpTemplate.Execute(w, data)
	if err != nil {
		panic(err)
	}
	w.Flush()
	return sb.String()
}
