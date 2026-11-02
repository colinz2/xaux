package path

import "os"

func Exists(path string) error {
	_, err := os.Stat(path)
	return err
}
