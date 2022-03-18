# opts

[![Go Reference](https://pkg.go.dev/badge/github.com/isobit/opts.svg)](https://pkg.go.dev/github.com/isobit/opts)

## Example

```go
package main

import (
	"fmt"

	"github.com/isobit/opts"
)

type App struct {
	Excited  bool   `opts:"help='when true, use exclamation point'"`
	Greeting string `opts:"env=GREETING,help=the greeting to use"`
	Name     string `opts:"required,short=n,help=your name"`
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
	opts.New("greet", &App{Greeting: "Hey"}).
		Parse().
		RunFatal()
}
```

```console
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
```
