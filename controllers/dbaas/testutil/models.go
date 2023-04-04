package testutil

import (
	"fmt"
)

type SqlUser struct {
	Name     string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
}

type credential struct {
	CredentialField1 string
	CredentialField2 string
}

type APIErrorMessage struct {
	Code     int
	Message  string
	HttpCode int
}

func (e *APIErrorMessage) String() string {
	return fmt.Sprintf("%v-%v", e.Code, e.Message)
}
