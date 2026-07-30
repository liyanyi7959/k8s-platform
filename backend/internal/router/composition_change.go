package router

import (
	changemysql "k8s-platform-backend/internal/change/adapters/mysql"
	changeapp "k8s-platform-backend/internal/change/application"
)

func buildChangeModule(d Deps, runtime moduleRuntime) changeModule {
	applicationService := changeapp.NewService(changemysql.NewRepository(d.DB))
	return changeModule{
		application: applicationService,
	}
}
