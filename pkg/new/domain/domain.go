package domain

import "errors"

type Secret struct {
	s string
}

func NewSecret(s string) (Secret, error) {
	if s == "" {
		return Secret{}, errors.New("secret can't be an empty string")
	}

	// todo len check, is this the same length as nostr key?

	return Secret{s: s}, nil
}

func (s Secret) String() string {
	return s.s
}
