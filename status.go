package steamcmd

func (scmd *SteamCmd) Status() SteamCmdStatus {
	scmd.mu.RLock()
	defer scmd.mu.RUnlock()

	return SteamCmdStatus{
		Status:        scmd.state.Status,
		StatusDetails: scmd.state.StatusDetails,
		Progress:      scmd.state.Progress,
	}
}

func (scmd *SteamCmd) ChangeStatus(status, details string) {
	scmd.mu.Lock()
	scmd.state.Status = status
	scmd.state.StatusDetails = details
	scmd.state.Progress = 0
	state := scmd.state
	cb := scmd.OnStatusChange
	scmd.mu.Unlock()

	if cb != nil {
		cb(state)
	}
}

func (scmd *SteamCmd) ChangeProgress(progress float64) {
	scmd.mu.Lock()
	scmd.state.Progress = progress
	state := scmd.state
	cb := scmd.OnStatusChange
	scmd.mu.Unlock()

	if cb != nil {
		cb(state)
	}
}
