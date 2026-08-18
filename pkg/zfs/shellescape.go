package zfs

import "strings"

// shellQuote returns s quoted so a POSIX shell (sh/dash/bash/ksh) receives it as
// a single argument, regardless of spaces or metacharacters. Safe, common
// characters are returned unquoted; everything else is single-quoted, and any
// embedded single quote is escaped so the quoting cannot be broken out of.
//
// Quoting the arguments here means the remote login shell cannot re-split them,
// so a value that legitimately contains spaces (a multi-network sharenfs such as
// "rw=@a ro=@b") survives intact.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if isShellSafe(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// isShellSafe reports whether s consists only of characters that a POSIX shell
// leaves untouched, so it can be passed without quoting.
func isShellSafe(s string) bool {
	const unquotedPunct = "@%_+=:,./-"
	for _, r := range s {
		safe := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			strings.ContainsRune(unquotedPunct, r)
		if !safe {
			return false
		}
	}
	return true
}
