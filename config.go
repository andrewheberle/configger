package configger

import (
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
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
// or loading will be skipped if the value from the confif key name is blank
// or unset.
//
// If [WithEnvPrefix] is provided then enviroment variables prefixed with
// "PREFIX_" will be included in the configuration.
func LoadConfig(f *pflag.FlagSet, opts ...Option) (Config, error) {
	config := &options{
		key: DefaultConfigKeyName,
	}

	for _, o := range opts {
		o(config)
	}

	k := koanf.New(".")

	// load any config file (unless key is set to blank or disabled)
	if config.key != "" && !config.disabled {
		if config, err := f.GetString(config.key); err != nil {
			return nil, fmt.Errorf("error getting flag value: %w", err)
		} else if config != "" {
			if err := k.Load(file.Provider(config), yaml.Parser()); err != nil {
				return nil, fmt.Errorf("error loading configuration: %w", err)
			}
		}
	}

	// Load env vars (unless env prefix is blank)
	if config.envprefix != "" {
		prefix := fmt.Sprintf("%s_", config.envprefix)
		if err := k.Load(env.Provider(".", env.Opt{
			Prefix: prefix,
			TransformFunc: func(k, v string) (string, any) {
				// Transform the key.
				k = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(k, prefix)), "_", ".")

				// Transform values with commas into slices
				if strings.Contains(v, ",") {
					return k, strings.Split(v, ",")
				}

				return k, v
			},
		}), nil); err != nil {
			return nil, fmt.Errorf("error reading env vars: %w", err)
		}
	}

	// Load command line options
	if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
		return nil, fmt.Errorf("error reading command line: %w", err)
	}

	return k, nil
}

type options struct {
	envprefix string
	key       string
	disabled  bool
}

// Option is used to modify the behaviour of [LoadConfig].
type Option func(*options)

// Sets a specific key name to lookup the configuration file name from.
func WithConfigKeyName(key string) Option {
	return func(c *options) {
		c.key = key
	}
}

// Explicitly disable configuration file loading.
func WithoutConfigurationFile() Option {
	return func(c *options) {
		c.disabled = true
	}
}

// Set the prefix for environment variable loading.
func WithEnvPrefix(prefix string) Option {
	return func(c *options) {
		// trim any undescore as we add it back later
		c.envprefix = strings.ToUpper(strings.TrimSuffix(prefix, "_"))
	}
}
