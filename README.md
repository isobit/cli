# opts

## Example

```go
package main

import (
	"fmt"

	"github.com/isobit/opts"
)

type App struct {
	Excited bool `opts:"help='when true, use exclamation point'"`
	Greeting string `opts:"help=the greeting to use"`
	Name string `opts:"required,short=n,help=your name"`
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
	opts.New("greet", &App{Greeting: "Hello"}).
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
    --excited           when true, use exclamation point
    --greeting <VALUE>  the greeting to use  (default: Hello)
    -n, --name <VALUE>  your name

error: flag: help requested
$ greet -n world --excited
Hello, world!
```
