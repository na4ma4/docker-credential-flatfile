package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"time"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/juju/fslock"
)

const (
	lockTimeout        time.Duration = 10 * time.Second
	credentialFileMode fs.FileMode   = 0o0600
)

// Flatfile implements the credentials.Helper interface.
type Flatfile struct{}

// Add appends credentials to the store.
func (f Flatfile) Add(creds *credentials.Credentials) error {
	if creds == nil {
		return credentials.NewErrCredentialsMissingUsername()
	}

	credStore, lock, err := openFile()
	defer lock.Unlock() //nolint:errcheck // ignore errors on Unlock.

	if err != nil {
		return err
	}

	credStore.Store[creds.ServerURL] = *creds

	return writeFile(credStore)
}

// Delete removes credentials from the store.
func (f Flatfile) Delete(serverURL string) error {
	if serverURL == "" {
		return credentials.NewErrCredentialsMissingServerURL()
	}

	credStore, lock, err := openFile()
	defer lock.Unlock() //nolint:errcheck // ignore errors on Unlock.

	if err != nil {
		return err
	}

	delete(credStore.Store, serverURL)

	return writeFile(credStore)
}

// Get retrieves credentials from the store.
// It returns username and secret as strings.
func (f Flatfile) Get(serverURL string) (string, string, error) {
	if serverURL == "" {
		return "", "", credentials.NewErrCredentialsMissingServerURL()
	}

	credStore, lock, err := openFile()
	defer lock.Unlock() //nolint:errcheck // ignore errors on Unlock.

	if err != nil {
		return "", "", err
	}

	if v, ok := credStore.Store[serverURL]; ok {
		return v.Username, v.Secret, nil
	}

	return "", "", credentials.NewErrCredentialsNotFound()
}

// List returns the stored serverURLs and their associated usernames.
func (f Flatfile) List() (map[string]string, error) {
	credStore, lock, err := openFile()
	defer lock.Unlock() //nolint:errcheck // ignore errors on Unlock.

	if err != nil {
		return map[string]string{}, err
	}

	o := map[string]string{}
	for k, v := range credStore.Store {
		o[k] = v.Username
	}

	return o, nil
}

type credStore struct {
	Store map[string]credentials.Credentials `json:"store"`
}

func getFlatFile() string {
	return fmt.Sprintf("%s%s%s", userHomeDir(), string(os.PathSeparator), ".creds.json")
}

func openFile() (*credStore, *fslock.Lock, error) {
	filename := getFlatFile()
	lock := fslock.New(filename)

	if err := lock.LockWithTimeout(lockTimeout); err != nil {
		return nil, lock, fmt.Errorf("unable to lock file: %w", err)
	}

	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, lock, fmt.Errorf("unable to read file: %w", err)
	}

	cs := &credStore{
		Store: make(map[string]credentials.Credentials),
	}
	_ = json.Unmarshal(file, cs)

	return cs, lock, nil
}

func writeFile(cs *credStore) error {
	filename := getFlatFile()

	data, dataErr := json.Marshal(cs)
	if dataErr != nil {
		return fmt.Errorf("unable to marshal credentials: %w", dataErr)
	}

	if err := os.WriteFile(filename, data, credentialFileMode); err != nil {
		return fmt.Errorf("unable to write file: %w", err)
	}

	return nil
}

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}

		return home
	}

	return os.Getenv("HOME")
}
