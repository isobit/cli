package cli

import (
	"flag"
	"fmt"
)

type boolFlag interface {
	flag.Value
	IsBoolFlag() bool
}

type parser struct {
	fields map[string]field
	parsed bool
	args   []string
}

func (p *parser) parse(arguments []string) error {
	p.parsed = true
	p.args = arguments
	for {
		seen, err := p.parseOne()
		if err != nil {
			return err
		}
		if !seen {
			break
		}
	}
	return nil
}

// copied and modified from the flag package
func (p *parser) parseOne() (bool, error) {
	if len(p.args) == 0 {
		return false, nil
	}
	s := p.args[0]
	if len(s) < 2 || s[0] != '-' {
		return false, nil
	}
	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			p.args = p.args[1:]
			return false, nil
		}
	}
	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return false, fmt.Errorf("bad flag syntax: %s", s)
	}

	if numMinuses == 1 {
		i := 0
		for ; i < len(name)-1; i++ {
			shortName := name[i]
			if err := p.parseOneFlag(string(shortName), false, "", false); err != nil {
				return false, err
			}
		}
		name = name[i:]
	}

	// it's a flag. does it have an argument?
	p.args = p.args[1:]
	hasValue := false
	value := ""
	for i := 1; i < len(name); i++ { // equals cannot be first
		if name[i] == '=' {
			value = name[i+1:]
			hasValue = true
			name = name[0:i]
			break
		}
	}

	if err := p.parseOneFlag(name, hasValue, value, true); err != nil {
		return false, err
	}

	return true, nil
}

func (p *parser) parseOneFlag(name string, hasValue bool, value string, canLookNext bool) error {
	field, ok := p.fields[name]
	if !ok {
		return fmt.Errorf("flag provided but not defined: %s", name)
	}

	if fv, ok := field.flagValue.(boolFlag); ok && fv.IsBoolFlag() { // special case: doesn't need an arg
		if hasValue {
			if err := fv.Set(value); err != nil {
				return fmt.Errorf("invalid boolean value %q for flag %s: %v", value, name, err)
			}
		} else {
			if err := fv.Set("true"); err != nil {
				return fmt.Errorf("invalid boolean flag %s: %v", name, err)
			}
		}
	} else {
		// It must have a value, which might be the next argument.
		if !hasValue && len(p.args) > 0 && canLookNext {
			// value is the next arg
			hasValue = true
			value, p.args = p.args[0], p.args[1:]
		}
		if !hasValue {
			return fmt.Errorf("flag needs an argument: %s", name)
		}
		if err := flag.Value.Set(value); err != nil {
			return fmt.Errorf("invalid value %q for flag %s: %v", value, name, err)
		}
	}
	return nil
}

func (cmd *Command) ParseArgsGNU(args []string) ParseResult {
	if args == nil {
		args = []string{}
	}

	r := ParseResult{Command: cmd}

	// Help command
	if cmd.parent == nil && len(args) > 0 && args[0] == "help" {
		curCmd := cmd
		for i := 1; i < len(args); i++ {
			cmdName := args[i]
			if subCmd, ok := curCmd.commandMap[cmdName]; ok {
				curCmd = subCmd
			} else {
				return r.err(UsageErrorf("unknown command: %s", cmdName))
			}
		}
		return ParseResult{Command: curCmd, Err: ErrHelp}
	}

	p := parser{Fields: cmd.fields, args: args}

	// Parse arguments using the flagset.
	if err := p.parse(args); err != nil {
		return r.err(UsageErrorf("failed to parse args: %w", err))
	}

	// Return ErrHelp if help was requested.
	if cmd.internalConfig.Help {
		return r.err(ErrHelp)
	}

	// Parse environment variables.
	if err := cmd.parseEnvVars(); err != nil {
		return r.err(UsageErrorf("failed to parse environment variables: %w", err))
	}

	// Handle remaining arguments so we get unknown command errors before
	// invoking Before.
	var subCmd *Command
	rargs := p.args
	if len(rargs) > 0 {
		switch {
		case cmd.argsField != nil:
			cmd.argsField.setter(rargs)

		case len(cmd.commandMap) > 0:
			cmdName := rargs[0]
			if cmd, ok := cmd.commandMap[cmdName]; ok {
				subCmd = cmd
			} else {
				return r.err(UsageErrorf("unknown command: %s", cmdName))
			}

		default:
			return r.err(UsageErrorf("command does not take arguments"))
		}
	}

	// Return an error if any required fields were not set at least once.
	if err := cmd.checkRequired(); err != nil {
		return r.err(UsageError(err))
	}

	// If the config implements a Before method, run it before we recursively
	// parse subcommands.
	if beforer, ok := cmd.config.(Beforer); ok {
		if err := beforer.Before(); err != nil {
			return r.err(err)
		}
	}

	// Recursive to subcommand parsing, if applicable.
	if subCmd != nil {
		return subCmd.ParseArgsGNU(rargs[1:])
	}

	r.runFunc = getRunFunc(cmd.config)
	if r.runFunc == nil && len(cmd.commands) != 0 {
		return r.err(UsageErrorf("no command specified"))
	}

	return r
}
