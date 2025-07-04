# UI Testing for lmon

This directory contains UI tests for the lmon web interface using the [go-rod](https://github.com/go-rod/rod) library, which is a high-level driver for web automation and testing.

## Prerequisites

- Go 1.24 or later
- Chrome or Chromium browser installed (go-rod will download and use its own browser if none is found)

## Running the Tests

To run the UI tests, simply execute:

```bash
go test -v ./uitest
```

The tests will:
1. Start the lmon web server programmatically
2. Check that the server is healthy
3. Run the UI tests
4. Shut down the server when tests are complete

To skip UI tests (useful in CI environments without a display):

```bash
go test -v -short ./uitest
```

## Test Coverage

The UI tests cover the following functionality:

### Dashboard Page Tests
- Verify the page loads correctly
- Verify system, disk, and health check cards are present
- Verify percentages are properly rounded to 2 decimal places
- Verify memory icon is correct (memory_alt)

### Configuration Page Tests
- Verify the page loads correctly
- Verify disk and system configuration cards are present
- Verify root partition doesn't have a delete icon
- Verify memory icon is correct in system configuration (memory_alt)

## Adding New Tests

To add new UI tests:

1. Add new test functions to `ui_test.go`
2. Follow the pattern of existing tests, using go-rod's API to interact with the page
3. Use assertions to verify expected behavior

## Troubleshooting

If tests fail, check:

1. Is the lmon server running on http://localhost:8080?
2. Is Chrome/Chromium installed and accessible?
3. Are there any JavaScript errors in the browser console?

For more detailed debugging, you can modify the tests to run in non-headless mode by changing:

```
launcher := launcher.New().Headless(true)
```

to:

```
launcher := launcher.New().Headless(false)
```

This will open a visible browser window during test execution.

## Running in Restricted Environments

When running in environments with restricted permissions (like CI systems or containers), you may encounter sandbox-related errors. In these cases, add the `--no-sandbox` flag:

```
launcher := launcher.New().Headless(true).Set("no-sandbox")
```

This disables the Chrome sandbox, which is less secure but allows the browser to run in restricted environments.
