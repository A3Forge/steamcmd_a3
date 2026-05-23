package steamcmd

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

const (
	STATUS_FINE                         = ""
	STATUS_STEAMGUARD_CODE              = "STEAM_GUARD_CODE"
	STATUS_INIT_ERR                     = "STEAMCMD_INIT_ERR"
	STATUS_INVALID_USER                 = "INVALID_USER"
	STATUS_STEAMCMD_INSTALLING          = "STEAMCMD_INSTALLING"
	STATUS_STEAMCMD_ERROR_DOWNLOAD      = "ERROR_DOWNLOAD_STEAMCMD"
	STATUS_STEAMCMD_CANT_OPEN_ZIP       = "CANT_OPEN_INSTALLER"
	STATUS_STEAMCMD_CANT_WRITE          = "CANT_WRITE_INSTALLER"
	STATUS_STEAMCMD_RATELIMIT           = "RATE_LIMIT"
	STATUS_STEAMCMD_INTALLING_A3SERVER  = "A3SERVER_INSTALLING"
	STATUS_STEAMCMD_MODS_VALIDATING     = "VALIDATE_MODS"
	STATUS_STEAMCMD_MODS_VALIDATING_ERR = "VALIDATE_MODS_ERROR"
	STATUS_STEAMCMD_MISSING_PARAMS      = "MISSING_PARAMS"
)

var (
	ErrSteamCmdAlreadyActive = errors.New("SteamCmd already active")
)

type SteamCmdStatus struct {
	Status        string  `json:"status"`
	StatusDetails string  `json:"statusDetails"`
	Progress      float64 `json:"progress"`
}

type SteamCmd struct {
	SteamCmdDir      string
	OnStatusChange   func(SteamCmdStatus)
	state            SteamCmdStatus
	CredentialsStore CredentialsStore
	SGuardCode       string
	mu               sync.RWMutex
	cmdMu            sync.Mutex
	cmd              *exec.Cmd
}

func NewSteamCmd() *SteamCmd {
	newSteamCmd := &SteamCmd{
		mu:    sync.RWMutex{},
		state: SteamCmdStatus{},
		CredentialsStore: CredentialsStore{
			AllUsers: make(map[int]CredentialsUser),
		},
		cmd: nil,
	}

	newSteamCmd.loadCredentials()

	return newSteamCmd
}

func (scmd *SteamCmd) kill() error {
	scmd.mu.Lock()
	cmd := scmd.cmd
	scmd.cmd = nil
	scmd.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	return nil
}

func (scmd *SteamCmd) beginCmdOperation() error {
	if !scmd.cmdMu.TryLock() {
		return ErrSteamCmdAlreadyActive
	}

	if scmd.IsActive() {
		scmd.cmdMu.Unlock()
		return ErrSteamCmdAlreadyActive
	}

	return nil
}

func (scmd *SteamCmd) clearCmd(cmd *exec.Cmd) {
	scmd.mu.Lock()
	defer scmd.mu.Unlock()

	if scmd.cmd == cmd {
		scmd.cmd = nil
	}
}

func (scmd *SteamCmd) IsActive() bool {
	scmd.mu.RLock()
	defer scmd.mu.RUnlock()
	return scmd.cmd != nil && scmd.cmd.Process != nil
}

func (scmd *SteamCmd) ValidateSteamCMD() error {
	_, err := os.Stat(scmd.steamCmdExePath())
	if err != nil {
		err = scmd.downloadSteamCmd()
	}

	if err != nil {
		return err
	}

	return nil
}

func (scmd *SteamCmd) Close() {
	scmd.kill()
}

func (scmd *SteamCmd) CloseOnInterrupt() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sig
		scmd.Close()
		//os.Exit(0)
	}()
}
