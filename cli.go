package cli

// New creates a new Command with the provided name and config. The config must be
// a pointer to a configuration struct. Default values can be specified by
// simply setting them on the config struct.
//
// New returns an Command pointer for further method chaining. If an error is
// encountered while building the options, such as a struct field having an
// unsupported type, New will panic. If you would like to have errors returned
// for handling, use Build instead.
func New(name string, config interface{}) *Command {
	return DefaultContext.New(name, config)
}

// Build is like New, but it returns any errors instead of calling panic, at
// the expense of being harder to chain.
func Build(name string, config interface{}) (*Command, error) {
	return DefaultContext.Build(name, config)
}

type Runner interface {
	Run() error
}

type Beforer interface {
	Before() error
}

type Setuper interface {
	SetupCommand(cmd *Command)
}
