# configger

An opinionated loader that merges a program's configuration from command line flags, a YAML file and environment variables into one read-only view.

## Language

**Source**:
One place configuration values come from: flag defaults, the config file, environment variables, or command line flags.
_Avoid_: provider, layer

**Precedence**:
The fixed order in which sources override each other for a key: flag defaults, then config file, then env vars, then command line flags.
_Avoid_: priority, merge order

**Config key**:
The dotted name a value is read by, matching the flag name (e.g. `log-level`, `db.host`).
_Avoid_: path, setting name

**Config file key**:
The flag whose value names the YAML config file to load; `config` unless changed.
_Avoid_: config flag

**Env prefix**:
The upper-case prefix that marks an environment variable as belonging to this program; without one, no env vars are read.

**Env name**:
The environment variable that sets a flag: the env prefix, an underscore, then the flag name upper-cased with `-` and `.` turned into `_` (e.g. `APP_LOG_LEVEL`).
_Avoid_: env key, env var name
