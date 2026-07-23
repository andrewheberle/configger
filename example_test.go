package configger_test

import (
	"fmt"
	"os"

	"github.com/andrewheberle/configger"
	"github.com/spf13/pflag"
)

func ExampleLoadConfig_withoutConfig() {
	// set up a flagset and parse flags
	f := pflag.NewFlagSet("example", pflag.ContinueOnError)
	f.String("config", "", "path to configuration file")
	f.String("foo", "default-foo", "foo flag")
	f.String("bar", "default-bar", "bar flag")
	f.StringSlice("baz", []string{"default-baz"}, "baz flag")
	if err := f.Parse([]string{"--foo", "flag-foo"}); err != nil {
		panic(err)
	}

	config, err := configger.LoadConfig(f)
	if err != nil {
		panic(err)
	}

	baz := config.Strings("baz")
	fmt.Printf("foo: %s; bar: %s; baz: %s; len(baz): %d\n", config.String("foo"), config.String("bar"), baz, len(baz))
	// Output:
	// foo: flag-foo; bar: default-bar; baz: [default-baz]; len(baz): 1
}

func ExampleLoadConfig_withConfig() {
	// set up a flagset and parse flags
	f := pflag.NewFlagSet("example", pflag.ContinueOnError)
	f.String("config", "", "path to configuration file")
	f.String("foo", "default-foo", "foo flag")
	f.String("bar", "default-bar", "bar flag")
	f.StringSlice("baz", []string{"default-baz"}, "baz flag")
	if err := f.Parse([]string{"--config", "testdata/config.yml"}); err != nil {
		panic(err)
	}

	config, err := configger.LoadConfig(f)
	if err != nil {
		panic(err)
	}

	baz := config.Strings("baz")
	fmt.Printf("foo: %s; bar: %s; baz: %s; len(baz): %d\n", config.String("foo"), config.String("bar"), baz, len(baz))
	// Output:
	// foo: file-foo; bar: file-bar; baz: [default-baz]; len(baz): 1
}

func ExampleWithEnvPrefix() {
	// set up a flagset and parse flags
	f := pflag.NewFlagSet("example", pflag.ContinueOnError)
	f.String("config", "", "path to configuration file")
	f.String("foo", "default-foo", "foo flag")
	f.String("bar", "default-bar", "bar flag")
	f.StringSlice("baz", []string{"default-baz"}, "baz flag")
	if err := f.Parse([]string{"--baz", "flag-baz-a,flag-baz-b"}); err != nil {
		panic(err)
	}

	// set an env var
	if err := os.Setenv("TEST_FOO", "env-foo"); err != nil {
		panic(err)
	}
	defer func() {
		_ = os.Unsetenv("TEST_FOO")
	}()

	// load config
	config, err := configger.LoadConfig(f, configger.WithEnvPrefix("test"))
	if err != nil {
		panic(err)
	}

	baz := config.Strings("baz")
	fmt.Printf("foo: %s; bar: %s; baz: %s; len(baz): %d\n", config.String("foo"), config.String("bar"), baz, len(baz))
	// Output:
	// foo: env-foo; bar: default-bar; baz: [flag-baz-a flag-baz-b]; len(baz): 2
}
