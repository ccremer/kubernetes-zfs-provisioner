package zfs

import "strings"

// shellQuote returns s quoted so a POSIX shell (sh/dash/bash/ksh) receives it as
// a single argument, regardless of spaces or metacharacters. Safe, common
// characters are returned unquoted; everything else is wrapped in single quotes
// with embedded single quotes escaped as '\”.
//
// This replaces the previous approach of letting the remote login shell re-split
// a flattened command string (see docker/zfs.sh), which broke arguments that
// legitimately contain spaces, e.g. a multi-network sharenfs "rw=@a ro=@b".
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if isShellSafe(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func isShellSafe(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune("@%_+=:,./-", r):
		default:
			return false
		}
	}
	return true
}
