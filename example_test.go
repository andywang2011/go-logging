package logging_test

import (
	"github.com/andywang2011/go-logging"
)

func ExampleSetupFromConfig() {
	err := logging.SetupFromConfig(logging.Config{
		Output:      logging.OutputFmt,
		LogTopic:    "my-app",
		AESDisabled: true,
	})
	if err != nil {
		return
	}
	defer logging.Close()

	logging.New().Info("service started", nil, nil)
}
