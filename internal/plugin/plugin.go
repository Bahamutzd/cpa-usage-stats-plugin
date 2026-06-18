// Package plugin implements the host-facing RPC dispatch for the
// cpa-usage-stats plugin. It owns plugin lifecycle (register/reconfigure/
// shutdown), routes usage.handle into the local SQLite store, and forwards
// management.* requests to the embedded HTTP router via an adapter.
package plugin

import (
	"encoding/json"
	"sync"
)

// Envelope mirrors pluginabi.Envelope. The plugin must marshal one of these
// for every RPC call the host issues.
type Envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *EnvelopeError  `json:"error,omitempty"`
}

// EnvelopeError mirrors pluginabi.Error.
type EnvelopeError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable,omitempty"`
}

var (
	stateMu sync.Mutex
	state   *runtimeState
)

// Dispatch routes a single RPC call to the appropriate handler and returns the
// fully marshalled envelope bytes. It never returns nil — callers always get a
// JSON envelope to write back to the host.
func Dispatch(method string, payload []byte) []byte {
	switch method {
	case "plugin.register":
		return handleRegister(payload)
	case "plugin.reconfigure":
		return handleReconfigure(payload)
	case "plugin.shutdown":
		return handleShutdown()
	case "usage.handle":
		return handleUsage(payload)
	case "management.register":
		return handleManagementRegister(payload)
	case "management.handle":
		return handleManagementHandle(payload)
	default:
		return ErrorEnvelope("unknown_method", "unknown method: "+method)
	}
}

// Shutdown is invoked by the cgo shim when the host unloads the library.
func Shutdown() {
	stateMu.Lock()
	defer stateMu.Unlock()
	if state != nil {
		state.Close()
		state = nil
	}
}

// OkEnvelope marshals a success envelope around an arbitrary result value.
func OkEnvelope(result any) []byte {
	if result == nil {
		raw, _ := json.Marshal(Envelope{OK: true, Result: json.RawMessage(`{}`)})
		return raw
	}
	body, errMarshal := json.Marshal(result)
	if errMarshal != nil {
		return ErrorEnvelope("encode_error", errMarshal.Error())
	}
	raw, _ := json.Marshal(Envelope{OK: true, Result: body})
	return raw
}

// ErrorEnvelope marshals a failure envelope with the given code and message.
func ErrorEnvelope(code, message string) []byte {
	raw, _ := json.Marshal(Envelope{OK: false, Error: &EnvelopeError{Code: code, Message: message}})
	return raw
}
