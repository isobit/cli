package opts

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

var errWriter io.Writer = os.Stderr

type Opts struct {
	Name           string
	Help           string
	ShortHelp      string
	parent         *Opts
	config         interface{}
	internalConfig internalConfig
	fields         []field
	flagset        *flag.FlagSet
	commands       map[string]*Opts
}

type internalConfig struct {
	Help bool `opts:"short=h,help=show usage help"`
}

// Create a new Opts with the provider name and config. The config must be a
// pointer to a configuration struct. Default values for fields can be
// specified by simply setting them on the passed in struct value.
//
// The parsing behavior for config fields can be controlled with the following
// struct field tags, specified like `opts:"key1,key2=value2"`:
//
// `required` return a usage error if the field is not set explicitly
//
// `help=<text>` help text to be printed with the flag usage
//
// `placeholder=<text>` custom placeholder to use in the flag usage (the
// default placeholder is "VALUE")
//
// `name=<name>` override the flag name derived from the field name with a
// custom one
//
// `short=<shortname>` adds a short flag alias for the field; must be 1 char
//
// `env=<varName>` parse the value from the specified environment variable name
// if it is not set via args
func New(name string, config interface{}) *Opts {
	opts := Opts{
		Name:     name,
		parent:   nil,
		config:   config,
		commands: map[string]*Opts{},
	}

	fields, err := getFieldsFromConfig(config)
	if err != nil {
		panic(err)
	}

	internalFields, err := getFieldsFromConfig(&opts.internalConfig)
	if err != nil {
		panic(err)
	}
	fields = append(internalFields, fields...)

	opts.fields = fields

	opts.flagset = flag.NewFlagSet(name, flag.ContinueOnError)
	opts.flagset.SetOutput(ioutil.Discard)
	for _, f := range fields {
		opts.flagset.Var(f.flagValue, f.Name, f.Help)
		if f.ShortName != "" {
			opts.flagset.Var(f.flagValue, f.ShortName, f.Help)
		}
	}

	return &opts
}

func (opts *Opts) SetHelp(help string) *Opts {
	opts.Help = help
	return opts
}

func (opts *Opts) SetShortHelp(help string) *Opts {
	opts.ShortHelp = help
	return opts
}

// AddCommand registers another Opts instance as a subcommand of this Opts
// instance.
func (opts *Opts) AddCommand(cmdOpts *Opts) *Opts {
	cmdOpts.parent = opts
	opts.commands[cmdOpts.Name] = cmdOpts
	return opts
}

func (opts *Opts) AddCommands(cmds []*Opts) *Opts {
	for _, cmd := range cmds {
		opts.AddCommand(cmd)
	}
	return opts
}

// Parse is a shortcut for calling `ParseArgs(os.Args)`
func (opts *Opts) Parse() ParsedOpts {
	return opts.ParseArgs(os.Args)
}

// ParseEnvVars sets any unset field values using the environment variable
// matching the "env" tag of the field, if present.
func (opts *Opts) ParseEnvVars() error {
	for _, f := range opts.fields {
		if f.EnvVarName == "" || f.flagValue.setCount > 0 {
			continue
		}
		if s, ok := os.LookupEnv(f.EnvVarName); ok {
			if err := f.flagValue.Set(s); err != nil {
				return err
			}
		}
	}
	return nil
}

// Returns an error if any fields are required but have not been set.
func (opts *Opts) CheckRequired() error {
	for _, f := range opts.fields {
		if f.Required && f.flagValue.setCount < 1 {
			return fmt.Errorf("required flag %s not set", f.Name)
		}
	}
	return nil
}

// Parse parses using the passed-in args slice and OS-provided environment
// variables and returns a ParsedOpts instance which can be used for further
// method chaining.
func (opts *Opts) ParseArgs(args []string) ParsedOpts {
	po := ParsedOpts{Opts: opts}

	// if we're the root, the first arg is the program name and should be
	// skipped.
	if opts.parent == nil && len(args) > 0 {
		args = args[1:]
	}

	if err := opts.flagset.Parse(args); err != nil {
		return po.err(errors.Wrap(err, "failed to parse args"))
	}

	if opts.internalConfig.Help {
		return po.err(flag.ErrHelp)
	}

	if err := opts.ParseEnvVars(); err != nil {
		return po.err(errors.Wrap(err, "failed to environment variables"))
	}

	if err := opts.CheckRequired(); err != nil {
		return po.err(err)
	}

	if beforer, ok := opts.config.(Beforer); ok {
		err := beforer.Before()
		if err != nil {
			return po.err(err)
		}
	}

	rargs := opts.flagset.Args()
	if len(rargs) > 0 {
		cmdName := rargs[0]
		if cmd, ok := opts.commands[cmdName]; ok {
			return cmd.ParseArgs(rargs[1:])
		} else {
			return po.err(fmt.Errorf("unknown command %s", cmdName))
		}
	}

	runner, isRunnable := opts.config.(Runner)
	if !isRunnable && len(opts.commands) > 0 {
		return po.err(fmt.Errorf("no command specified"))
	}
	po.runner = runner

	return po
}

type ParsedOpts struct {
	Err    error
	Opts   *Opts
	runner Runner
}

// Convenience method for returning errors wrapped as ParsedOpts.
func (po ParsedOpts) err(err error) ParsedOpts {
	po.Err = err
	return po
}

// Run calls the Run method of the underlying Opts config or, if an error
// occurred during parsing, prints the help text and returns that error
// instead.
func (po ParsedOpts) Run() error {
	if po.Err != nil {
		po.Opts.WriteHelp(errWriter)
		return po.Err
	}
	if po.runner == nil {
		return fmt.Errorf("no run method implemented")
	}
	return po.runner.Run()
}

// RunFatal is like Run, except it automatically handles printing out any
// errors returned by the Run method of the underlying Opts config, and exits
// with an appropriate status (1 if error, 0 otherwise).
func (po ParsedOpts) RunFatal() {
	err := po.Run()
	if err != nil {
		fmt.Fprintf(errWriter, "error: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

type Runner interface {
	Run() error
}

type Beforer interface {
	Before() error
}
