package opts

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

type Opts struct {
	Name           string
	parent         *Opts
	config         interface{}
	internalConfig internalConfig
	fields         []field
	flagset        *flag.FlagSet
	commands       map[string]*Opts
	// run     func() error
}

type internalConfig struct {
	Help bool `opts:"short=h" help:"show usage help"`
}

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

	// for _, f := range fields {
	// 	fmt.Printf("opts: %s: field: %+v flagValue.String()=%v\n", name, f, f.flagValue.String())
	// }
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

func (opts *Opts) AddCommand(cmdOpts *Opts) *Opts {
	cmdOpts.parent = opts
	opts.commands[cmdOpts.Name] = cmdOpts
	return opts
}

// func (opts *Opts) AddCommand(name string,  config interface{}) *Opts {
// 	cmdOpts := New(name, config)
// 	cmdOpts.parent = opts
// 	opts.commands[name] = cmdOpts
// 	return opts
// }

// func (opts *Opts) AddCommands(commands map[string]interface{}) *Opts {
// 	for name, config := range commands {
// 		opts.AddCommand(name, config)
// 	}
// 	return opts
// }

func (opts *Opts) Parse() ParsedOpts {
	return opts.ParseArgs(os.Args)
}

func (opts *Opts) ParseArgs(args []string) ParsedOpts {
	po := ParsedOpts{Opts: opts}

	// if we're the root, the first arg is the program name
	if opts.parent == nil {
		// prog := ""
		if len(args) > 0 {
			// prog = args[0]
			args = args[1:]
		}
	}

	err := opts.flagset.Parse(args)
	if err != nil {
		return po.err(err)
	}

	if opts.internalConfig.Help {
		return po.err(flag.ErrHelp)
	}

	for _, f := range opts.fields {
		if f.EnvVarName != "" && f.flagValue.setCount < 1 {
			if s, ok := os.LookupEnv(f.EnvVarName); ok {
				f.flagValue.Set(s)
			}
		}
	}

	for _, f := range opts.fields {
		if f.Required && f.flagValue.setCount < 1 {
			return po.err(fmt.Errorf("required flag %s not set", f.Name))
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

	return po
}

type ParsedOpts struct {
	Err error
	// run        func() error
	Opts *Opts
}

func (po ParsedOpts) err(err error) ParsedOpts {
	po.Err = err
	return po
}

func (po ParsedOpts) Run() error {
	if po.Err != nil {
		os.Stderr.WriteString(po.Opts.Help())
		return po.Err
	}
	run := func() error { return fmt.Errorf("run not implemented") }
	if runner, ok := po.Opts.config.(Runner); ok {
		run = runner.Run
	}
	return run()
}

func (po ParsedOpts) RunFatal() {
	err := po.Run()
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

type Runner interface {
	Run() error
}
