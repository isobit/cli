package opts

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

var errWriter io.Writer = os.Stderr

type Opts struct {
	Name           string
	ShortName      string
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

// New creates a new Opts with the provided name and config. The config must be
// a pointer to a configuration struct. Default values can be specified by
// simply setting them on the config struct.
//
// New returns an Opts pointer for further method chaining. If an error is
// encountered while building the options, such as a struct field having an
// unsupported type, New will panic. If you would like to have errors returned
// for handling, use Build instead.
func New(name string, config interface{}) *Opts {
	opts, err := Build(name, config)
	if err != nil {
		panic(fmt.Sprintf("opts: %s", err))
	}
	return opts
}

// Build is like New, but it returns any errors instead of calling panic, at
// the expense of being harder to chain.
func Build(name string, config interface{}) (*Opts, error) {
	opts := Opts{
		Name:     name,
		parent:   nil,
		config:   config,
		commands: map[string]*Opts{},
	}

	fields, err := getFieldsFromConfig(config)
	if err != nil {
		return nil, err
	}

	internalFields, err := getFieldsFromConfig(&opts.internalConfig)
	if err != nil {
		return nil, fmt.Errorf("error building internal config: %w", err)
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

	return &opts, nil
}

// SetShortName configures a short alias for the command.
func (opts *Opts) SetShortName(shortName string) *Opts {
	opts.ShortName = shortName
	return opts
}

// SetHelp configures the command help string.
func (opts *Opts) SetHelp(help string) *Opts {
	opts.Help = help
	return opts
}

// SetHelp configures the command short help string.
func (opts *Opts) SetShortHelp(help string) *Opts {
	opts.ShortHelp = help
	return opts
}

// AddCommand registers another Opts instance as a subcommand of this Opts
// instance.
func (opts *Opts) AddCommand(cmdOpts *Opts) *Opts {
	cmdOpts.parent = opts
	opts.commands[cmdOpts.Name] = cmdOpts
	if cmdOpts.ShortName != "" {
		opts.commands[cmdOpts.ShortName] = cmdOpts
	}
	return opts
}

// AddCommands registers multiple Opts instances as subcommands of this Opts
// instance.
func (opts *Opts) AddCommands(cmds []*Opts) *Opts {
	for _, cmd := range cmds {
		opts.AddCommand(cmd)
	}
	return opts
}

// Parse is a convenience method for calling ParseArgs(os.Args)
func (opts *Opts) Parse() ParsedOpts {
	return opts.ParseArgs(os.Args)
}

// ParseArgs parses using the passed-in args slice and OS-provided environment
// variables and returns a ParsedOpts instance which can be used for further
// method chaining.
//
// If there are args remaining after parsing this Opts' fields, subcommands
// will be parsed recursively. The returned ParsedOpts.Runner represents the
// command which was specified (if it has a Run method). If a Before method is
// implemented on the Opts' config, this method will call it before recursing
// into any subcommand parsing.
func (opts *Opts) ParseArgs(args []string) ParsedOpts {
	po := ParsedOpts{Opts: opts}

	// if we're the root, the first arg is the program name and should be
	// skipped.
	if opts.parent == nil && len(args) > 0 {
		args = args[1:]
	}

	// Parse arguments using the flagset.
	if err := opts.flagset.Parse(args); err != nil {
		return po.err(fmt.Errorf("failed to parse args: %w", err))
	}

	// Return flag.ErrHelp if help was requested.
	if opts.internalConfig.Help {
		return po.err(flag.ErrHelp)
	}

	// Parse environment variables.
	if err := opts.parseEnvVars(); err != nil {
		return po.err(fmt.Errorf("failed to parse environment variables: %w", err))
	}

	// Return an error if any required fields were not set at least once.
	if err := opts.checkRequired(); err != nil {
		return po.err(err)
	}

	// If the config implements a Before method, run it before we recursively
	// parse subcommands.
	if beforer, ok := opts.config.(Beforer); ok {
		err := beforer.Before()
		if err != nil {
			return po.err(err)
		}
	}

	// Handle remaining arguments, recursively parse subcommands.
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
	po.Runner = runner

	return po
}

// parseEnvVars sets any unset field values using the environment variable
// matching the "env" tag of the field, if present.
func (opts *Opts) parseEnvVars() error {
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

// checkRequired returns an error if any fields are required but have not been set.
func (opts *Opts) checkRequired() error {
	for _, f := range opts.fields {
		if f.Required && f.flagValue.setCount < 1 {
			return fmt.Errorf("required flag %s not set", f.Name)
		}
	}
	return nil
}

type ParsedOpts struct {
	Err    error
	Opts   *Opts
	Runner Runner
}

// Convenience method for returning errors wrapped as ParsedOpts.
func (po ParsedOpts) err(err error) ParsedOpts {
	po.Err = err
	return po
}

// Run calls the Run method of the Opts config for the parsed command or, if an
// error occurred during parsing, prints the help text and returns that error
// instead. If help was requested, the error will flag.ErrHelp.
func (po ParsedOpts) Run() error {
	if po.Err != nil {
		po.Opts.WriteHelp(errWriter)
		return po.Err
	}
	if po.Runner == nil {
		return fmt.Errorf("no run method implemented")
	}
	return po.Runner.Run()
}

// RunFatal is like Run, except it automatically handles printing out any
// errors returned by the Run method of the underlying Opts config, and exits
// with an appropriate status (1 if error, 0 otherwise).
func (po ParsedOpts) RunFatal() {
	err := po.Run()
	if err != nil {
		if err != flag.ErrHelp {
			fmt.Fprintf(errWriter, "error: %s\n", err)
		}
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
