package steamcmd

import (
	"errors"
	"strconv"
	"strings"
)

func (scmd *SteamCmd) parseStdOut(text string) error {
	lower := strings.ToLower(text)

	if strings.Contains(lower, "steam guard code:") {
		return errors.New(STATUS_STEAMGUARD_CODE)
	} else if strings.Contains(lower, "invalid password") {
		return errors.New(STATUS_INVALID_USER)
	} else if strings.Contains(lower, "error (rate limit exceeded)") {
		return errors.New(STATUS_STEAMCMD_RATELIMIT)
	} else if strings.Contains(lower, "error! missing parameters.") {
		return errors.New(STATUS_STEAMCMD_MISSING_PARAMS)
	}

	// Parse: "progress: 45.23 (downloaded / total)"
	if idx := strings.Index(lower, "progress: "); idx != -1 {
		rest := lower[idx+len("progress: "):]
		if end := strings.IndexAny(rest, " ("); end != -1 {
			rest = rest[:end]
		}
		if progress, err := strconv.ParseFloat(rest, 64); err == nil {
			scmd.ChangeProgress(progress)
		}
	}

	return nil
}
