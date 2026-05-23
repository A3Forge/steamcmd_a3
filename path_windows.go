//go:build windows

package steamcmd

import (
	"path/filepath"
)

func (scmd *SteamCmd) steamCmdFilePath() string {
	return filepath.Join(scmd.steamCmdPath(), "steamcmd.exe")
}
