package cli

import (
	"fmt"
	"io"
	"os"
)

type LookupEnvFunc func(string) (string, bool, error)
type SetterFunc func(interface{}) Setter

type Context struct {
	ErrWriter io.Writer
	LookupEnv LookupEnvFunc
	Setter    SetterFunc
}

var DefaultContext = Context{
	ErrWriter: os.Stderr,
	LookupEnv: func(key string) (string, bool, error) {
		val, ok := os.LookupEnv(key)
		return val, ok, nil
	},
	Setter: nil,
}

// func Build2(name string, config interface{}) (*Command, error) {
// 	return DefaultContext.Build(name, config)
// }

// New creates a new Command with the provided name and config. The config must be
// a pointer to a configuration struct. Default values can be specified by
// simply setting them on the config struct.
//
// New returns an Command pointer for further method chaining. If an error is
// encountered while building the options, such as a struct field having an
// unsupported type, New will panic. If you would like to have errors returned
// for handling, use Build instead.
func (ctx Context) New(name string, config interface{}) *Command {
	cmd, err := newCommand(ctx, name, config)
	if err != nil {
		panic(fmt.Sprintf("cli: %s", err))
	}
	return cmd
}

// Build is like New, but it returns any errors instead of calling panic, at
// the expense of being harder to chain.
func (ctx Context) Build(name string, config interface{}) (*Command, error) {
	return newCommand(ctx, name, config)
}
