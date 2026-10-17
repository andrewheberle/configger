# configger

[![codecov](https://codecov.io/gh/andrewheberle/configger/graph/badge.svg?token=2Dmqn2LEKR)](https://codecov.io/gh/andrewheberle/configger)
[![GoDoc](https://pkg.go.dev/badge/github.com/andrewheberle/configger?utm_source=godoc)](https://pkg.go.dev/github.com/andrewheberle/configger)

This Go package handles loading of configuration from the following sources:

* Flags
* A configuration file
* Environment variables

Underneath this uses [github.com/knadh/koanf](github.com/knadh/koanf) to handle
this process, however this package is opinionated as follows:

* Command line flags are via `github.com/spf13/pflag`
* Configuration file support is YAML
* The order of loading keys is as follows with values from higher levels
  overriding lower ones:
  1. Defaults from command line flags
  2. Configuration file
  3. Environment variables
  4. Command line flags
* Environment variables (when an env prefix is set via `WithEnvPrefix`) can
  only set values that have a command line flag. For example with a prefix of
  `APP` the flag `log-level` is set by `APP_LOG_LEVEL`. Values are parsed by
  the flag itself, and flags set this way are updated in the `pflag.FlagSet`.
* The returned `configger.Config` does not provide access to the underlying
  `*koanf.Koanf` type and does not implement all of its functionality.

If you require more flexibility than provided above it is recommended to use
[github.com/knadh/koanf](github.com/knadh/koanf) directly.

## Usage

Basic usage is shown below:

```go
package main

import (
	"fmt"
	"os"

	"github.com/andrewheberle/configger"
	"github.com/spf13/pflag"
)

func main() {
	// set up a flagset and parse flags
	f := pflag.NewFlagSet("my-command", pflag.ContinueOnError)
	f.String("config", "", "path to configuration file")
	f.String("foo", "default-foo", "foo flag")
	f.String("bar", "default-bar", "bar flag")
	if err := f.Parse(os.Args[1:]); err != nil {
		panic(err)
	}

	config, err := configger.LoadConfig(f)
	if err != nil {
		panic(err)
	}

	fmt.Printf("foo: %s; bar: %s\n", config.String("foo"), config.String("bar"))
}
```
