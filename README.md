# gemini

A small go module to serve Gemini servers, create requests, etc.

## Installation

`go get github.com/CPunch/gemini`

## Example

```go
package main

import (
	"github.com/CPunch/gemini"
)

func main() {
	response, _ := gemini.LazyRequest("gemini://gemini.circumlunar.space/docs/specification.gmi")

	println(response)
}
```
> More examples (including servers!) can be found in the `/examples` directory