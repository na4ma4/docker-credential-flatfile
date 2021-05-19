package main

import (
	"encoding/json"
	"errors"
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
func (f Flatfile) Add(c *credentials.Credentials) (err error) {
	cs, lock, err := openFile()
	defer lock.Unlock() //nolint: errcheck

	if err != nil {
		return err
	}

	cs.Store[c.ServerURL] = *c

	if err = writeFile(cs); err != nil {
		return err
	}

	return err
}

// Delete removes credentials from the store.
func (f Flatfile) Delete(serverURL string) error {
	cs, lock, err := openFile()
	defer lock.Unlock() //nolint: errcheck

	if err != nil {
		return err
	}

	delete(cs.Store, serverURL)

	return writeFile(cs)
}

// ErrNotFound returned when credentials not found.
var ErrNotFound = errors.New("credentials not found")

// Get retrieves credentials from the store.
// It returns username and secret as strings.
func (f Flatfile) Get(serverURL string) (string, string, error) {
	cs, lock, err := openFile()
	defer lock.Unlock() //nolint: errcheck

	if err != nil {
		return "", "", err
	}

	if v, ok := cs.Store[serverURL]; ok {
		return v.Username, v.Secret, nil
	}

	return "", "", ErrNotFound
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

	err = ioutil.WriteFile(filename, data, 0600)
	if err != nil {
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
