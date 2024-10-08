# cli

[![Go Reference](https://pkg.go.dev/badge/github.com/isobit/cli.svg)](https://pkg.go.dev/github.com/isobit/cli)

Package `cli` makes it easy to create CLIs by defining options using struct
tags.

:warning: This package is under active development and is subject to major
breaking changes prior to the 1.0 release.

## Example

```go
package main

import (
	"fmt"

	"github.com/isobit/cli"
)

type App struct {
	Excited  bool   `cli:"help='when true, use exclamation point'"`
	Greeting string `cli:"env=GREETING,help=the greeting to use"`
	Name     string `cli:"required,short=n,help=your name"`
}

func (app *App) Run() error {
	punctuation := "."
	if app.Excited {
		punctuation = "!"
	}
	fmt.Printf("%s, %s%s\n", app.Greeting, app.Name, punctuation)
	return nil
}

func main() {
	cli.New("greet", &App{
		Greeting: "Hey",
	}).
		Parse().
		RunFatal()
}
```

```console
$ greet --help
USAGE:
    greet [OPTIONS]

OPTIONS:
    -h, --help                    show usage help
    --excited                     when true, use exclamation point
    --greeting <VALUE>  GREETING  the greeting to use  (default: Hey)
    -n, --name <VALUE>            your name  (required)

$ GREETING="Hello" greet -n world --excited
Hello, world!
```

## Struct Tags

The parsing behavior for config fields can be controlled by adding a struct tag
that `cli` understands. Command struct tags look like
`cli:"key1,key2=value,key3='blah'"`; for example:

```go
struct Example {
	Foo string `cli:"required,placeholder=quux,short=f,env=FOO,help='hello, world'"`
}
```

| Tag           | Value | Description                                                                                          |
| -             | -     | -                                                                                                    |
| `-`           | No    | Ignore field (similar to `encoding/json`)                                                            |
| `required`    | No    | Error if the field is not set at least once                                                          |
| `help`        | Yes   | Custom help text                                                                                     |
| `placeholder` | Yes   | Custom value placeholder in help text                                                                |
| `name`        | Yes   | Explicit flag name (by default names are derived from the struct field name)                         |
| `short`       | Yes   | Single character short name alias                                                                    |
| `env`         | Yes   | Environment variable to use as a default value                                                       |
| `default`     | Yes   | Custom default string in help text (does not affect actual default value)                            |
| `nodefault`   | No    | Don't show default value in help text                                                                |
| `hidden`      | No    | Don't show field in help text                                                                        |
| `append`      | No    | Change flag setting behavior to append to value when specified multiple times (must be a slice type) |
| `args`        | No    | Set this field to the remaining non-flag args instead of recursively parsing them as subcommands.    |

Tags are parsed according to this ABNF:

	tags = "cli:" DQUOTE *(tag ",") tag DQUOTE
	tag = key [ "=" value ]
	key = *<anything except "=">
	value = *<anything except ","> / "'" *<anything except "'"> "'"

## Field Types and Value Parsing

Primitive types (e.g. `int` and `string`), and pointers to primitive types
(e.g. `*int` and `*string`) are handled natively by `cli`. In the case of
pointers, if the default value is a nil pointer, and a value is passed, `cli`
will construct a new value of the inner type and set the struct field to be a
pointer to the newly constructed value.

There is no special parsing for string fields, they are set directly from input.

The following primitives are parsed by `fmt.Sscanf` using the `%v` directive:

- `bool`
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint8`, `uint16`, `uint32`, `uint64`
- `float32`, `float64`

Additionally, `time.Duration` fields are automatically parsed using
`time.ParseDuration`.

All other types are parsed using the first method below that is implemented
with the type itself or a pointer to the type as the receiver:

- `Set(s string) error` (similar to `flag.Value`)
- `UnmarshalText(text []byte) error` (`encoding.TextUnmarshaler`)
- `UnmarshalBinary(data []byte) error` (`encoding.BinaryUnmarshaler`)

Many standard library types already implement one of these methods. For
example, time.Time implements `encoding.TextUnmarshaler` for parsing RFC 3339
timestamps.

Custom types can be used so long as they implement one of the above methods.
Here is an example which parses a string slice from a comma-delimited flag
value string:

```go
type App struct {
	Foos Foos
}

type Foos []string

func (foos *Foos) UnmarshalText(text []byte) error {
	s := string(text)
	*foos = append(*foos, strings.Split(s, ",")...)
	return nil
}
```

Default values for custom types are represented in help text using
`fmt.Sprintf("%v", value)`. This can be overridden by defining a `String()
string` method with the type itself or a pointer to the type as the receiver.

## Contexts and Signal Handling

Here is an example of a "sleep" program which sleeps for the specified
duration, but can be cancelled by a SIGINT or SIGTERM:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/isobit/cli"
)

type SleepCommand struct {
	Duration time.Duration `cli:"short=d,required"`
}

func (cmd *Sleep) Run(ctx context.Context) error {
	fmt.Printf("sleeping for %s\n", cmd.Duration)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(cmd.Duration):
		fmt.Println("done")
	}
	return nil
}

func main() {
	cli.New("sleep", &SleepCommand{}).
		Parse().
		RunFatalWithSigCancel()
}
```


## Command Line Syntax

Arguments are parsed using a more GNU-like modification of the algorithm used
in the standard [`flag` package](https://pkg.go.dev/flag); notably, single
dashes are treated differently from double dashes. The following forms are
permitted:

```
--flag    // long boolean flag (no argument)
-f        // single short flag (no argument)

--flag=x  // long flag with argument
--flag x  // long flag with argument
-f=x      // single short flag with argument
-f x      // single short flag with argument

-abc      // multiple short boolean flags (a, b, and c)

-abf x    // multiple short boolean flags (a, b) combined 
          // with single short flag with argument (f)

-abf=x    // multiple short boolean flags (a, b) combined 
          // with single short flag with argument (f)
```

Flag parsing for each command stops just before the first non-flag argument
(`-` is a non-flag argument) or after the terminator `--`. If the command has a
field with the `cli:"args"` tag, its value is set to a string slice containing
the remaining arguments. Otherwise, if the first non-flag argument is a
subcommand, the remaining arguments are further parsed by that subcommand,
recursively.
