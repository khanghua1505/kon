package shell

import (
	"os"
)

// promptPlaceholder is replaced by the kubie prompt in the rc files and
// scripts of the child shells.
const promptPlaceholder = "%%PROMPT%%"

// writeTempFile writes content to a new temporary file named prefix*suffix and
// returns its path. The caller removes the file.
func writeTempFile(prefix, suffix, content string) (string, error) {
	file, err := os.CreateTemp("", prefix+"*"+suffix)
	if err != nil {
		return "", err
	}

	path := file.Name()
	if _, err := file.WriteString(content); err != nil {
		file.Close()
		os.Remove(path)
		return "", err
	}

	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", err
	}

	return path, nil
}

// writeTempDir creates a new temporary directory and returns its path. The
// caller removes the directory.
func writeTempDir(prefix string) (string, error) {
	return os.MkdirTemp("", prefix+"*")
}
