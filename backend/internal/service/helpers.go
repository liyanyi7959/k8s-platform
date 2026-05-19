// helpers.go 包含 service 层共用的通用工具函数。
//
// 放置原则：
// - 被多个 service 文件共用的纯工具函数放在此文件；
// - 仅被单个 service 使用的工具函数留在对应文件中。
package service

// normalizePage 统一分页参数：默认 page=1/pageSize=10，并限制最大 pageSize=200。
// 被 rbac_service / server_service / cluster_registry_service / credential_service 等共用。
func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
