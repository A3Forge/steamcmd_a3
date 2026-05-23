package steamcmd

import "os/exec"

func (scmd *SteamCmd) runCmd(params ...string) *exec.Cmd {
	cmd := exec.Command(scmd.steamCmdExePath(), params...)

	scmd.mu.Lock()
	scmd.cmd = cmd
	scmd.mu.Unlock()

	return cmd
}
