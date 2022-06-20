package cli

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

var errWriter io.Writer = os.Stderr

type Command struct {
	Name           string
	ShortName      string
	Help           string
	ShortHelp      string
	parent         *Command
	config         interface{}
	internalConfig internalConfig
	fields         []field
	flagset        *flag.FlagSet
	commands       map[string]*Command
}

type internalConfig struct {
	Help bool `cli:"short=h,help=show usage help"`
}

// New creates a new Command with the provided name and config. The config must be
// a pointer to a configuration struct. Default values can be specified by
// simply setting them on the config struct.
//
// New returns an Command pointer for further method chaining. If an error is
// encountered while building the options, such as a struct field having an
// unsupported type, New will panic. If you would like to have errors returned
// for handling, use Build instead.
func New(name string, config interface{}) *Command {
	cmd, err := Build(name, config)
	if err != nil {
		panic(fmt.Sprintf("cli: %s", err))
	}
	return cmd
}

// Build is like New, but it returns any errors instead of calling panic, at
// the expense of being harder to chain.
func Build(name string, config interface{}) (*Command, error) {
	cmd := Command{
		Name:     name,
		parent:   nil,
		config:   config,
		commands: map[string]*Command{},
	}

	fields, err := getFieldsFromConfig(config)
	if err != nil {
		return nil, err
	}

	internalFields, err := getFieldsFromConfig(&cmd.internalConfig)
	if err != nil {
		return nil, fmt.Errorf("error building internal config: %w", err)
	}
	fields = append(internalFields, fields...)

	cmd.fields = fields

	cmd.flagset = flag.NewFlagSet(name, flag.ContinueOnError)
	cmd.flagset.SetOutput(ioutil.Discard)
	for _, f := range fields {
		cmd.flagset.Var(f.flagValue, f.Name, f.Help)
		if f.ShortName != "" {
			cmd.flagset.Var(f.flagValue, f.ShortName, f.Help)
		}
	}

	return &cmd, nil
}

// SetShortName configures a short alias for the command.
func (cmd *Command) SetShortName(shortName string) *Command {
	cmd.ShortName = shortName
	return cmd
}

// SetHelp configures the command help string.
func (cmd *Command) SetHelp(help string) *Command {
	cmd.Help = help
	return cmd
}

// SetHelp configures the command short help string.
func (cmd *Command) SetShortHelp(help string) *Command {
	cmd.ShortHelp = help
	return cmd
}

// AddCommand registers another Command instance as a subcommand of this Command
// instance.
func (cmd *Command) AddCommand(subCmd *Command) *Command {
	subCmd.parent = cmd
	cmd.commands[subCmd.Name] = subCmd
	if subCmd.ShortName != "" {
		cmd.commands[subCmd.ShortName] = subCmd
	}
	return cmd
}

// AddCommands registers multiple Command instances as subcommands of this Command
// instance.
func (cmd *Command) AddCommands(cmds []*Command) *Command {
	for _, cmd := range cmds {
		cmd.AddCommand(cmd)
	}
	return cmd
}

// Parse is a convenience method for calling ParseArgs(os.Args)
func (cmd *Command) Parse() ParsedCommand {
	return cmd.ParseArgs(os.Args)
}

// ParseArgs parses using the passed-in args slice and OS-provided environment
// variables and returns a ParsedCommand instance which can be used for further
// method chaining.
//
// If there are args remaining after parsing this Command' fields, subcommands
// will be parsed recursively. The returned ParsedCommand.Runner represents the
// command which was specified (if it has a Run method). If a Before method is
// implemented on the Command' config, this method will call it before
// recursing into any subcommand parsing.
func (cmd *Command) ParseArgs(args []string) ParsedCommand {
	po := ParsedCommand{Command: cmd}

	// if we're the root, the first arg is the program name and should be
	// skipped.
	if cmd.parent == nil && len(args) > 0 {
		args = args[1:]
	}

	// Parse arguments using the flagset.
	if err := cmd.flagset.Parse(args); err != nil {
		return po.err(fmt.Errorf("failed to parse args: %w", err))
	}

	// Return ErrHelp if help was requested.
	if cmd.internalConfig.Help {
		return po.err(ErrHelp)
	}

	// Parse environment variables.
	if err := cmd.parseEnvVars(); err != nil {
		return po.err(fmt.Errorf("failed to parse environment variables: %w", err))
	}

	// Return an error if any required fields were not set at least once.
	if err := cmd.checkRequired(); err != nil {
		return po.err(err)
	}

	// If the config implements a Before method, run it before we recursively
	// parse subcommands.
	if beforer, ok := cmd.config.(Beforer); ok {
		err := beforer.Before()
		if err != nil {
			return po.err(err)
		}
	}

	// Handle remaining arguments, recursively parse subcommands.
	rargs := cmd.flagset.Args()
	if len(rargs) > 0 {
		cmdName := rargs[0]
		if cmd, ok := cmd.commands[cmdName]; ok {
			return cmd.ParseArgs(rargs[1:])
		} else {
			return po.err(fmt.Errorf("unknown command %s", cmdName))
		}
	}

	runner, isRunnable := cmd.config.(Runner)
	if !isRunnable && len(cmd.commands) > 0 {
		return po.err(fmt.Errorf("no command specified"))
	}
	po.Runner = runner

	return po
}

// parseEnvVars sets any unset field values using the environment variable
// matching the "env" tag of the field, if present.
func (cmd *Command) parseEnvVars() error {
	for _, f := range cmd.fields {
		if f.EnvVarName == "" || f.flagValue.setCount > 0 {
			continue
		}
		if s, ok := os.LookupEnv(f.EnvVarName); ok {
			if err := f.flagValue.Set(s); err != nil {
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

type ParsedCommand struct {
	Err     error
	Command *Command
	Runner  Runner
}

// Convenience method for returning errors wrapped as ParsedCommand.
func (po ParsedCommand) err(err error) ParsedCommand {
	po.Err = err
	return po
}

// Run calls the Run method of the Command config for the parsed command or, if
// an error occurred during parsing, prints the help text and returns that
// error instead. If help was requested, the error will flag.ErrHelp.
func (po ParsedCommand) Run() error {
	if po.Err != nil {
		po.Command.WriteHelp(errWriter)
		return po.Err
	}
	if po.Runner == nil {
		return fmt.Errorf("no run method implemented")
	}
	return po.Runner.Run()
}

// RunFatal is like Run, except it automatically handles printing out any
// errors returned by the Run method of the underlying Command config, and
// exits with an appropriate status (1 if error, 0 otherwise).
func (po ParsedCommand) RunFatal() {
	err := po.Run()
	if err != nil {
		if err != ErrHelp {
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
