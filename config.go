package configger

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

const (
	DefaultConfigKeyName = "config"
)

// Config represents a configuration source.
type Config interface {
	Bool(key string) bool
	Bools(key string) []bool
	BoolMap(key string) map[string]bool
	Duration(key string) time.Duration
	Float64(key string) float64
	Float64s(key string) []float64
	Float64Map(key string) map[string]float64
	Int(key string) int
	Ints(key string) []int
	IntMap(key string) map[string]int
	Int64(key string) int64
	Int64s(key string) []int64
	Int64Map(key string) map[string]int64
	String(key string) string
	Strings(key string) []string
	StringMap(key string) map[string]string
}

// Returns a [Config] by reading a YAML based configuration file,
// environment variables and command line flags.
//
// The configuration file is loaded based on the [WithConfigKeyName]
// or the default of [DefaultConfigKeyName] being set in the provided
// [pflag.FlagSet]. Loading of a configuration file can be disabled via
// [WithoutConfigurationFile], via [WithConfigKeyName] as a blank string
// or loading will be skipped if the value from the config key name is blank
// or unset. Loading is also skipped if the [pflag.FlagSet] has no flag for
// [DefaultConfigKeyName], however a key given via [WithConfigKeyName] must
// have a flag or an error is returned.
//
// If [WithEnvPrefix] is provided then environment variables can set any flag
// not given on the command line, including the flag naming the configuration
// file. Each flag is set from the variable named "PREFIX_" followed by the
// flag name upper-cased with "-" and "." replaced by "_", so "log-level" is
// set by "PREFIX_LOG_LEVEL". Values are parsed by the flag itself, so slice
// flags take comma separated values (quote an item to keep a comma in it).
// Environment variables only ever set keys that have a flag.
//
// Flags set from environment variables are updated in the provided
// [pflag.FlagSet], so they report as changed and hold the value in effect.
// An environment variable value the flag rejects, or two flags sharing an
// environment variable name, results in an error.
func LoadConfig(f *pflag.FlagSet, opts ...Option) (Config, error) {
	config := &options{
		key: DefaultConfigKeyName,
	}

	for _, o := range opts {
		o(config)
	}

	// Apply env vars to flags (unless env prefix is blank). This happens
	// first so an env var can select the config file, while flags set from
	// env still override the config file as they are marked as changed.
	if config.envprefix != "" {
		if err := applyEnv(f, config.envprefix); err != nil {
			return nil, fmt.Errorf("error reading env vars: %w", err)
		}
	}

	k := koanf.New(".")

	// load any config file (unless key is set to blank or disabled, or the
	// default key was not explicitly requested and has no flag)
	if config.key != "" && !config.disabled && (config.keyset || f.Lookup(config.key) != nil) {
		if config, err := f.GetString(config.key); err != nil {
			return nil, fmt.Errorf("error getting flag value: %w", err)
		} else if config != "" {
			if err := k.Load(file.Provider(config), yaml.Parser()); err != nil {
				return nil, fmt.Errorf("error loading configuration: %w", err)
			}
		}
	}

	// Load command line options
	if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
		return nil, fmt.Errorf("error reading command line: %w", err)
	}

	return k, nil
}

// applyEnv sets each flag not given on the command line from its env name,
// so the flag parses the value according to its own type.
func applyEnv(f *pflag.FlagSet, prefix string) error {
	var errs []error

	// check no two flags share an env name before setting anything
	names := make(map[string][]string)
	f.VisitAll(func(flag *pflag.Flag) {
		name := envName(prefix, flag.Name)
		names[name] = append(names[name], flag.Name)
	})
	for name, flags := range names {
		if len(flags) > 1 {
			errs = append(errs, fmt.Errorf("flags %s share env name %s", strings.Join(flags, ", "), name))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	f.VisitAll(func(flag *pflag.Flag) {
		if flag.Changed {
			return
		}

		name := envName(prefix, flag.Name)
		v, ok := os.LookupEnv(name)
		if !ok {
			return
		}

		if err := f.Set(flag.Name, v); err != nil {
			errs = append(errs, fmt.Errorf("%s=%q: %w", name, v, err))
		}
	})

	return errors.Join(errs...)
}

// envName returns the environment variable name for a flag, eg "log-level"
// with a prefix of "APP" is "APP_LOG_LEVEL".
func envName(prefix, flag string) string {
	return prefix + "_" + strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(flag))
}

type options struct {
	envprefix string
	key       string
	keyset    bool
	disabled  bool
}

// Option is used to modify the behaviour of [LoadConfig].
type Option func(*options)

// Sets a specific key name to lookup the configuration file name from. Unlike
// the default of [DefaultConfigKeyName], a key set this way must have a flag
// in the [pflag.FlagSet] passed to [LoadConfig].
func WithConfigKeyName(key string) Option {
	return func(c *options) {
		c.key = key
		c.keyset = true
	}
}

// Explicitly disable configuration file loading.
func WithoutConfigurationFile() Option {
	return func(c *options) {
		c.disabled = true
	}
}

// Set the prefix for environment variable loading. Environment variables
// only set values for flags in the [pflag.FlagSet] passed to [LoadConfig].
func WithEnvPrefix(prefix string) Option {
	return func(c *options) {
		// trim any undescore as we add it back later
		c.envprefix = strings.ToUpper(strings.TrimSuffix(prefix, "_"))
	}
}
