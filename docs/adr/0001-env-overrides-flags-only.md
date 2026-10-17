# Env vars only override flag-backed keys, applied through the FlagSet

Environment variables used to be mapped to config keys by guessing: `_` became `.` and any comma split the value into a slice. That broke flags with dashes in their names (`APP_LOG_LEVEL` landed on `log.level`, never `log-level`), corrupted string values containing commas, and turned a single value for a slice flag into an empty slice. We decided that an env var can only set a key that has a flag, and that its value is applied with `FlagSet.Set` on flags not already set on the command line, so each flag parses its own value with its own type.

## Considered Options

- **Keep reaching YAML-only nested keys from env.** Rejected: without a flag there is no type to parse with, which is exactly what caused the defects. A key that should be settable from env gets a flag.
- **Leave the FlagSet untouched and switch on `Value.Type()`.** Rejected: it only handles types we list, and invalid numbers silently read as zero instead of failing.

## Consequences

- `LoadConfig` changes the caller's FlagSet: a flag set from env reports `Changed` and holds the env value, so the FlagSet shows the value in effect.
- An env var can choose the config file (e.g. `APP_CONFIG`), since env is applied before the config file key is read.
- An invalid env value, or two flags with the same env name, makes `LoadConfig` return an error.
