<template>
  <div class="brand" @click="router.push('/')">
    <img :src="logoUrl" alt="星枢" class="brand-logo" />
    <div class="brand-mark">星枢</div>
    <div class="brand-text hidden md:block">
      <div class="brand-title">K8S管理平台</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import logoUrl from '@/assets/images/logo.svg'

const router = useRouter()
</script>

<style scoped>
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  margin-right: 32px;
  user-select: none;
  flex-shrink: 0;
}
.brand-logo {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
}
.brand-mark {
  font-weight: 900;
  font-size: 20px;
  background: linear-gradient(135deg, #2563eb 0%, #60a5fa 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  letter-spacing: -0.5px;
}
.brand-text {
  display: flex;
  flex-direction: column;
}
.brand-title {
  font-size: 16px;
  font-weight: 700;
  color: #1e293b;
  letter-spacing: 0.5px;
}
html.dark .brand-title {
  color: #f1f5f9;
}
</style>
