package supportbundle

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
)

func readLogFiles(path, namespace, name, container string) (io.Reader, error) {
	abs, err := filepath.Abs(filepath.Join(path, "logs", namespace, name, container+".log"))
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(content), nil
}
