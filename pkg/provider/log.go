package supportbundle

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func readLogFiles(path, namespace, name string) (io.Reader, error) {
	var contents [][]byte
	abs, err := filepath.Abs(filepath.Join(path, "logs", namespace, name))
	if err != nil {
		return nil, err
	}
	err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		if !info.IsDir() && info.Size() != 0 {
			content, err := ioutil.ReadFile(info.Name())
			if err != nil {
				return err
			}
			contents = append(contents, content)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return bytes.NewReader(bytes.Join(contents, nil)), nil
}
