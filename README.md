# go-logging

[![test](https://github.com/andywang2011/go-logging/actions/workflows/test.yml/badge.svg)](https://github.com/andywang2011/go-logging/actions/workflows/test.yml)

Structured JSON business logging keyed by `request_id` and v1–v3 dimensions, with rotation, async writes, and field masking.

## Install

```bash
go get github.com/andywang2011/go-logging@v0.1.0
```

## Quick start

```go
err := logging.SetupFromConfig(logging.Config{
	Output:      logging.OutputFmt,
	LogTopic:    "my-app",
	AESDisabled: true,
})
if err != nil {
	// handle
}
defer logging.Close()

logging.New().Info("service started", nil, nil)
```

## Examples

```bash
cd go-logging/examples/context && go run .
```

## License

MIT — see [LICENSE](LICENSE).
