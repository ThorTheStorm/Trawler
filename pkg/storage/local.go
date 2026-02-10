package storage

import (
	"fmt"
	"os"
)

func ValidateLocalStoragePaths(paths ...string) error {
	for _, path := range paths {
		if path == "" {
			return fmt.Errorf("storage path is not set in configuration")
		}
		if CheckIfFolderExists(path) {
			continue
		} else {
			return fmt.Errorf("storage path does not exist: %s", path)
		}
	}
	return nil
}

func SaveCRLToFile(filename string, crlData []byte) error {
	err := os.WriteFile(filename, crlData, 0644)
	if err != nil {
		return err
	}
	return nil
}

func CheckIfFolderExists(folderPath string) bool {
	info, err := os.Stat(folderPath)
	if os.IsNotExist(err) {
		return false
	} else if err != nil {
		return false
	}

	return info.IsDir()
}

func CreateFolderIfNotExists(folderPath string) error {
	if !CheckIfFolderExists(folderPath) {
		err := os.MkdirAll(folderPath, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func CopyFile(srcPath, dstPath string) error {
	input, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	err = os.WriteFile(dstPath, input, 0644)
	if err != nil {
		return err
	}
	return nil
}

func CopyFolder(srcPath, dstPath string) error {
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dstPath, 0755)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcEntryPath := fmt.Sprintf("%s/%s", srcPath, entry.Name())
		dstEntryPath := fmt.Sprintf("%s/%s", dstPath, entry.Name())

		if entry.IsDir() {
			err = CopyFolder(srcEntryPath, dstEntryPath)
			if err != nil {
				return err
			}
		} else {
			err = CopyFile(srcEntryPath, dstEntryPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
