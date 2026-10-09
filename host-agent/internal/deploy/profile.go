package deploy

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

const maxProfileSize = 64 << 10

// LoadSpec reads one bounded, strictly decoded deployment profile.
func LoadSpec(path string) (Spec, error) {
	file, err := os.Open(path)
	if err != nil {
		return Spec{}, errors.New("deployment profile unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxProfileSize {
		return Spec{}, errors.New("deployment profile must be a regular file of at most 64 KiB")
	}
	var spec Spec
	decoder := json.NewDecoder(io.LimitReader(file, maxProfileSize+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return Spec{}, errors.New("invalid deployment profile")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Spec{}, errors.New("deployment profile must contain one JSON object")
	}
	if err := spec.Validate(); err != nil {
		return Spec{}, err
	}
	return spec, nil
}
