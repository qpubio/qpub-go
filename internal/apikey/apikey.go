package apikey

import "errors"

var ErrInvalidFormat = errors.New("Invalid API key credential format")

// Parsed credential parts.
type Parsed struct {
	PublicID string
	Secret   string
}

// Parse splits publicId:secret (single colon separator).
func Parse(credential string) (Parsed, error) {
	sep := -1
	for i := 0; i < len(credential); i++ {
		if credential[i] == ':' {
			if sep >= 0 {
				return Parsed{}, ErrInvalidFormat
			}
			sep = i
		}
	}
	if sep <= 0 || sep == len(credential)-1 {
		return Parsed{}, ErrInvalidFormat
	}
	return Parsed{
		PublicID: credential[:sep],
		Secret:   credential[sep+1:],
	}, nil
}
