package cli

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

type Runner interface {
	Run() error
}

type ContextRunner interface {
	Run(context.Context) error
}

type Beforer interface {
	Before() error
}

type Setuper interface {
	SetupCommand(cmd *Command)
}

type ExitCoder interface {
	ExitCode() int
}

type Command struct {
	cli             CLI
	name            string
	help            string
	description     string
	config          interface{}
	internalConfig  internalConfig
	fields          []field
	flagset         *flag.FlagSet
	flagsetInternal *flag.FlagSet
	parent          *Command
	commands        []*Command
	commandMap      map[string]*Command
}

type internalConfig struct {
	Help bool `cli:"short=h,help=show usage help"`
}

func (cli CLI) New(name string, config interface{}, opts ...CommandOption) *Command {
	cmd, err := cli.Build(name, config, opts...)
	if err != nil {
		panic(fmt.Sprintf("cli: %s", err))
	}
	return cmd
}

func (cli CLI) Build(name string, config interface{}, opts ...CommandOption) (*Command, error) {
	cmd := &Command{
		cli:        cli,
		name:       name,
		config:     config,
		fields:     []field{},
		commands:   []*Command{},
		commandMap: map[string]*Command{},
	}

	internalFields, err := cli.getFieldsFromConfig(&cmd.internalConfig)
	if err != nil {
		return nil, fmt.Errorf("error building internal config: %w", err)
	}
	cmd.fields = append(cmd.fields, internalFields...)
	cmd.flagsetInternal = newFlagSet(name, internalFields)

	configFields, err := cli.getFieldsFromConfig(config)
	if err != nil {
		return nil, err
	}
	cmd.fields = append(cmd.fields, configFields...)
	cmd.flagset = newFlagSet(name, cmd.fields)

	if setuper, ok := cmd.config.(Setuper); ok {
		setuper.SetupCommand(cmd)
	}

	for _, opt := range opts {
		opt.Apply(cmd)
	}

	return cmd, nil
}

func newFlagSet(name string, fields []field) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(ioutil.Discard)
	for _, f := range fields {
		fs.Var(f.flagValue, f.Name, f.Help)
		if f.ShortName != "" {
			fs.Var(f.flagValue, f.ShortName, f.Help)
		}
	}
	return fs
}

func (cmd *Command) SetHelp(help string) *Command {
	cmd.help = help
	return cmd
}

func (cmd *Command) SetDescription(description string) *Command {
	cmd.description = description
	return cmd
}

// AddCommand registers another Command instance as a subcommand of this Command
// instance.
func (cmd *Command) AddCommand(subCmd *Command) *Command {
	subCmd.parent = cmd
	cmd.commands = append(cmd.commands, subCmd)
	cmd.commandMap[subCmd.name] = subCmd
	return cmd
}

func (cmd *Command) Apply(parent *Command) {
	parent.AddCommand(cmd)
}

// Parse is a convenience method for calling ParseArgs(os.Args)
func (cmd *Command) Parse() ParseResult {
	return cmd.ParseArgs(os.Args)
}

// ParseArgs parses using the passed-in args slice and OS-provided environment
// variables and returns a ParseResult which can be used for further method
// chaining.
//
// If there are args remaining after parsing this Command's fields, subcommands
// will be recursively parsed until a concrete result is returned. If a Before
// method is implemented on the config, this method will call it before
// recursing into any subcommand parsing.
func (cmd *Command) ParseArgs(args []string) ParseResult {
	r := ParseResult{Command: cmd}

	if cmd.parent == nil && len(args) > 0 {
		// if we're the root, the first arg is the program name and should be
		// skipped.
		args = args[1:]

		// Do a minimal recursive parsing pass (only on the internal flagset)
		// so we can exit early with help if the help flag is passed on this
		// command or any subcommand before proceeding.
		helpParsedArgs := cmd.helpPass(args)
		if helpParsedArgs.Err == ErrHelp {
			return helpParsedArgs
		}
	}

	// Parse arguments using the flagset.
	if err := cmd.flagset.Parse(args); err != nil {
		return r.err(fmt.Errorf("failed to parse args: %w", err))
	}

	// Return ErrHelp if help was requested.
	if cmd.internalConfig.Help {
		return r.err(ErrHelp)
	}

	// Parse environment variables.
	if err := cmd.parseEnvVars(); err != nil {
		return r.err(fmt.Errorf("failed to parse environment variables: %w", err))
	}

	// Return an error if any required fields were not set at least once.
	if err := cmd.checkRequired(); err != nil {
		return r.err(err)
	}

	// If the config implements a Before method, run it before we recursively
	// parse subcommands.
	if beforer, ok := cmd.config.(Beforer); ok {
		err := beforer.Before()
		if err != nil {
			return r.err(err)
		}
	}

	// Handle remaining arguments, recursively parse subcommands.
	rargs := cmd.flagset.Args()
	if len(rargs) > 0 {
		cmdName := rargs[0]
		if cmd, ok := cmd.commandMap[cmdName]; ok {
			return cmd.ParseArgs(rargs[1:])
		} else {
			return r.err(fmt.Errorf("unknown command %s", cmdName))
		}
	}

	r.runInfo = getRunInfo(cmd.config)
	if r.runInfo == nil && len(cmd.commands) != 0 {
		return r.err(fmt.Errorf("no command specified"))
	}

	return r
}

type runInfo struct {
	run             func(context.Context) error
	supportsContext bool
}

func getRunInfo(config interface{}) *runInfo {
	if r, ok := config.(Runner); ok {
		run := func(context.Context) error {
			return r.Run()
		}
		return &runInfo{
			run:             run,
			supportsContext: false,
		}
	}
	if r, ok := config.(ContextRunner); ok {
		return &runInfo{
			run:             r.Run,
			supportsContext: true,
		}
	}
	return nil
}

// helpPass does a minimal recursive parsing pass using only the internal
// flagset, so that help flags can be detected on subcommands early.
func (cmd *Command) helpPass(args []string) ParseResult {
	r := ParseResult{Command: cmd}

	// Parse arguments using the flagset.
	// Intentionally ignore errors since we want to ignore any non-internal
	// flags.
	_ = cmd.flagsetInternal.Parse(args)

	// Return ErrHelp if help was requested.
	if cmd.internalConfig.Help {
		return r.err(ErrHelp)
	}

	// Handle remaining arguments, recursively parse subcommands.
	rargs := cmd.flagsetInternal.Args()
	if len(rargs) > 0 {
		cmdName := rargs[0]
		if cmd, ok := cmd.commandMap[cmdName]; ok {
			return cmd.helpPass(rargs[1:])
		}
	}

	return r
}

// parseEnvVars sets any unset field values using the environment variable
// matching the "env" tag of the field, if present.
func (cmd *Command) parseEnvVars() error {
	for _, f := range cmd.fields {
		if f.EnvVarName == "" || f.flagValue.setCount > 0 {
			continue
		}
		val, ok, err := cmd.cli.LookupEnv(f.EnvVarName)
		if err != nil {
			// TODO?
			return err
		}
		if ok {
			if err := f.flagValue.Set(val); err != nil {
				return fmt.Errorf("error parsing %s: %w", f.EnvVarName, err)
			}
		}
	}
	return nil
}

// checkRequired returns an error if any fields are required but have not been set.
func (cmd *Command) checkRequired() error {
	for _, f := range cmd.fields {
		if f.Required && f.flagValue.setCount < 1 {
			return fmt.Errorf("required flag %s not set", f.Name)
		}
	}
	return nil
}

// ParseResult contains information about the results of command argument
// parsing.
type ParseResult struct {
	Err     error
	Command *Command
	runInfo *runInfo
}

// Convenience method for returning errors wrapped as a ParsedResult.
func (r ParseResult) err(err error) ParseResult {
	r.Err = err
	return r
}

// Run calls the Run method of the Command config for the parsed command or, if
// an error occurred during parsing, prints the help text and returns that
// error instead. If help was requested, the error will flag.ErrHelp. If the
// underlying command Run method accepts a context, context.Background() will
// be passed.
func (r ParseResult) Run() error {
	return r.RunWithContext(context.Background())
}

// RunWithContext is like Run, but it accepts an explicit context which will be
// passed to the command's Run method, if it accepts one.
func (r ParseResult) RunWithContext(ctx context.Context) error {
	if r.Err != nil {
		if r.Command != nil && r.Command.cli.HelpWriter != nil {
			r.Command.WriteHelp(r.Command.cli.HelpWriter)
		}
		return r.Err
	}
	if r.runInfo == nil {
		return fmt.Errorf("no run method implemented")
	}
	return r.runInfo.run(ctx)
}

// RunWithSigCancel is like Run, but it automatically registers a signal
// handler for SIGINT and SIGTERM that will cancel the context that is passed
// to the command's Run method, if it accepts one.
func (r ParseResult) RunWithSigCancel() error {
	ctx, stop := r.contextWithSigCancelIfSupported(context.Background())
	defer stop()
	return r.RunWithContext(ctx)
}

// RunFatal is like Run, except it automatically handles printing out any
// errors returned by the Run method of the underlying Command config, and
// exits with an appropriate status code.
//
// If no error occurs, the exit code will be 0. If an error is returned and it
// implements the ExitCoder interface, the result of ExitCode() will be used as
// the exit code. If an error is returned that does not implement ExitCoder,
// the exit code will be 1.
func (r ParseResult) RunFatal() {
	r.RunFatalWithContext(context.Background())
}

// RunFatalWithContext is like RunFatal, but it accepts an explicit context
// which will be passed to the command's Run method if it accepts one.
func (r ParseResult) RunFatalWithContext(ctx context.Context) {
	err := r.RunWithContext(ctx)
	if err != nil {
		if err != ErrHelp && r.Command != nil && r.Command.cli.ErrWriter != nil {
			fmt.Fprintf(r.Command.cli.ErrWriter, "error: %s\n", err)
		}
		if ec, ok := err.(ExitCoder); ok {
			os.Exit(ec.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
}

// RunFatalWithSigCancel is like RunFatal, but it automatically registers a
// signal handler for SIGINT and SIGTERM that will cancel the context that is
// passed to the command's Run method, if it accepts one.
func (r ParseResult) RunFatalWithSigCancel() {
	ctx, stop := r.contextWithSigCancelIfSupported(context.Background())
	defer stop()
	r.RunFatalWithContext(ctx)
}

func (r ParseResult) contextWithSigCancelIfSupported(ctx context.Context) (context.Context, context.CancelFunc) {
	if r.runInfo == nil || !r.runInfo.supportsContext {
		return ctx, func() {}
	}
	return signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
}

type CommandOption interface {
	Apply(cmd *Command)
}

type commandOptionFunc func(cmd *Command)

func (of commandOptionFunc) Apply(cmd *Command) {
	of(cmd)
}

func WithHelp(help string) CommandOption {
	return commandOptionFunc(func(cmd *Command) {
		cmd.SetHelp(help)
	})
}

func WithDescription(description string) CommandOption {
	return commandOptionFunc(func(cmd *Command) {
		cmd.SetDescription(description)
	})
}
