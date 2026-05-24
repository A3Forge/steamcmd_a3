package steamcmd

import (
	"bufio"
	"fmt"
	"path/filepath"
)

const (
	ARMA_3_SERVER_APPID = "233780"
	ARMA_3_APPID        = "107410"
)

func (scmd *SteamCmd) ValidateArma3Server(arma3ServerPath string) error {
	if err := scmd.beginCmdOperation(); err != nil {
		return err
	}

	if err := scmd.ValidateSteamCMD(); err != nil {
		scmd.cmdMu.Unlock()
		return err
	}

	scmd.ChangeStatus(STATUS_STEAMCMD_INTALLING_A3SERVER, "")

	go scmd.installUpdateAppLocked(filepath.Clean(arma3ServerPath))

	return nil
}

func (scmd *SteamCmd) installUpdateAppLocked(serverPath string) {
	defer scmd.cmdMu.Unlock()

	resultStatus := STATUS_FINE
	defer func() {
		scmd.ChangeStatus(resultStatus, "")
	}()

	args := []string{"+force_install_dir", serverPath}
	args = append(args, scmd.credentialsArgs()...)
	args = append(args, "+app_update", ARMA_3_SERVER_APPID, "-beta", "public", "validate")
	args = append(args, "+quit")

	cmd := scmd.runCmd(args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		scmd.clearCmd(cmd)
		return
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		fmt.Println(err)
		scmd.clearCmd(cmd)
		return
	}

	scanner := bufio.NewScanner(stdout)

	var text string
	for scanner.Scan() {
		text = scanner.Text()
		fmt.Println(text)
		if err := scmd.parseStdOut(text); err != nil {
			resultStatus = err.Error()
			_ = scmd.kill()
			return
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		_ = scmd.kill()
		return
	}

	_ = cmd.Wait()
	scmd.clearCmd(cmd)
}
