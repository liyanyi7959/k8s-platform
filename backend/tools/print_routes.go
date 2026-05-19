package main

import (
	"context"
	"fmt"
	"time"

	"k8s-platform-backend/internal/auth"
	"k8s-platform-backend/internal/config"
	"k8s-platform-backend/internal/controller"
	"k8s-platform-backend/internal/db"
	"k8s-platform-backend/internal/router"
	"k8s-platform-backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	gdb, sdb, err := db.Open(db.Config{MySQLDSN: cfg.DB.MySQLDSN})
	if err != nil {
		panic(err)
	}
	defer sdb.Close()

	if cfg.Migrate.Auto {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if err := db.Migrate(ctx, gdb); err != nil {
			panic(err)
		}
	}

	jwtMgr := auth.NewManager(cfg.JWT.Secret)
	cacheStore, err := service.NewRedisCacheStore(cfg.Redis)
	if err != nil {
		panic(err)
	}
	defer cacheStore.Close()

	rbacSvc := service.NewRbacService(gdb, cacheStore, 15*time.Minute)
	authCtl := controller.NewAuthController(jwtMgr, rbacSvc, cfg.ParsedTokenTTL())
	rbacCtl := controller.NewRbacController(rbacSvc)

	r, err := router.New(router.Deps{
		DB:             gdb,
		JWTMgr:         jwtMgr,
		AuthCtl:        authCtl,
		RbacCtl:        rbacCtl,
		RbacSvc:        rbacSvc,
		EncryptionKey:  cfg.EncryptionKey(),
		OSSCfg:         cfg.OSS,
		CacheStore:     cacheStore,
		CacheTTL:       cfg.ParsedRedisDefaultTTL(),
		K8sInsecureTLS: cfg.K8s.InsecureSkipTLSVerify,
	})
	if err != nil {
		panic(err)
	}

	for _, rt := range r.Routes() {
		if rt.Path == "/api/v1/automation/jobs" || rt.Path == "/api/v1/automation/jobs/:id" || rt.Path == "/api/v1/automation/jobs/:id/run" {
			fmt.Printf("%s %s -> %s\n", rt.Method, rt.Path, rt.Handler)
		}
	}
}
