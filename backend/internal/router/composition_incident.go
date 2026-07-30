package router

import (
	incidenthttp "k8s-platform-backend/internal/incident/adapters/http"
	incidentmysql "k8s-platform-backend/internal/incident/adapters/mysql"
	incidentapp "k8s-platform-backend/internal/incident/application"
)

func buildIncidentModule(d Deps) incidentModule {
	repository := incidentmysql.NewRepository(d.DB)
	applicationService := incidentapp.NewService(repository, repository)
	monitoringService := incidentapp.NewMonitoringService(repository)
	return incidentModule{
		legacy: incidenthttp.NewLegacyController(monitoringService, applicationService),
		v2:     incidenthttp.NewController(applicationService),
	}
}
