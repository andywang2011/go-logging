package logging

// Log level values written to the JSON level field.
const (
	// LevelFatal is the fatal log level (JSON level value 1).
	LevelFatal = 1
	// LevelError is the error log level (JSON level value 2).
	LevelError = 2
	// LevelWarn is the warn log level (JSON level value 3).
	LevelWarn = 3
	// LevelInfo is the info log level (JSON level value 4).
	LevelInfo = 4
)

// Config holds logging module settings for injection and unit tests.
type Config struct {
	// Output selects the sink: file, fmt, or std.
	Output string
	// AESDisabled disables field masking when true.
	AESDisabled bool
	// AESKey must be empty. AES field masking was removed; a non-empty value is rejected.
	AESKey string
	// AESFields lists field names to mask with "***" (case-insensitive substring match).
	AESFields []string
	// FileMaxSize is the max log file size in megabytes (file output only).
	FileMaxSize int
	// FileBackupCount is the number of rotated backups to retain (file output only).
	FileBackupCount int
	// LogTopic is the directory name under log/<LogTopic>/app.log (file output only).
	LogTopic string

	// AsyncSync forces synchronous writes when true (tests/debug); default is async queue.
	AsyncSync bool
	// AsyncQueueSize is the async queue capacity (default 4096).
	AsyncQueueSize int
	// AsyncBlockWhenFull blocks the caller when the queue is full; default drops and increments DroppedLogs.
	AsyncBlockWhenFull bool
}

const (
	// OutputFile writes rotated JSON logs under log/<LogTopic>/app.log.
	OutputFile = "file"
	// OutputFmt prints colorized JSON to stdout.
	OutputFmt = "fmt"
	// OutputStd writes JSON lines to stdout.
	OutputStd = "std"

	defaultAsyncQueueSize = 4096
)

// ConfigDefault returns async logging defaults (Output still required for Setup).
func ConfigDefault() Config {
	return Config{
		Output:         OutputStd,
		AsyncQueueSize: defaultAsyncQueueSize,
	}
}
