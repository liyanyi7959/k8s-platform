package router

import (
	audithttp "k8s-platform-backend/internal/audit/adapters/http"
	auditmysql "k8s-platform-backend/internal/audit/adapters/mysql"
	auditapp "k8s-platform-backend/internal/audit/application"
)

func buildAuditModule(d Deps) auditModule {
	auditService := auditapp.NewService(auditmysql.NewRepository(d.DB))
	return auditModule{service: auditService, controller: audithttp.NewController(auditService)}
}
