# Contributing to krnr

Thank you for your interest in contributing to krnr! We welcome contributions — whether you’re fixing bugs, adding features, or improving documentation.

## Table of Contents
- [Community Guidelines](#community-guidelines)
- [How to Contribute](#how-to-contribute)
  - [Follow the Code of Conduct](#follow-the-code-of-conduct)
  - [Check Existing Issues](#check-existing-issues-and-discussions)
  - [Fork and Clone](#fork-and-clone-the-repository)
  - [Development Environment](#development-environment)
  - [Making Changes](#making-changes)
  - [Testing](#testing)
  - [Submitting a Pull Request](#submitting-a-pull-request)
- [Coding Standards](#coding-standards)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Features](#suggesting-features)
- [Reporting Security Issues](#reporting-security-issues)
- [Getting Help](#getting-help)

## Community Guidelines
To maintain a safe and productive environment, please adhere to the following rules:

- **No Spam or Unverified Links**: Do not post unverified links or promotional content in issues or PRs.
- **Provide Constructive Feedback**: When reporting issues or reviewing PRs, include clear details, reproducible steps, and logs when relevant.
- **Respect the Community**: Be respectful and professional in comments, issues, and PRs.

If you notice violations, report them to [izeno.contact@gmail.com](mailto:izeno.contact@gmail.com) or open a private issue labeled "Code of Conduct Violation."

## How to Contribute

### Follow the Code of Conduct
All contributors are expected to adhere to our [Code of Conduct](CODE_OF_CONDUCT.md) and [Security Policy](SECURITY.md). By contributing, you agree that your contributions will be licensed under the project’s [MIT License](LICENSE).

### Check Existing Issues and Discussions
Look through the [Issues page](https://github.com/VoxDroid/krnr/issues) and [Discussions](https://github.com/VoxDroid/krnr/discussions) to see whether your idea or bug already exists. Use the appropriate issue templates in `.github/ISSUE_TEMPLATE/` when opening a new issue:
- `bug_report.yml`
- `feature_request.yml`
- `support_question.yml`
- `documentation_issue.yml`
- `security_report.yml`

### Fork and Clone the Repository
1. Fork the repository on GitHub and clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/krnr.git
   cd krnr
   ```

### Development Environment
1. Ensure you have **Go 1.22+** installed and `GOPATH`/`GOMOD` set up.
2. Build locally:
   ```bash
   go build -v -o krnr .
   ```
3. Format and lint before committing (scripts are provided):
   ```bash
   ./scripts/fmt.sh
   ./scripts/lint.sh
   ```

### Making Changes
- Keep changes small and focused; open an issue first for larger work.
- Write clear commit messages and reference the issue (e.g., `Fixes #123`).
- Add or update tests in the appropriate package under `internal/` or `cmd/`.
- Update `docs/` or `README.md` if the user-visible behavior changes.

### Testing
- Run unit tests:
  ```bash
  go test ./...
  ```
- For package-specific tests use `go test ./pkg/path -v`.
- Ensure linters and formatters pass locally.

### Submitting a Pull Request
1. Push your branch to your fork:
   ```bash
   git push origin feature/your-branch
   ```
2. Create a PR against `VoxDroid/krnr:main` and use the [Pull Request template](.github/PULL_REQUEST_TEMPLATE.md).
3. Include a clear description, motivation, and links to issues addressed.
4. Respond to review feedback and update your branch as needed.

## Coding Standards
- Use `gofmt`/`gofumpt` for formatting and `go vet` for basic checks.
- Follow idiomatic Go patterns and keep packages small and focused.
- Use descriptive names and add tests for non-trivial logic.
- Run `golangci-lint` via `./scripts/lint.sh` and fix reported issues.

## Reporting Bugs
Open an issue using the Bug Report template (`.github/ISSUE_TEMPLATE/bug_report.yml`) and include:
- Steps to reproduce the issue
- Expected vs actual behavior
- Relevant logs and environment (OS, Go version, krnr version)

## Suggesting Features
Open a Feature Request (`.github/ISSUE_TEMPLATE/feature_request.yml`) or start a Discussion with a clear use case, benefits, and potential drawbacks.

## Reporting Security Issues
If you discover a security vulnerability, please follow our [Security Policy](SECURITY.md):
- Email [izeno.contact@gmail.com](mailto:izeno.contact@gmail.com) with details (preferred).
- Use the `security_report.yml` template only if public disclosure is acceptable.

## Getting Help
Check `docs/` and the [Discussions tab](https://github.com/VoxDroid/krnr/discussions) for community support, or open an issue marked as a support question.

Thank you for contributing to krnr!