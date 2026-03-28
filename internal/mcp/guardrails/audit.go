package guardrails

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

// AuditEntry is a single structured audit log record.
type AuditEntry struct {
	Timestamp  string `json:"ts"`
	Tool       string `json:"tool"`
	Result     string `json:"result"`               // "ok" or "blocked"
	Reason     string `json:"reason,omitempty"`     // set when result == "blocked"
	FilesCount int    `json:"files,omitempty"`      // number of files written/scanned
	DurationMs int64  `json:"duration_ms,omitempty"` // wall-clock ms
}

// Auditor writes JSON-lines audit logs to an io.Writer (typically os.Stderr).
// Writing to stderr keeps audit logs out of the stdio MCP transport.
type Auditor struct {
	w io.Writer
}

// NewAuditor creates an Auditor writing to w. If w is nil, os.Stderr is used.
func NewAuditor(w io.Writer) *Auditor {
	if w == nil {
		w = os.Stderr
	}
	return &Auditor{w: w}
}

// Log serialises entry as a JSON line. Errors are silently discarded to
// ensure logging never interferes with normal operation.
func (a *Auditor) Log(entry AuditEntry) {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = a.w.Write(append(b, '\n'))
}
