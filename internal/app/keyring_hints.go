package app

import (
	"errors"
	"strings"

	"github.com/svaruag/mocli/internal/secrets"
)

func secretsBackendHint(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, secrets.ErrFileBackendPasswordRequired) {
		return strings.TrimSpace(`File keyring backend is active, but MO_KEYRING_PASSWORD is not set.
Set a password and retry:
  export MO_KEYRING_BACKEND=file
  export MO_KEYRING_PASSWORD='choose-a-strong-password'

If your system keychain is available, you can also use:
  export MO_KEYRING_BACKEND=auto`)
	}
	return err.Error()
}
