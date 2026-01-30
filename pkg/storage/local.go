package storage

import "os"

func SaveCRLToFile(filename string, crlData []byte) error {
	err := os.WriteFile(filename, crlData, 0664)
	if err != nil {
		return err
	}
	return nil
}

func CheckIfFolderExists(folderPath string) bool {
	info, err := os.Stat(folderPath)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func CreateFolderIfNotExists(folderPath string) error {
	if !CheckIfFolderExists(folderPath) {
		err := os.MkdirAll(folderPath, 0664)
		if err != nil {
			return err
		}
	}
	return nil
}
