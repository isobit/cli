/*
Package opts makes it easy to create CLIs by defining options using struct tags.

Example

Greet program:

		package main

		import (
			"fmt"

			"github.com/isobit/opts"
		)

		type Greet struct {
			Excited  bool   `opts:"help='when true, use exclamation point'"`
			Greeting string `opts:"env=GREETING,help=the greeting to use"`
			Name     string `opts:"required,short=n,help=your name"`
		}

		func (g *Greet) Run() error {
			punctuation := "."
			if g.Excited {
				punctuation = "!"
			}
			fmt.Printf("%s, %s%s\n", g.Greeting, g.Name, punctuation)
			return nil
		}

		func main() {
			opts.New("greet", &Greet{Greeting: "Hey"}).
				Parse().
				RunFatal()
		}

Usage:

		$ greet --help
		USAGE:
			greet [OPTIONS]

		OPTIONS:
			-h, --help          show usage help
			--excited           use exclamation point
			--greeting <VALUE>  the greeting to use  (default: Hey)
			-n, --name <VALUE>  your name

		error: flag: help requested
		$ GREETING="Hello" greet -n world --excited
		Hello, world!


Struct Tags

The parsing behavior for config fields can be controlled by adding a struct tag
that opts understands. Opts struct tags look like `opts:"key1,key2=value"`. For
example:

		struct MyOpts {
			F1 string `opts:"-"`                               // opts skips any fields with "-"
			F2 string `opts:"required"`                        // error if the field is not set at least once
			F3 string `opts:"help=the value for F3"`           // custom help text
			F4 string `opts:"help='to help, or not to help?'"` // custom help text with a comma
			F5 string `opts:"placeholder=<D>"`                 // custom help placeholder, .e.g "--d <D>"
			F6 string `opts:"name=eee"`                        // explicitly set the flag name
			F7 string `opts:"short=f"`                         // add a short alias name (must be 1 rune)
			F8 string `opts:"env=MY_F8_VALUE"`                 // use an environment variable as a default value
			F9 string `opts:"short=F,required,help=some help"` // combining multiple tags
		}

Field Types and Flag Parsing

Primitive types (e.g. int and string), and pointers to primitive types (e.g.
*int and *string) are handled natively by opts. In the case of pointers, if the
default value is a nil pointer, and a value is passed, opts will construct a
new value of the inner type and set the struct field to be a pointer to the
newly constructed value.

There is no special parsing for string fields, they are set directly from input.

The following primitives are parsed by fmt.Sscanf using the "%v" directive:

		bool,
		int, int8, int16, int32, int64,
		uint8, uint16, uint32, uint64,
		float32, float64

Additionally, time.Duration fields are automatically parsed using time.ParseDuration.

All other types are parsed using the first method below that is implemented
with the type itself or a pointer to the type as the receiver:

		Set(s string) error                 // similar to flag.Value
		UnmarshalText(text []byte) error    // encoding.TextUnmarshaler
		UnmarshalBinary(data []byte) error  // encoding.BinaryUnmarshaler

Many standard library types already implement one of these methods. For
example, time.Time implements encoding.TextUnmarshaler for parsing RFC 3339
timestamps.

Custom types can be used so long as they implement one of the above methods.
Here is an example which parses a string slice from a comma-delimited flag
value string:

		type App struct {
			Foos Foos
		}

		type Foos []string

		func (foos *Foos) UnmarshalText(text []byte) error {
			s := string(text)
			*foos = append(*foos, strings.Split(s, ",")...)
			return nil
		}
*/
package opts
