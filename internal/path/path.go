// Package path provides file path utilities.
package path

import "github.com/mitchellh/go-homedir"

func GetUserHome() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return homeDir, nil
}

func ExpandUserHome(path string) (string, error) {
	expandedPath, err := homedir.Expand(path)
	if err != nil {
		return "", err
	}
	return expandedPath, nil
}
