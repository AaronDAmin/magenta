# Contributing to Magenta

Thanks for taking the time to contribute! Here's everything you need to get started.

---

## Getting Started

**1. Fork & clone**

```bash
git clone https://github.com/<your-username>/magenta.git
cd magenta
```

**2. Set up the project**

```bash
cp config.yaml.example config.yaml
# Edit config.yaml with your local media paths
go mod tidy
go run ./cmd/mediaserver
```

**3. Create a branch**

```bash
git checkout -b feat/your-feature   # new feature
git checkout -b fix/your-bug        # bug fix
```

---

## Commit Style

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add subtitle support
fix: correct range header parsing for large files
refactor: simplify media index lookup
docs: update API reference in README
```

---

## Before Submitting a PR

```bash
go build ./...       # must compile
go vet ./...         # must pass
```

- Keep changes focused — one PR per feature or fix
- Do not commit `config.yaml`, `users.json`, `*.key`, or `*.crt`
- Match the existing code style

---

## Reporting Bugs

Open an issue using the **Bug Report** template. Include logs (`log.level: debug` in `config.yaml`), your OS, and Go version.

## Suggesting Features

Open an issue using the **Feature Request** template before starting implementation — it avoids duplicate work.

---

## License

By contributing, you agree that your contributions will be licensed under the [GPL v3](LICENSE).
