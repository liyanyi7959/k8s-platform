<template>
  <el-dropdown @command="onUserCommand">
    <div class="user-profile">
      <div class="avatar-ring">
        <span class="user-avatar">
          {{ (userStore.me?.username ?? 'U').slice(0, 1).toUpperCase() }}
        </span>
      </div>
      <div class="user-info hidden lg:flex">
        <span class="user-name">{{ userStore.me?.username ?? 'Admin' }}</span>
        <span class="user-role">Administrator</span>
      </div>
    </div>
    <template #dropdown>
      <el-dropdown-menu class="user-dropdown">
        <el-dropdown-item command="logout">
          <el-icon><SwitchButton /></el-icon>退出登录
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useUserStore } from '@/app/store/user'
import { SwitchButton } from '@element-plus/icons-vue'

const router = useRouter()
const userStore = useUserStore()

async function onUserCommand(cmd: string) {
  if (cmd === 'logout') {
    await userStore.logout()
    await router.push('/login')
  }
}
</script>

<style scoped>
.user-profile {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 12px 4px 6px;
  border-radius: 30px;
  cursor: pointer;
  transition: all 0.2s;
  background: rgba(255, 255, 255, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.3);
  backdrop-filter: blur(8px);
  flex: 0 0 auto;
}
html.dark .user-profile {
  background: rgba(30, 41, 59, 0.4);
  border-color: rgba(255, 255, 255, 0.08);
}
.user-profile:hover {
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 6px 16px rgba(59, 130, 246, 0.12);
  transform: translateY(-1px);
}
html.dark .user-profile:hover {
  background: rgba(30, 41, 59, 0.82);
  box-shadow: 0 6px 16px rgba(37, 99, 235, 0.18);
}
.avatar-ring {
  padding: 2px;
  border: 1px solid rgba(59, 130, 246, 0.3);
  border-radius: 10px;
  background: #fff;
}
html.dark .avatar-ring {
  background: #1e293b;
  border-color: rgba(96, 165, 250, 0.22);
}
.user-avatar {
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #2563eb, #60a5fa);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 700;
  font-size: 14px;
  box-shadow: 0 4px 10px rgba(37, 99, 235, 0.28);
}
.user-info {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
  min-width: 0;
}
.user-name {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
html.dark .user-name {
  color: #f1f5f9;
}
.user-role {
  font-size: 11px;
  color: #64748b;
  font-weight: 500;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
html.dark .user-role {
  color: #94a3b8;
}
</style>
