package steamcmd

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type CredentialsStore struct {
	ActiveUserID int                     `json:"ActiveUserID"`
	NextUserID   int                     `json:"NextUserID"`
	AllUsers     map[int]CredentialsUser `json:"AllUsers"`
}

type CredentialsUser struct {
	UserID   int    `json:"UserID"`
	Username string `json:"Username"`
	Password string `json:"Password"`
}

func (scmd *SteamCmd) credentialsArgs() []string {
	scmd.mu.RLock()
	defer scmd.mu.RUnlock()

	activeUser := CredentialsUser{}
	for _, user := range scmd.CredentialsStore.AllUsers {
		if user.UserID == scmd.CredentialsStore.ActiveUserID {
			activeUser = user
			break
		}
	}

	args := []string{}

	if scmd.SGuardCode != "" {
		args = append(args, "+set_steam_guard_code", scmd.SGuardCode)
	}

	args = append(args, "+login", activeUser.Username, activeUser.Password)

	return args
}

func (scmd *SteamCmd) credentialsConfigFile() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		panic("LOCALAPPDATA not set")
	}

	return filepath.Join(localAppData, "steamcmd_wr", "credentials.json")
}

func (scmd *SteamCmd) saveOrResetCredentials() error {
	path := scmd.credentialsConfigFile()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(&scmd.CredentialsStore)
}

func (scmd *SteamCmd) loadCredentials() {
	data, err := os.ReadFile(scmd.credentialsConfigFile())
	if err != nil {
		scmd.saveOrResetCredentials()
		return
	}

	if err := json.Unmarshal(data, &scmd.CredentialsStore); err != nil {
		scmd.saveOrResetCredentials()
		return
	}
}
