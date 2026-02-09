# Contributing

Contributions to ghmux are welcome!

## Issues

Please use [Issues](https://github.com/nurazon59/ghmux/issues) for bug reports and feature requests.

- **Bug reports**: Include steps to reproduce, expected behavior, and actual behavior
- **Feature requests**: Describe the use case and the problem you want to solve

## Development Setup

```bash
git clone https://github.com/nurazon59/ghmux.git
cd ghmux
go test ./...
```

## Pull Requests

1. Open an issue first to discuss the approach
2. Fork the repository and create a branch
3. Implement your changes
4. Ensure tests and lint pass

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/).

```
<type>(<scope>): <subject>
```

Examples: `feat(sync): add bash hook support`, `fix(config): handle missing profile`
