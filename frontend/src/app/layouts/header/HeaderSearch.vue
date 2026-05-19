<template>
  <el-autocomplete
    v-model="commandSearch"
    :fetch-suggestions="queryCommand"
    class="search-input-glass"
    popper-class="cmd-popper"
    :trigger-on-focus="false"
    @select="handleCommandSelect"
    placeholder="搜索资源..."
  >
    <template #prefix>
      <el-icon class="search-icon"><Search /></el-icon>
    </template>
    <template #suffix>
      <span class="search-shortcut">⌘ K</span>
    </template>
    <template #default="{ item }">
      <div class="command-item">
        <div class="command-content">
          <span class="command-title">{{ item.title }}</span>
          <span v-if="item.desc" class="command-desc">{{ item.desc }}</span>
        </div>
        <el-icon class="command-icon"><Monitor /></el-icon>
      </div>
    </template>
  </el-autocomplete>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Search, Monitor } from '@element-plus/icons-vue'

type MenuOption = { path: string; title: string }

const router = useRouter()
const commandSearch = ref('')

const menuOptions = computed<MenuOption[]>(() => {
  const out: MenuOption[] = []
  const seen = new Set<string>()
  for (const r of router.getRoutes()) {
    const title = String((r.meta as any)?.title ?? '')
    const requiresAuth = Boolean((r.meta as any)?.requiresAuth)
    const path = String(r.path ?? '')
    if (!path.startsWith('/')) continue
    if (!requiresAuth) continue
    if (!title) continue
    if (path === '/login') continue
    if (seen.has(path)) continue
    seen.add(path)
    out.push({ path, title })
  }
  out.sort((a, b) => a.title.localeCompare(b.title, 'zh-Hans-CN'))
  return out
})

const queryCommand = (queryString: string, cb: any) => {
  const results: any[] = []

  if (queryString) {
    const menuMatches = menuOptions.value.filter(it =>
      it.title.toLowerCase().includes(queryString.toLowerCase()) ||
      it.path.toLowerCase().includes(queryString.toLowerCase())
    )
    results.push(...menuMatches.map(m => ({ ...m, type: 'menu', desc: '跳转至菜单' })))
  }

  cb(results)
}

function handleCommandSelect(item: any) {
  commandSearch.value = ''
  router.push(item.path)
}
</script>

<style scoped>
.search-input-glass {
  width: auto;
  flex: 1 1 220px;
  min-width: 160px;
  max-width: 320px;
  transition: max-width 0.3s ease;
}
@media (min-width: 1280px) {
  .search-input-glass {
    max-width: 360px;
  }
}
@media (min-width: 1536px) {
  .search-input-glass {
    max-width: 480px;
  }
}
.search-input-glass :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid rgba(203, 213, 225, 0.6);
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.02);
  padding: 4px 12px;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
.search-input-glass :deep(.el-input__wrapper.is-focus),
.search-input-glass :deep(.el-input__wrapper:hover) {
  background: rgba(255, 255, 255, 0.95);
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.15), 0 8px 20px rgba(59, 130, 246, 0.1);
}
html.dark .search-input-glass :deep(.el-input__wrapper) {
  background: rgba(30, 41, 59, 0.5);
  border-color: rgba(100, 116, 139, 0.3);
}
html.dark .search-input-glass :deep(.el-input__wrapper.is-focus),
html.dark .search-input-glass :deep(.el-input__wrapper:hover) {
  background: rgba(30, 41, 59, 0.9);
  border-color: #60a5fa;
}
.search-input-glass :deep(.el-input__inner) {
  font-size: 14px;
  color: #0f172a;
  height: 32px;
  font-weight: 500;
}
html.dark .search-input-glass :deep(.el-input__inner) {
  color: #f1f5f9;
}
.search-icon {
  font-size: 18px;
  color: #64748b;
  margin-right: 8px;
}
.search-shortcut {
  font-size: 12px;
  font-family: 'Inter', sans-serif;
  color: #64748b;
  background: rgba(255,255,255,0.8);
  padding: 2px 6px;
  border-radius: 6px;
  border: 1px solid rgba(0,0,0,0.1);
  user-select: none;
  font-weight: 600;
}
html.dark .search-shortcut {
  background: rgba(30, 41, 59, 0.8);
  border-color: rgba(255,255,255,0.1);
  color: #94a3b8;
}
.command-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 4px;
}
.command-content {
  display: flex;
  flex-direction: column;
}
.command-title {
  font-weight: 500;
  color: #1e293b;
  font-size: 14px;
}
html.dark .command-title {
  color: #f1f5f9;
}
.command-desc {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 2px;
}
.command-icon {
  color: #94a3b8;
  font-size: 16px;
}
</style>
