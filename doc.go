// Package logging provides structured JSON business logs keyed by request_id and v1–v3 dimensions.
//
// Design constraints (for use in application frameworks):
//   - No viper, no log.Fatal inside the library.
//   - ConfigDefault() supplies async queue defaults; Output must still be set for Setup.
//   - Config is injected by the caller via SetupFromConfig (process-level; call once, then Close).
//   - Call Close before process exit to drain the async queue.
//
// Features:
//   - File rotation (lumberjack), stdout, or colored fmt output.
//   - Async writes with drop-on-full (default) or optional blocking mode.
//   - Field-level masking with "***" for sensitive request/response fields (no AES).
//   - AES masking was removed; Config.AESKey must be empty (SetupFromConfig returns ErrAESKeyUnsupported otherwise).
//     Config.AESFields / AESDisabled keep their names for compatibility but only control "***" masking.
//   - DefaultRedactor.RedactJSONBody replaces non-JSON bodies with "[non-json body omitted]".
package logging
