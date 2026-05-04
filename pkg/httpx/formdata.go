package httpx

import (
	"io"
	"path/filepath"
)

type FormField struct {
	Key   string
	Value string
}

type FormData struct {
	Fields []FormField
	files  []formFile
}

type formFile struct {
	key      string
	path     string
	filename string
	data     []byte
}

func NewFormData() *FormData {
	return &FormData{}
}

func (fd *FormData) Set(key, value string) *FormData {
	fd.Fields = append(fd.Fields, FormField{Key: key, Value: value})
	return fd
}

func (fd *FormData) SetFile(key, filePath string) *FormData {
	fd.files = append(fd.files, formFile{
		key:      key,
		path:     filePath,
		filename: filepath.Base(filePath),
	})
	return fd
}

func (fd *FormData) SetFileReader(key, filename string, r io.Reader) *FormData {
	data, err := io.ReadAll(r)
	if err != nil {
		return fd
	}
	fd.files = append(fd.files, formFile{
		key:      key,
		filename: filename,
		data:     data,
	})
	return fd
}

