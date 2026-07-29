package ports

import "context"

type SettingsRepository interface {
	Load(context.Context) (map[string]string, error)
	Save(context.Context, map[string]string) error
}
