package db

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/db/migrations"
)

// Migrate 执行数据库迁移（SQL 文件驱动）。
//
// 迁移文件来源：internal/db/migrations/*.sql（通过 embed 打包到二进制中）。
// 执行策略：
// - 按文件名排序后依次执行（因此文件名通常用 001_xxx.sql 这种可排序前缀）
// - 每个文件拆分为多条 SQL 语句，放在一个事务中执行
// - 执行成功后把文件名写入 schema_migrations 表，确保幂等
//
// 注意：
//   - splitSQLStatements 是“轻量拆分”，不支持复杂的 SQL 分隔（例如存储过程内部的分号）。
//     当前项目迁移 SQL 以建表/索引为主，满足需求。
func Migrate(ctx context.Context, gdb *gorm.DB) error {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return err
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	// 初始化迁移记录表（不存在则创建）。
	if err := ensureMigrationsTable(ctx, gdb); err != nil {
		return err
	}

	// 已执行迁移版本：key 为文件名。
	applied, err := listAppliedMigrations(ctx, gdb)
	if err != nil {
		return err
	}

	for _, filename := range files {
		// 幂等：已执行过的 migration 跳过。
		if applied[filename] {
			continue
		}
		raw, err := migrations.FS.ReadFile(filename)
		if err != nil {
			return err
		}
		stmts := splitSQLStatements(raw)
		if len(stmts) == 0 {
			// 空文件也记录为已执行，避免下次重复读取。
			if err := markMigrationApplied(ctx, gdb, filename); err != nil {
				return err
			}
			continue
		}

		// 一个文件在一个事务里执行：中途失败会回滚，避免半成品 schema。
		if err := gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, s := range stmts {
				if err := tx.Exec(s).Error; err != nil {
					return fmt.Errorf("apply migration %s failed: %w", filename, err)
				}
			}
			return markMigrationApplied(ctx, tx, filename)
		}); err != nil {
			return err
		}
	}

	return nil
}

// ensureMigrationsTable 创建 schema_migrations 表，用于记录已执行的迁移版本。
func ensureMigrationsTable(ctx context.Context, gdb *gorm.DB) error {
	return gdb.WithContext(ctx).Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version VARCHAR(64) NOT NULL PRIMARY KEY,
  applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);`).Error
}

// listAppliedMigrations 读取 schema_migrations 表，返回已执行的 migration 文件名集合。
func listAppliedMigrations(ctx context.Context, gdb *gorm.DB) (map[string]bool, error) {
	type row struct {
		Version string `gorm:"column:version"`
	}
	var rows []row
	if err := gdb.WithContext(ctx).Raw(`SELECT version FROM schema_migrations`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.Version] = true
	}
	return out, nil
}

// markMigrationApplied 把指定 migration 版本写入 schema_migrations，表示已成功执行。
func markMigrationApplied(ctx context.Context, gdb *gorm.DB, version string) error {
	return gdb.WithContext(ctx).Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, version).Error
}

// splitSQLStatements 将 .sql 文件内容拆分为多条 SQL 语句。
//
// 处理规则：
// - 忽略空行
// - 忽略以 `--` 开头的单行注释
// - 使用分号 `;` 拆分语句
//
// 局限性：
// - 不解析字符串字面量/存储过程等复杂语法，仅适用于简单 DDL/DML 迁移脚本。
func splitSQLStatements(src []byte) []string {
	var buf bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(src))
	for sc.Scan() {
		line := sc.Text()
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if strings.HasPrefix(trim, "--") {
			continue
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	clean := buf.String()
	parts := strings.Split(clean, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}
