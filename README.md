Workgroup
=========
A module for managing the execution of a group of goroutines.
* A workgroup always waits for all goroutines to complete (prevents leaking goroutines)
* Buildin support for common execution patterns:
  * CancelOnFirstError (similar to [Promise.all](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise/all))
  * CancelOnFirstSuccess (similar to [Promise.any](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise/any))
  * CancelOnFirstDone (similar to [Promise.race](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise/race))
* Easily limit concurrency if needed
* Extensible architecture allows behavior to be customized

## Quick Start

```sh
go get -u github.com/dxmaxwell/workgroup
```

## Basic Usage

### Simple Pipeline (without Error Handling)

```go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	wg "github.com/dxmaxwell/workgroup"
)

func main() {
	// Find the size of all files in the currect directory
	var total = 0
	var names = make(chan string)
	var sizes = make(chan int)

	wg.Work(nil, nil, nil,
		func(ctx context.Context) error {
			defer close(names)
			files, _ := filepath.Glob("*")
			for _, f := range files {
				names <- f
			}
			return nil
		},
		func(ctx context.Context) error {
			defer close(sizes)
			return wg.WorkFor(ctx, nil, nil, 5,
				func(ctx context.Context, i int) error {
					for name := range names {
						info, _ := os.Stat(name)
						if !info.IsDir() {
							sizes <- int(info.Size())
						}
					}
					return nil
				},
			)
		},
		func(ctx context.Context) error {
			for s := range sizes {
				total += s
			}
			return nil
		},
	)

	fmt.Printf("Total: %d\n", total)
}

```


## Similar Modules
* [go-promise](https://pkg.go.dev/github.com/fanliao/go-promise)
* [errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)
* [tomb](https://pkg.go.dev/gopkg.in/tomb.v2)