package configger_test

import (
	"slices"
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

	if err := f.Parse(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	return f
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
				default:
					t.Errorf("unhandled type: %t", want)
					continue
				}
			}
		})
	}
}
