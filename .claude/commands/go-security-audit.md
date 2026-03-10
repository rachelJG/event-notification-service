---
description: Run security audit on Go project using govulncheck and golangci-lint
---

# Go Security Audit

Run a full security audit on the Go project using official and community tools.

## Steps

// turbo-all

1. Ensure `govulncheck` is installed:
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

2. Ensure `golangci-lint` is installed:
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

3. Run `govulncheck` to detect known vulnerabilities in dependencies:
```bash
govulncheck ./...
```

4. Run `golangci-lint` with security-focused linters enabled:
```bash
golangci-lint run --enable gosec,bodyclose,sqlclosecheck --timeout 5m ./...
```

5. Summarize the results to the user, highlighting:
   - Any known vulnerabilities found by `govulncheck` (CVEs, affected packages)
   - Security issues found by `gosec` via `golangci-lint`
   - Recommended fixes or upgrades for each issue
