package storage

import "os"

func SaveCRLToFile(filename string, crlData []byte) error {
	err := os.WriteFile(filename, crlData, 0664)
	if err != nil {
		return err
	}
	return nil
}
