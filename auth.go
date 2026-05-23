package steamcmd

import (
	"bufio"
	"errors"
	"fmt"
)

func (scmd *SteamCmd) SetActiveUser(id int) error {
	scmd.mu.Lock()

	_, ok := scmd.CredentialsStore.AllUsers[id]
	if !ok {
		scmd.mu.Unlock()
		return fmt.Errorf("User with id %d not found", id)
	}

	scmd.CredentialsStore.ActiveUserID = id

	scmd.mu.Unlock()
	scmd.saveOrResetCredentials()

	return nil
}

func (scmd *SteamCmd) TryLoginWithSteamGuardCode(code string) error {
	if code == "" {
		return errors.New("SteamGuard code is empty!")
	}

	scmd.mu.Lock()
	scmd.SGuardCode = code
	scmd.mu.Unlock()

	if err := scmd.TryLogin(); err != nil {
		return err
	}

	scmd.mu.Lock()
	scmd.SGuardCode = ""
	scmd.mu.Unlock()

	return nil
}

func (scmd *SteamCmd) TryLogin() (err error) {
	if err := scmd.beginCmdOperation(); err != nil {
		return err
	}
	defer scmd.cmdMu.Unlock()

	if err := scmd.ValidateSteamCMD(); err != nil {
		return err
	}

	activeUser := scmd.GetActiveUser()
	if activeUser.Password == "" || activeUser.Username == "" {
		return ErrCredentialsIsEmpty
	}

	args := scmd.credentialsArgs()
	args = append(args, "+quit")
	cmd := scmd.runCmd(args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		scmd.clearCmd(cmd)
		return err
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		scmd.clearCmd(cmd)
		return err
	}

	scanner := bufio.NewScanner(stdout)

	var text string
	for scanner.Scan() {
		text = scanner.Text()
		//fmt.Println(text)
		if err2 := scmd.parseStdOut(text); err2 != nil {
			err = errors.New(err2.Error())
			_ = scmd.kill()
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		_ = scmd.kill()
		return err
	}

	err = cmd.Wait()
	scmd.clearCmd(cmd)
	return err
}
