package path

import "os"

func Exists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	return err
}
