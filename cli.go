package cli

import (
	"io"
	"os"
)

// CLI defines functionality which is global to all commands which it
// constructs. The top-level New and Build methods use a CLI with good defaults
// for most cases, but custom CLI structs can be used to modify behavior.
type CLI struct {
	// HelpWriter is used to print help output when calling ParseResult.Run
	// (and other similar methods).
	HelpWriter io.Writer

	// ErrWriter is used to print errors when calling ParseResult.Run (and
	// other similar methods).
	ErrWriter io.Writer

	// LookupEnv is called during parsing for any fields which define an env
	// var key, but are not set by argument.
	LookupEnv LookupEnvFunc

	// Setter can be used to define custom setters for arbitrary field types,
	// or to override the default field setters.
	//
	// Here is an example which uses a custom layout for parsing any time.Time
	// fields:
	//
	//  type customTime time.Time
	//  func (t *customTime) Set(s string) error {
	//  	parsed, err := time.Parse("2006-01-02 15:04", s)
	//  	if err != nil {
	//  		return err
	//  	}
	//  	*ts.value = parsed
	//  	return nil
	//  }
	//  cli := cli.NewCLI()
	//  cli.Setter = func(i interface{}) cli.Setter {
	//  	switch v := i.(type) {
	//  	case *time.Time:
	//  		return (*customTime)(v)
	//  	default:
	//  		// return nil to fall back on default behavior
	//  		return nil
	//  	}
	//  }
	Setter SetterFunc
}

func NewCLI() *CLI {
	return &CLI{
		HelpWriter: os.Stderr,
		ErrWriter:  os.Stderr,
		LookupEnv:  osLookupEnv,
		Setter:     nil,
	}
}

var defaultCLI *CLI = NewCLI()

// osLookupEnv wraps os.LookupEnv as a LookupEnvFunc
func osLookupEnv(key string) (string, bool, error) {
	val, ok := os.LookupEnv(key)
	return val, ok, nil
}

// New creates a new Command with the provided name and config. The config must be
// a pointer to a configuration struct. Default values can be specified by
// simply setting them on the config struct.
//
// Command options (e.g. help text and subcommands) can be passed as additonal
// CommandOption arguments, or set using chained method calls. Note that
// *Command implements CommandOption, so subcommands can be registered by
// simply passing them as arugments.
//
// New returns an Command pointer for further method chaining. If an error is
// encountered while building the options, such as a struct field having an
// unsupported type, New will panic. If you would like to have errors returned
// for handling, use Build instead.
func New(name string, config interface{}, opts ...CommandOption) *Command {
	return defaultCLI.New(name, config, opts...)
}

// Build is like New, but it returns any errors instead of calling panic, at
// the expense of being harder to chain.
func Build(name string, config interface{}, opts ...CommandOption) (*Command, error) {
	return defaultCLI.Build(name, config, opts...)
}

type LookupEnvFunc func(key string) (val string, ok bool, err error)

type SetterFunc func(interface{}) Setter
