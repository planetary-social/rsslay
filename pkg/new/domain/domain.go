package domain

import "errors"

type Secret struct {
	s string
}

func NewSecret(s string) (Secret, error) {
	if s == "" {
		return Secret{}, errors.New("secret can't be an empty string")
	}
	return Secret{s: s}, nil
}

func (s Secret) String() string {
	return s.s
}
