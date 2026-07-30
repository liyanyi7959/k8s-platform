package router

import (
	platformhttp "k8s-platform-backend/internal/platform/adapters/http"
	platformmysql "k8s-platform-backend/internal/platform/adapters/mysql"
	platformapp "k8s-platform-backend/internal/platform/application"
)

func buildPlatformModule(d Deps) platformModule {
	settingsService := platformapp.NewService(platformmysql.NewSettingsRepository(d.DB))
	return platformModule{
		settings: platformhttp.NewSettingsController(settingsService),
	}
}
