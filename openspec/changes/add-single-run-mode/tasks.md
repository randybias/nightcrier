## 1. Implementation

- [x] 1.1 Add `--single-run` flag in `init()` function (`cmd/nightcrier/main.go`)
- [x] 1.2 Read flag value in `run()` function
- [x] 1.3 After successful `processEvent()` call, trigger `cancel()` if single-run mode
- [x] 1.4 Add info log message when exiting in single-run mode
- [x] 1.5 Add guard to cleanly drop events arriving during shutdown

## 2. Testing

- [x] 2.1 Add unit test for single-run flag parsing
- [x] 2.2 Manual verification with test cluster

## 3. Documentation

- [x] 3.1 Add `--single-run` to `--help` output (automatic via cobra)
