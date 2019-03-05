package commons

import (
	"io"
	"os"
)

func CopyFile(src, dest string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer Close(source)

	destination, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer Close(destination)
	_, err = io.Copy(destination, source)
	return err
}