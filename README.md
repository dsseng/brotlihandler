Brotli Handler
============

This is a tiny Go package which wraps HTTP handlers to transparently compress the
response body using Brotli, for clients which support it. Although it's usually simpler to
leave that to a reverse proxy (like nginx or Varnish), this package is useful
when that's undesirable.

This is the clone of [GzipHandler middleware](https://github.com/nytimes/gziphandler/) with Gzip replaced with Brotli. The most of the code was copied from the original repository.

## Install
```bash
go get -u github.com/sh7dm/brotlihandler
```

## Usage

Call `BrotliHandler` with any handler (an object which implements the
`http.Handler` interface), and it'll return a new handler which compresses the
response with Brotli. For example:

```go
package main

import (
	"io"
	"net/http"
	"github.com/sh7dm/brotlihandler"
)

func main() {
	withoutBr := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "Hello, World")
	})

	withBr := brotlihandler.BrotliHandler(withoutBr)

	http.Handle("/", withBr)
	http.ListenAndServe("0.0.0.0:8000", nil)
}
```


## Documentation

The docs can be found at [godoc.org][docs], as usual.


## License

[Apache 2.0][license].

[docs]:     https://godoc.org/github.com/sh7dm/brotlihandler
[license]:  https://github.com/sh7dm/brotlihandler/blob/master/LICENSE
