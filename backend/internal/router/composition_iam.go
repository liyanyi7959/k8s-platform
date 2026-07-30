package router

import (
	iamhttp "k8s-platform-backend/internal/iam/adapters/http"
	iammysql "k8s-platform-backend/internal/iam/adapters/mysql"
	iamapp "k8s-platform-backend/internal/iam/application"
)

func buildIAMModule(d Deps) iamModule {
	repository := iammysql.NewAuthRepository(d.DB)
	authService := d.IAMAuthService
	if authService == nil {
		authService = iamapp.NewAuthService(repository, iammysql.BcryptHasher{}, d.CacheStore, d.CacheTTL)
	}
	users := iamapp.NewUserManagement(repository, iammysql.BcryptHasher{}, authService)
	roles := iamapp.NewRoleManagement(repository, authService)
	return iamModule{users: iamhttp.NewController(users, roles)}
}
