package cli

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

type Command struct {
	context         Context
	name            string
	help            string
	description     string
	config          interface{}
	internalConfig  internalConfig
	fields          []field
	flagset         *flag.FlagSet
	flagsetInternal *flag.FlagSet
	parent          *Command
	commands        map[string]*Command
}

type internalConfig struct {
	Help bool `cli:"short=h,help=show usage help"`
}

func newCommand(ctx Context, name string, config interface{}) (*Command, error) {
	cmd := &Command{
		context:  ctx,
		name:     name,
		config:   config,
		fields:   []field{},
		commands: map[string]*Command{},
	}

	internalFields, err := ctx.getFieldsFromConfig(&cmd.internalConfig)
	if err != nil {
		return nil, fmt.Errorf("error building internal config: %w", err)
	}
	cmd.fields = append(cmd.fields, internalFields...)
	cmd.flagsetInternal = newFlagSet(name, internalFields)

	configFields, err := ctx.getFieldsFromConfig(config)
	if err != nil {
		return nil, err
	}
	cmd.fields = append(cmd.fields, configFields...)
	cmd.flagset = newFlagSet(name, cmd.fields)

	if setuper, ok := cmd.config.(Setuper); ok {
		setuper.SetupCommand(cmd)
	}

	return cmd, nil
}

func (cmd *Command) Help(help string) *Command {
	cmd.help = help
	return cmd
}

func (cmd *Command) Description(description string) *Command {
	cmd.description = description
	return cmd
}

// AddCommand registers another Command instance as a subcommand of this Command
// instance.
func (cmd *Command) AddCommand(subCmd *Command) *Command {
	subCmd.parent = cmd
	cmd.commands[subCmd.name] = subCmd
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

// helpPass does a minimal recursive parsing pass using only the internal
// flagset, so that help flags can be detected on subcommands early.
func (cmd *Command) helpPass(args []string) ParsedCommand {
	po := ParsedCommand{Command: cmd}

	// Parse arguments using the flagset.
	// Intentionally ignore errors since we want to ignore any non-internal
	// flags.
	_ = cmd.flagsetInternal.Parse(args)

	// Return ErrHelp if help was requested.
	if cmd.internalConfig.Help {
		return po.err(ErrHelp)
	}

	// Handle remaining arguments, recursively parse subcommands.
	rargs := cmd.flagsetInternal.Args()
	if len(rargs) > 0 {
		cmdName := rargs[0]
		if cmd, ok := cmd.commands[cmdName]; ok {
			return cmd.helpPass(rargs[1:])
		}
	}

	return po
}

// parseEnvVars sets any unset field values using the environment variable
// matching the "env" tag of the field, if present.
func (cmd *Command) parseEnvVars() error {
	for _, f := range cmd.fields {
		if f.EnvVarName == "" || f.flagValue.setCount > 0 {
			continue
		}
		val, ok, err := cmd.context.LookupEnv(f.EnvVarName)
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
		po.Command.WriteHelp(po.Command.context.ErrWriter)
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
			fmt.Fprintf(po.Command.context.ErrWriter, "error: %s\n", err)
		}
		os.Exit(1)
	}
	os.Exit(0)
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
