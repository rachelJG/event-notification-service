Generate unit tests for the file or function specified in $ARGUMENTS.

## Guidelines

### Test Structure
- Use **table-driven tests** with `t.Run()` for each test case
- Name tests as `Test<Function>_<Scenario>` (e.g., `TestParseLog_EmptyInput`)
- Place the test file alongside the source file with the `_test.go` suffix
- Use `t.Parallel()` when tests are independent and have no shared mutable state
- Use `t.Helper()` in any test helper functions

### Test Coverage
Include at least the following cases for each function:
- **Happy path**: valid input producing expected output
- **Error cases**: invalid input, expected errors
- **Edge cases**: empty/nil inputs, boundary values, zero values
- **Concurrency safety**: if the function is used concurrently, test with `-race`

### Assertions with Testify
- Use `github.com/stretchr/testify/assert` for non-fatal assertions
- Use `github.com/stretchr/testify/require` for fatal assertions that should stop the test
- Prefer `require` for setup steps and `assert` for actual test validations
- Use descriptive failure messages: `assert.Equal(t, expected, actual, "description of what failed")`

### Mocking with GoMock
- Use `go.uber.org/mock/gomock` to generate and use mocks for interface dependencies
- Generate mocks with `mockgen` and place them in a `mocks/` subdirectory within the package
- Always call `ctrl.Finish()` or use `t.Cleanup(ctrl.Finish)` to verify expectations
- Set up explicit expectations with `EXPECT()` for each mock call
- Use `gomock.Any()` only when the argument value is truly irrelevant to the test

### Execution
After generating the tests, run `go vet` first to catch common issues:
```bash
go vet ./path/to/package/...
```

Then run the tests:
```bash
go test -v -race -count=1 ./path/to/package/...
```

Then show coverage:
```bash
go test -cover ./path/to/package/...
```

Report any failures and suggest fixes.
