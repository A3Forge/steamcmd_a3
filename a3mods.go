package steamcmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type InstalledMod struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func (scmd *SteamCmd) modsDir() string {
	abs, err := filepath.Abs(filepath.Join(scmd.steamCmdPath(), "steamapps", "workshop", "content", ARMA_3_APPID))
	if err != nil {
		return filepath.Join(scmd.steamCmdPath(), "steamapps", "workshop", "content", ARMA_3_APPID)
	}
	return abs
}

func (scmd *SteamCmd) ModsDir() string {
	return scmd.modsDir()
}

func (scmd *SteamCmd) InstalledMods() ([]InstalledMod, error) {
	dir := scmd.modsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []InstalledMod{}, nil
		}
		return nil, err
	}

	mods := make([]InstalledMod, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			mods = append(mods, InstalledMod{
				ID:   e.Name(),
				Path: filepath.Join(dir, e.Name()),
			})
		}
	}
	return mods, nil
}

func (scmd *SteamCmd) ValidateMods(ids []string) error {
	validatedIDs := validatedModIds(ids)

	if len(validatedIDs) == 0 {
		return errors.New("Not found valid mod IDs")
	}

	if err := scmd.beginCmdOperation(); err != nil {
		return err
	}

	if err := scmd.ValidateSteamCMD(); err != nil {
		scmd.cmdMu.Unlock()
		return err
	}

	scmd.ChangeStatus(STATUS_STEAMCMD_MODS_VALIDATING, fmt.Sprintf("Mods : %s", strings.Join(validatedIDs, ", ")))

	go scmd.runVerifyModsLocked(validatedIDs)

	return nil
}

func validatedModIds(ids []string) []string {
	validatedMods := make([]string, 0, len(ids))

	for i, id := range ids {
		modID, err := strconv.ParseUint(id, 10, 64)
		if err == nil && modID > 0 {
			validatedMods = append(validatedMods, ids[i])
		}
	}

	return validatedMods
}

func (scmd *SteamCmd) runVerifyModsLocked(ids []string) (resultMessage string) {
	defer scmd.cmdMu.Unlock()

	defer func() {
		scmd.ChangeStatus(resultMessage, "")
	}()

	resultMessage = STATUS_FINE

	for i := range ids {
		modID := ids[i]
		args := make([]string, 0, 10)
		args = append(args, scmd.credentialsArgs()...)
		args = append(args, "+workshop_download_item", ARMA_3_APPID, modID, "+quit")

		cmd := scmd.runCmd(args...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			resultMessage = STATUS_STEAMCMD_MODS_VALIDATING_ERR
			scmd.clearCmd(cmd)
			return
		}

		if err := cmd.Start(); err != nil {
			_ = stdout.Close()
			resultMessage = STATUS_STEAMCMD_MODS_VALIDATING_ERR
			scmd.clearCmd(cmd)
			return
		}

		scanner := bufio.NewScanner(stdout)

		var text string
		for scanner.Scan() {
			text = scanner.Text()
			fmt.Println(text)
			if err := scmd.parseStdOut(text); err != nil {
				resultMessage = err.Error()
				_ = stdout.Close()
				_ = scmd.kill()
				return
			}
		}

		if err := scanner.Err(); err != nil {
			_ = stdout.Close()
			resultMessage = STATUS_STEAMCMD_MODS_VALIDATING_ERR
			_ = scmd.kill()
			return
		}

		if err := stdout.Close(); err != nil {
			resultMessage = STATUS_STEAMCMD_MODS_VALIDATING_ERR
			_ = scmd.kill()
			return
		}

		if err := cmd.Wait(); err != nil {
			resultMessage = STATUS_STEAMCMD_MODS_VALIDATING_ERR
			scmd.clearCmd(cmd)
			return
		}

		scmd.clearCmd(cmd)
	}

	return
}
