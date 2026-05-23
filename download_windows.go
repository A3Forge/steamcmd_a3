//go:build windows

package steamcmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func (scmd *SteamCmd) downloadSteamCmd() error {
	scmd.ChangeStatus(STATUS_STEAMCMD_INSTALLING, "")

	// Get the data
	resp, err := http.Get("https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip")
	if err != nil {
		scmd.ChangeStatus(STATUS_STEAMCMD_ERROR_DOWNLOAD, fmt.Sprintf("Error download steamcmd: %v", err.Error()))
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create("steamcmd.zip")
	if err != nil {
		scmd.ChangeStatus(STATUS_STEAMCMD_CANT_OPEN_ZIP, fmt.Sprintf("Can't open steamcmd installer: %v", err.Error()))
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		scmd.ChangeStatus(STATUS_STEAMCMD_CANT_WRITE, fmt.Sprintf("Can't write steamcmd installer: %v", err.Error()))
		return err
	}

	err = Unzip("steamcmd.zip", scmd.steamCmdPath())
	if err != nil {
		scmd.ChangeStatus(STATUS_STEAMCMD_CANT_WRITE, fmt.Sprintf("Can't write steamcmd installer: %v", err.Error()))
		return err
	}

	out.Close()
	os.Remove("steamcmd.zip")

	scmd.ChangeStatus(STATUS_FINE, "")

	cmd := scmd.runCmd("+login", "anonymous", "+quit")
	err = cmd.Run()
	scmd.clearCmd(cmd)

	return err
}
