## ADDED Requirements

### Requirement: Build Version Display

The runner SHALL display version and build information at startup and via command-line flag.

#### Scenario: Version in startup banner
- **WHEN** the runner starts
- **THEN** the startup banner displays the version number and build timestamp

#### Scenario: Version flag
- **WHEN** the user runs the runner with `--version` flag
- **THEN** the runner prints version, build time, and git commit then exits
