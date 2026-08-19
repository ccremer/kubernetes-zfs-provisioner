package zfs

import (
	"context"
	"errors"
	"strings"
)

// RunError is returned when a zfs/chmod invocation failed. Ran is true only if
// the command actually executed on the ZFS host (local exec or a remote session
// that started). Transport failures (SSH dial, timeout before exec) have Ran=false
// and must never be treated as "dataset does not exist".
type RunError struct {
	Op     string
	Host   string
	Err    error
	Stderr string
	Exit   int // -1 if unknown
	Ran    bool
}

func (e *RunError) Error() string {
	if e == nil || e.Err == nil {
		return "zfs command failed"
	}
	if e.Stderr != "" {
		return e.Op + ": " + e.Err.Error() + ": " + e.Stderr
	}
	return e.Op + ": " + e.Err.Error()
}

func (e *RunError) Unwrap() error { return e.Err }

// not-found strings observed on OpenZFS (Linux), illumos/OpenIndiana and FreeBSD.
var notFoundPatterns = []string{
	"does not exist",
	"dataset does not exist",
	"no such dataset",
	"no such file or directory",
}

func looksLikeNotFound(s string) bool {
	s = strings.ToLower(s)
	for _, p := range notFoundPatterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// IsNotFound reports that ZFS itself confirmed the dataset is absent.
// Transport and permission errors are not not-found.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var re *RunError
	if errors.As(err, &re) {
		if !re.Ran {
			return false
		}
		return looksLikeNotFound(re.Stderr) || looksLikeNotFound(re.Error())
	}
	return looksLikeNotFound(err.Error())
}

var transientPatterns = []string{
	"timeout", "timed out", "i/o timeout", "deadline exceeded",
	"connection refused", "connection reset", "broken pipe",
	"no route to host", "network is unreachable",
	"ssh dial", "temporary", "unavailable", "try again",
	"resource temporarily", "eof",
	"dataset is busy", "target is busy",
}

// IsTransient reports errors that should be retried (network blip, busy dataset).
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var re *RunError
	if errors.As(err, &re) && !re.Ran {
		return true
	}
	msg := strings.ToLower(err.Error())
	// Permanent: do not treat as transient even if another pattern matches.
	if strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "invalid property") ||
		strings.Contains(msg, "invalid dataset") ||
		strings.Contains(msg, "bad property") {
		return false
	}
	for _, p := range transientPatterns {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}
