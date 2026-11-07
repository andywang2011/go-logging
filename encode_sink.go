package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

type encodeSink struct {
	output   string
	writer   io.Writer
	fmtColor *int8
	indent   bool
}

func newEncodeSink(cfg Config) (*encodeSink, error) {
	s := &encodeSink{output: cfg.Output}
	switch cfg.Output {
	case OutputFile:
		topic := cfg.LogTopic
		if topic == "" {
			topic = "app"
		}
		s.writer = &lumberjack.Logger{
			Filename:   "log/" + topic + "/app.log",
			MaxSize:    cfg.FileMaxSize,
			MaxBackups: cfg.FileBackupCount,
			LocalTime:  true,
		}
	case OutputFmt:
		var n int8
		s.fmtColor = &n
		s.indent = true
	default:
		// stdout reads os.Stdout at Write time so tests can redirect it.
	}
	return s, nil
}

func (s *encodeSink) Write(item *LogItem, encrypt bool) error {
	if encrypt {
		maskLogItem(item)
	}
	b, err := encodeLogItem(item, s.indent)
	if err != nil {
		return err
	}
	if s.output == OutputFmt && s.fmtColor != nil {
		*s.fmtColor = (*s.fmtColor + 3) & 7
		_, err = fmt.Printf("\x1b[0;%dm%s\x1b[0m", *s.fmtColor+30, b)
		return err
	}
	w := s.writer
	if w == nil {
		_, err = os.Stdout.Write(b)
		return err
	}
	_, err = w.Write(b)
	return err
}

func (s *encodeSink) Close() error {
	if c, ok := s.writer.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func encodeLogItem(item *LogItem, indent bool) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if indent {
		enc.SetIndent("", "\t")
	}
	if err := enc.Encode(item); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
