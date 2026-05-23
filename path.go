package steamcmd

import (
	"os"
	"path/filepath"
)

func (scmd *SteamCmd) SetSteamCmdDir(p string) error {
	scmd.mu.Lock()
	defer scmd.mu.Unlock()

	scmd.SteamCmdDir = filepath.Clean(p)

	return nil
}

func (scmd *SteamCmd) GetSteamCmdDir() string {
	scmd.mu.RLock()
	defer scmd.mu.RUnlock()

	cleanPath := filepath.Clean(scmd.SteamCmdDir)

	if _, err := os.Stat(cleanPath); err != nil {
		os.MkdirAll(cleanPath, 0755)
	}

	return scmd.SteamCmdDir
}

func (scmd *SteamCmd) steamCmdPath() string {
	return filepath.Join(scmd.GetSteamCmdDir(), "steamcmd_wr")
}

func (scmd *SteamCmd) steamCmdExePath() string {
	return filepath.Join(scmd.steamCmdPath(), "steamcmd.exe")
}

func (scmd *SteamCmd) steamCmdBinExists() bool {
	if _, err := os.Stat(scmd.steamCmdFilePath()); err != nil {
		return false
	}

	return true
}
