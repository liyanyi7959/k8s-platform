// migrations 以 embed 方式把 SQL 迁移文件打包进二进制。
//
// 这样部署时不依赖外部 .sql 文件：
// - 便于发布单一可执行文件
// - 避免“迁移文件忘记带上”导致线上启动失败
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
