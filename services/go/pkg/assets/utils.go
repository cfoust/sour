package assets

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func FileExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

func WriteBytes(data []byte, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	out, err = os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func DownloadFile(url string, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
