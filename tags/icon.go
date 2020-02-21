package tags

import (
	"encoding/base64"
	"io/ioutil"
	"log"
	"mime"
	"path/filepath"
	"strings"
)

// IconData can be either a base64-encoded string or an URL.
type IconData struct {
	URL string

	FileExt string
	Buf     []byte
}

func DecodeIconData(data string) IconData {
	var icon IconData
	if strings.HasPrefix(data, "http") {
		icon.URL = data
		return icon
	}

	data = strings.TrimLeft(data, "data:")
	comps := strings.SplitN(data, ";", 2)

	ext, err := mime.ExtensionsByType(comps[0])
	if ext != nil && err == nil {
		icon.FileExt = ext[0]
	}

	encoding := strings.Split(comps[1], ",")[0]
	if encoding == "base64" {
		comps[1] = comps[1][7:]
	} else {
		log.Printf("Don't know how to decode value: %v", encoding)
		return icon
	}
	icon.Buf, err = base64.StdEncoding.DecodeString(comps[1])
	if err != nil {
		log.Printf("Could not decode base64 data: %v", err)
	}
	return icon
}

func (i *IconData) WriteToDisk(path string, filename string, baseURL string) (url string, err error) {
	if i.URL != "" {
		return i.URL, nil
	}
	url = baseURL + filename + i.FileExt
	return url, ioutil.WriteFile(filepath.Join(path, filename+i.FileExt), i.Buf, 0777)
}
