package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/juju/fslock"
)

const lockTimeout = 10 * time.Second

// Flatfile implements the credentials.Helper interface.
type Flatfile struct{}

// Add appends credentials to the store.
func (f Flatfile) Add(creds *credentials.Credentials) (err error) {
	if creds == nil {
		return credentials.NewErrCredentialsMissingUsername()
	}

	cs, lock, err := openFile()
	defer lock.Unlock() //nolint: errcheck

	if err != nil {
		return err
	}

	cs.Store[creds.ServerURL] = *creds

	if err = writeFile(cs); err != nil {
		return err
	}

	return err
}

// Delete removes credentials from the store.
func (f Flatfile) Delete(serverURL string) error {
	if serverURL == "" {
		return credentials.NewErrCredentialsMissingServerURL()
	}

	cs, lock, err := openFile()
	defer lock.Unlock() //nolint: errcheck

	if err != nil {
		return err
	}

	delete(cs.Store, serverURL)

	return writeFile(cs)
}

// Get retrieves credentials from the store.
// It returns username and secret as strings.
func (f Flatfile) Get(serverURL string) (string, string, error) {
	if serverURL == "" {
		return "", "", credentials.NewErrCredentialsMissingServerURL()
	}

	cs, lock, err := openFile()
	defer lock.Unlock() //nolint: errcheck

	if err != nil {
		return "", "", err
	}

	if v, ok := cs.Store[serverURL]; ok {
		return v.Username, v.Secret, nil
	}

	return "", "", credentials.NewErrCredentialsNotFound()
}

// List returns the stored serverURLs and their associated usernames.
func (f Flatfile) List() (map[string]string, error) {
	cs, lock, err := openFile()
	defer lock.Unlock() //nolint: errcheck

	if err != nil {
		return map[string]string{}, err
	}

	o := map[string]string{}
	for k, v := range cs.Store {
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

	file, err := ioutil.ReadFile(filename)
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

	data, err := json.Marshal(cs)
	if err != nil {
		return fmt.Errorf("unable to marshal credentials: %w", err)
	}

	if err := ioutil.WriteFile(filename, data, 0600); err != nil {
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
