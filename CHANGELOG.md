# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-11-07

### Breaking

- See repository diff for API changes in this release (secure defaults, renamed packages where applicable, context-aware constructors).

### Security

- Hardened defaults: TLS, timeouts, body limits, PII redaction where logging applies.

### Fixed

- Bug fixes from the v0.2.0 hardening pass.

### Changed

- CI uses go-version-file; added race, staticcheck, govulncheck, golangci-lint (gosec).



### Security / Breaking

- Removed AES-ECB field masking; matched fields are always replaced with `***`
- `SetupFromConfig` now returns `ErrAESKeyUnsupported` when `Config.AESKey` is non-empty (`AESFields` / `AESDisabled` keep their names and now only control `***` masking)
- `DefaultRedactor.RedactJSONBody` returns `[non-json body omitted]` for non-JSON or unparseable bodies instead of echoing them

### Added

- `Redactor` interface and `DefaultRedactor` for HTTP header / JSON body redaction (shared by API frameworks)

## [0.1.0] - 2026-05-19

### Added

- Async log pipeline with bounded queue (default 4096) and drop-on-full policy
- Field-level masking (`***` or AES-ECB mask) before JSON encode
- File rotation via lumberjack, stdout/std/fmt outputs
- `Close()` for graceful drain; `DroppedLogs` / `WrittenLogs` counters
- Package-level concurrency safety (`coreMu`, `cryptoMu`)
- Level constants `LevelFatal` … `LevelInfo` for JSON `level` field

### Security

- Documented: AES masking is not a substitute for secrets management or transport encryption

[0.1.0]: https://github.com/andywang2011/go-logging/releases/tag/v0.1.0

