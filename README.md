# opts

## Example

```go
package main

import (
	"github.com/isobit/opts"
)

type App struct {
	Excited bool
	Name string `opts:"required,short=n"`
}

func (app *App) Run() error {
	punctuation := "."
	if app.Excited {
		punctuation = "!"
	}
	fmt.Printf("Hello, %s%s\n", app.Name, punctuation)
	return nil
}


func main() {
	opts.New("greet", app).
		Parse().
		RunFatal()
}
```

```console
$ greet -n world --excited
Hello, world!
```
