// db 封装数据库连接与迁移相关的基础能力。
//
// 该文件负责：
// - 根据 DSN 打开 MySQL 连接（GORM + database/sql）
// - 配置连接池参数（最大连接数、空闲连接数、连接生命周期）
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	// MySQLDSN 为 MySQL 连接串，例如：
	//   user:pass@tcp(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local
	MySQLDSN string
}

// LoadConfigFromEnv 从环境变量加载数据库配置（便于本地/容器运行）。
func LoadConfigFromEnv() Config {
	return Config{
		MySQLDSN: firstNonEmpty(os.Getenv("MYSQL_DSN"), os.Getenv("DB_DSN")),
	}
}

// Open 打开数据库连接并返回：
// - *gorm.DB：业务层主要使用的 ORM 入口
// - *sql.DB：用于设置连接池参数/健康检查等底层能力
func Open(cfg Config) (*gorm.DB, *sql.DB, error) {
	dsn := strings.TrimSpace(cfg.MySQLDSN)
	if dsn == "" {
		return nil, nil, errors.New("MYSQL_DSN is required")
	}
	if err := ensureDatabase(dsn); err != nil {
		return nil, nil, err
	}
	gdb, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, nil, err
	}
	sdb, err := gdb.DB()
	if err != nil {
		return nil, nil, err
	}
	sdb.SetMaxOpenConns(25)
	sdb.SetMaxIdleConns(10)
	sdb.SetConnMaxLifetime(30 * time.Minute)
	return gdb, sdb, nil
}

func ensureDatabase(dsn string) error {
	parsed, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return err
	}
	dbName := strings.TrimSpace(parsed.DBName)
	if dbName == "" {
		return nil
	}

	parsed.DBName = ""
	serverDB, err := sql.Open("mysql", parsed.FormatDSN())
	if err != nil {
		return err
	}
	defer func() { _ = serverDB.Close() }()

	if _, err := serverDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", quoteIdentifier(dbName))); err != nil {
		return err
	}
	return nil
}

func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
