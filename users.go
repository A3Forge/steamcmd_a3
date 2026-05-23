package steamcmd

import (
	"errors"
	"fmt"
	"slices"
)

var (
	ErrCredentialsIsEmpty = errors.New("Please add steam account for using this service")
)

func (scmd *SteamCmd) AddUser(u, p string) (int, error) {
	if scmd.userExistsByName(u) {
		return 0, fmt.Errorf("Account with username %s already exists", u)
	}

	scmd.mu.Lock()
	defer scmd.mu.Unlock()

	nextID := scmd.CredentialsStore.NextUserID

	for {
		_, ok := scmd.CredentialsStore.AllUsers[nextID]
		if !ok {
			break
		} else {
			nextID++
		}
	}

	scmd.CredentialsStore.AllUsers[nextID] = CredentialsUser{
		UserID:   nextID,
		Username: u,
		Password: p,
	}

	scmd.CredentialsStore.NextUserID++

	scmd.saveOrResetCredentials()

	return nextID, nil
}

func (scmd *SteamCmd) DeleteUser(id int) error {
	if !scmd.userExistsByID(id) {
		return fmt.Errorf("Account with id %d is not exists", id)
	}

	scmd.mu.Lock()
	defer scmd.mu.Unlock()

	userIndex := -1

	for forId := range scmd.CredentialsStore.AllUsers {
		if forId == id {
			userIndex = forId
			break
		}
	}

	delete(scmd.CredentialsStore.AllUsers, userIndex)

	scmd.saveOrResetCredentials()

	return nil
}

func (scmd *SteamCmd) userExistsByName(u string) bool {
	scmd.mu.RLock()
	defer scmd.mu.RUnlock()

	for _, v := range scmd.CredentialsStore.AllUsers {
		if v.Username == u {
			return true
		}
	}

	return false
}

func (scmd *SteamCmd) userExistsByID(id int) bool {
	scmd.mu.RLock()
	defer scmd.mu.RUnlock()

	_, ok := scmd.CredentialsStore.AllUsers[id]
	if !ok {
		return false
	}

	return true
}

func (scmd *SteamCmd) AllUsers() []CredentialsUser {
	scmd.mu.RLock()
	defer scmd.mu.RUnlock()

	users := make([]CredentialsUser, 0, len(scmd.CredentialsStore.AllUsers))

	for _, v := range scmd.CredentialsStore.AllUsers {
		users = append(users, v)
	}

	slices.SortFunc(users, func(i, v CredentialsUser) int {
		return i.UserID - v.UserID
	})

	return users
}

func (scmd *SteamCmd) GetActiveUser() CredentialsUser {
	scmd.mu.RLock()
	defer scmd.mu.RUnlock()

	store := scmd.CredentialsStore

	for _, v := range store.AllUsers {
		if v.UserID == store.ActiveUserID {
			return v
		}
	}

	return CredentialsUser{}
}
