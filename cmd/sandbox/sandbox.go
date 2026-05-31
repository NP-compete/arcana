package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

// secretPrefixes lists environment variable prefixes that must never be leaked
// to sandboxed user code. Any env var whose name starts with one of these
// prefixes (case-insensitive) is stripped by SanitizeEnv.
var secretPrefixes = []string{
	"POSTGRES_",
	"MINIO_",
	"JWT_",
	"ADMIN_",
	"AUDIT_HMAC_",
	"AWS_",
	"REDIS_",
	"NATS_",
	"TEMPORAL_",
}

// safeKeys is the allowlist of environment variables that are safe to expose
// to sandboxed processes.
var safeKeys = map[string]bool{
	"PATH": true,
	"HOME": true,
	"LANG": true,
	"TZ":   true,
}

// SanitizeEnv returns a minimal, safe environment variable slice suitable for
// sandboxed process execution. Only PATH, HOME, LANG, and TZ are included.
// All secret-bearing variables (POSTGRES_*, MINIO_*, JWT_*, ADMIN_*, etc.)
// are explicitly excluded.
func SanitizeEnv() []string {
	var safe []string
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]

		if !safeKeys[key] {
			continue
		}

		// Double-check the key does not match any secret prefix.
		skip := false
		for _, prefix := range secretPrefixes {
			if strings.HasPrefix(strings.ToUpper(key), prefix) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		safe = append(safe, entry)
	}
	return safe
}

// limitedWriter wraps a bytes.Buffer and stops writing once limit bytes have
// been written. It prevents runaway output from consuming unbounded memory.
type limitedWriter struct {
	buf     *bytes.Buffer
	limit   int64
	written int64
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.written
	if remaining <= 0 {
		return len(p), nil // discard silently
	}
	toWrite := p
	if int64(len(p)) > remaining {
		toWrite = p[:remaining]
	}
	n, err := w.buf.Write(toWrite)
	w.written += int64(n)
	if err != nil {
		return n, err
	}
	if w.written >= w.limit && int64(len(p)) > remaining {
		w.buf.WriteString(fmt.Sprintf("\n... output truncated at %d bytes", w.limit))
	}
	return len(p), nil
}
