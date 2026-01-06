package storage

import "os"

func SaveCRLToFile(filename string, crlData []byte) error {
	err := os.WriteFile(filename, crlData, 0644)
	if err != nil {
		return err
	}
	return nil
}
