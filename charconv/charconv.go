package charconv

import (
	"bytes"
	"io/ioutil"

	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// Convert detects char encoding and converts data as result and encname.
func Convert(data []byte) (result, encname string, err error) {
	if len(data) == 0 {
		return "", "", nil
	}

	det := chardet.NewTextDetector()
	detresult, err := det.DetectBest(data)
	if err != nil {
		return "", "", err
	}

	switch detresult.Charset {
	case "Shift_JIS":
		decoded, err := ioutil.ReadAll(transform.NewReader(bytes.NewBuffer(data), japanese.ShiftJIS.NewDecoder()))
		if err != nil {
			return "", "", err
		}
		return string(decoded), detresult.Charset, nil
	case "UTF-8":
		return string(data), detresult.Charset, nil
	default:
		return string(data), "", nil
	}
}
