// Context-scoped logging (request_id + v1–v3), different from README quick start (process setup).
package main

import (
	"fmt"

	"github.com/andywang2011/go-logging"
)

func main() {
	if err := logging.SetupFromConfig(logging.Config{
		Output:      logging.OutputFmt,
		AESDisabled: true,
		AsyncSync:   true,
	}); err != nil {
		panic(err)
	}
	defer logging.Close()

	ctx := logging.NewContext("req-1", "order", "POST /orders", "create")
	log := logging.FromContext(ctx)
	log.Info("order accepted", map[string]int{"order_id": 42}, nil)
	fmt.Println("logged with request context")
}
