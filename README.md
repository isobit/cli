# opts

## Example

```go
package main

import (
	"github.com/isobit/opts"
)

type App struct {
	Excited bool
	Greeting string
	Name string `opts:"required,short=n"`
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
	opts.New("greet", &app{Greeting: "Hello"}).
		Parse().
		RunFatal()
}
```

```console
$ greet -n world --excited
Hello, world!
```
