package landlock

import (
	"os"
	"strings"

	"github.com/landlock-lsm/go-landlock/landlock"
)

// Landlock takes in files or directories that we should be allowed to read and/or write.
func Landlock(readOnlys []string, readWrites []string) error {
	// Check if all the entries exist, and divide them into files and dirs

	filteredRoDirs := []string{}
	filteredRoFiles := []string{}
	for _, entry := range readOnlys {
		fs, statErr := os.Stat(entry)
		if statErr == nil {
			if fs.IsDir() {
				filteredRoDirs = append(filteredRoDirs, entry)
			} else {
				filteredRoFiles = append(filteredRoFiles, entry)
			}
		}
	}

	// Filter rwdirs, if stat fails, remove it from the array
	filteredRwDirs := []string{}
	filteredRwFiles := []string{}
	for _, entry := range readWrites {
		fs, statErr := os.Stat(entry)
		if statErr == nil {
			if fs.IsDir() {
				filteredRwDirs = append(filteredRwDirs, entry)
			} else {
				filteredRwFiles = append(filteredRwFiles, entry)
			}
		}
	}

	err := landlock.V10.BestEffort().RestrictPaths(
		landlock.RODirs(filteredRoDirs...),
		landlock.RWDirs(filteredRwDirs...),
		landlock.ROFiles(filteredRoFiles...),
		landlock.RWFiles(filteredRwFiles...),

		landlock.RWDirs(os.TempDir()),
	)
	return err
}

// Return all the paths in the PATH environment variable
func PathToDirs() []string {
	pathVar := os.Getenv("PATH")
	parts := strings.Split(pathVar, ":")

	result := make([]string, 0, len(parts))

	for _, p := range parts {
		result = append(result, p)
	}

	return result
}

func RequiredRODirs() []string {
	return []string{
		"/etc/resolv.conf",
		"/etc/hosts",
		"/etc/ssl/certs",
		"/usr/share/zoneinfo",
		"/etc/localtime",
		"/dev/null",
		"/usr/lib",
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/pki/tls/certs/ca-bundle.crt",
		"/etc/ssl/ca-bundle.pem",
		"/etc/pki/tls/cacert.pem",
		"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem",
		"/etc/ssl/cert.pem",
	}
}
