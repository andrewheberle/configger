package configger_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/andrewheberle/configger"
	"github.com/spf13/pflag"
)

// newFlagSet builds the flag set shared by the table tests below and parses
// args against it, mimicking a real command line invocation.
func newFlagSet(t *testing.T, args []string) *pflag.FlagSet {
	t.Helper()

	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	f.String("config", "", "path to configuration file")
	f.String("otherconfig", "", "path to configuration file (custom key)")
	f.String("foo", "default-foo", "foo value")
	f.String("bar", "default-bar", "bar value")
	f.StringSlice("baz", []string{"default-baz"}, "baz value")
	f.String("log-level", "info", "log level")
	f.Int("port", 8080, "port number")
	f.String("db.host", "localhost", "database host")

	if err := f.Parse(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	return f
}

func TestLoadConfigAppliesEnvToFlagSet(t *testing.T) {
	t.Setenv("TEST_FOO", "env-foo")

	f := newFlagSet(t, []string{"--bar=flag-bar"})

	if _, err := configger.LoadConfig(f, configger.WithEnvPrefix("TEST")); err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	foo := f.Lookup("foo")
	if !foo.Changed || foo.Value.String() != "env-foo" {
		t.Errorf("flag foo = %q (changed %v), want %q (changed true)", foo.Value.String(), foo.Changed, "env-foo")
	}

	if got := f.Lookup("baz"); got.Changed {
		t.Errorf("flag baz changed with no env var set")
	}
}

func TestLoadConfigEnvNameClash(t *testing.T) {
	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	f.String("log-level", "info", "log level")
	f.String("log.level", "info", "log level (nested)")
	if err := f.Parse(nil); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	_, err := configger.LoadConfig(f, configger.WithEnvPrefix("TEST"), configger.WithoutConfigurationFile())
	if err == nil {
		t.Fatal("LoadConfig() succeeded unexpectedly")
	}

	for _, want := range []string{"TEST_LOG_LEVEL", "log-level", "log.level"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("LoadConfig() error %q does not mention %q", err, want)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		opts    []configger.Option
		want    map[string]any
		wantErr bool
	}{
		{
			name: "defaults only",
			want: map[string]any{"foo": "default-foo", "bar": "default-bar", "baz": []string{"default-baz"}},
		},
		{
			name: "config file overrides defaults",
			args: []string{"--config=testdata/config.yml"},
			want: map[string]any{"foo": "file-foo", "bar": "file-bar", "baz": []string{"default-baz"}},
		},
		{
			name: "explicitly blank config flag uses defaults",
			args: []string{"--config="},
			want: map[string]any{"foo": "default-foo", "bar": "default-bar", "baz": []string{"default-baz"}},
		},
		{
			name:    "nonexistent config file returns error",
			args:    []string{"--config=testdata/does-not-exist.yml"},
			wantErr: true,
		},
		{
			name: "custom config key name loads file",
			args: []string{"--otherconfig=testdata/config.yml"},
			opts: []configger.Option{configger.WithConfigKeyName("otherconfig")},
			want: map[string]any{"foo": "file-foo", "bar": "file-bar", "baz": []string{"default-baz"}},
		},
		{
			name: "env var overrides default when no config file",
			env:  map[string]string{"TEST_BAR": "env-bar"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"foo": "default-foo", "bar": "env-bar", "baz": []string{"default-baz"}},
		},
		{
			name: "env var without matching prefix option is ignored",
			env:  map[string]string{"TEST_FOO": "env-foo"},
			want: map[string]any{"foo": "default-foo"},
		},
		{
			name: "env var overrides config file value",
			args: []string{"--config=testdata/config.yml"},
			env:  map[string]string{"TEST_FOO": "env-foo"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"foo": "env-foo", "bar": "file-bar", "baz": []string{"default-baz"}},
		},
		{
			name: "flag overrides default when no config or env",
			args: []string{"--bar=flag-bar"},
			want: map[string]any{"foo": "default-foo", "bar": "flag-bar", "baz": []string{"default-baz"}},
		},
		{
			name: "flag overrides env var and config file",
			args: []string{"--config=testdata/config.yml", "--foo=flag-foo"},
			env:  map[string]string{"TEST_FOO": "env-foo", "TEST_BAR": "env-bar"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"foo": "flag-foo", "bar": "env-bar", "baz": []string{"default-baz"}},
		},
		{
			name: "full precedence chain resolved independently per key",
			args: []string{"--config=testdata/config.yml", "--bar=flag-bar"},
			env:  map[string]string{"TEST_FOO": "env-foo", "TEST_BAZ": "env-baz-a,env-baz-b"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"foo": "env-foo", "bar": "flag-bar", "baz": []string{"env-baz-a", "env-baz-b"}},
		},
		{
			name: "env var sets flag with dash in name",
			env:  map[string]string{"TEST_LOG_LEVEL": "debug"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"log-level": "debug"},
		},
		{
			name: "env var sets flag with dot in name",
			env:  map[string]string{"TEST_DB_HOST": "db.example.com"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"db.host": "db.example.com"},
		},
		{
			name: "env var with comma stays one value for string flag",
			env:  map[string]string{"TEST_FOO": "hello, world"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"foo": "hello, world"},
		},
		{
			name: "env var with single value sets slice flag",
			env:  map[string]string{"TEST_BAZ": "solo"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"baz": []string{"solo"}},
		},
		{
			name: "env var with quoted comma keeps it inside one slice item",
			env:  map[string]string{"TEST_BAZ": `"a,b",c`},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"baz": []string{"a,b", "c"}},
		},
		{
			name: "env var that is set but empty sets flag to empty",
			env:  map[string]string{"TEST_FOO": ""},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"foo": ""},
		},
		{
			name: "env var sets int flag",
			env:  map[string]string{"TEST_PORT": "9090"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"port": 9090},
		},
		{
			name: "env var selects config file",
			env:  map[string]string{"TEST_CONFIG": "testdata/config.yml"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"foo": "file-foo", "bar": "file-bar"},
		},
		{
			name: "config flag overrides config file from env var",
			args: []string{"--config=testdata/config.yml"},
			env:  map[string]string{"TEST_CONFIG": "testdata/does-not-exist.yml"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"foo": "file-foo", "bar": "file-bar"},
		},
		{
			name:    "env var with invalid value for int flag returns error",
			env:     map[string]string{"TEST_PORT": "abc"},
			opts:    []configger.Option{configger.WithEnvPrefix("TEST")},
			wantErr: true,
		},
		{
			name: "env var without a flag is ignored",
			env:  map[string]string{"TEST_EXTRA": "env-extra"},
			opts: []configger.Option{configger.WithEnvPrefix("TEST")},
			want: map[string]any{"extra": ""},
		},
		{
			name: "WithoutConfigurationFile skips loading even when config flag is set",
			args: []string{"--config=testdata/config.yml"},
			opts: []configger.Option{configger.WithoutConfigurationFile()},
			want: map[string]any{"foo": "default-foo", "bar": "default-bar", "baz": []string{"default-baz"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			f := newFlagSet(t, tt.args)

			got, gotErr := configger.LoadConfig(f, tt.opts...)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("LoadConfig() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("LoadConfig() succeeded unexpectedly")
			}

			for key, want := range tt.want {
				switch v := want.(type) {
				case string:
					if got := got.String(key); got != v {
						t.Errorf("LoadConfig() key %q = %q, want %q", key, got, v)
					}
				case []string:
					if got := got.Strings(key); !slices.Equal(got, v) {
						t.Errorf("LoadConfig() key %q = %q, want %q", key, got, v)
					}
				case int:
					if got := got.Int(key); got != v {
						t.Errorf("LoadConfig() key %q = %d, want %d", key, got, v)
					}
				default:
					t.Errorf("unhandled type: %t", want)
					continue
				}
			}
		})
	}
}
