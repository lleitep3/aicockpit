package hindsight

import (
	"errors"
	"fmt"
	"strings"
)

func redact(text string, secrets []string) string {
	if text == "" {
		return ""
	}
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		text = strings.ReplaceAll(text, secret, "[redacted]")
	}
	return text
}

// redactError returns an error whose text does not contain secrets.
// When redaction changes the text, the original error is not wrapped, so
// unwrapping cannot reveal the secret.
func redactError(err error, secrets []string) error {
	if err == nil {
		return nil
	}
	cleaned := redact(err.Error(), secrets)
	if cleaned == err.Error() {
		return err
	}
	return errors.New(cleaned)
}

func wrapOp(op string, err error, secrets []string) error {
	if err == nil {
		return nil
	}
	cleaned := redact(err.Error(), secrets)
	if cleaned != err.Error() {
		return fmt.Errorf("hindsight %s: %s", op, cleaned)
	}
	return fmt.Errorf("hindsight %s: %w", op, err)
}

func clearValues(values map[string]string) {
	for key := range values {
		values[key] = ""
		delete(values, key)
	}
}
