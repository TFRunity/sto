package config

import "log/slog"

const redacted = "[REDACTED]"

type Secret string

func (Secret) LogValue() slog.Value {
	return slog.StringValue(redacted)
}

func (Secret) String() string {
	return redacted
}

func (Secret) GoString() string {
	return redacted
}

func (Secret) MarshalText() ([]byte, error) {
	return []byte(redacted), nil
}

func (s Secret) Reveal() string {
	return string(s)
}
