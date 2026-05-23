<template>
  <div class="metrics-api-hint">
    <el-alert :title="titleText" type="warning" :closable="false" show-icon>
      <template #default>
        <div class="metrics-api-hint__desc">{{ descriptionText }}</div>

        <div class="metrics-api-hint__section">
          <div class="metrics-api-hint__section-head">
            <div class="metrics-api-hint__label">参考部署命令</div>
            <el-button size="small" text :icon="CopyDocument" @click="copySection(installCommand, '参考部署命令')">复制</el-button>
          </div>
          <pre class="metrics-api-hint__code">{{ installCommand }}</pre>
        </div>

        <div class="metrics-api-hint__section">
          <div class="metrics-api-hint__section-head">
            <div class="metrics-api-hint__label">常见补充参数</div>
            <el-button size="small" text :icon="CopyDocument" @click="copySection(patchCommand, '补充参数命令')">复制</el-button>
          </div>
          <pre class="metrics-api-hint__code">{{ patchCommand }}</pre>
        </div>

        <div class="metrics-api-hint__section">
          <div class="metrics-api-hint__section-head">
            <div class="metrics-api-hint__label">验证命令</div>
            <el-button size="small" text :icon="CopyDocument" @click="copySection(verifyCommands, '验证命令')">复制</el-button>
          </div>
          <pre class="metrics-api-hint__code">{{ verifyCommands }}</pre>
        </div>
      </template>
    </el-alert>
  </div>
</template>

<script setup lang="ts">
import { CopyDocument } from '@element-plus/icons-vue'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import { copyToClipboard } from '@/shared/utils/text'

const props = withDefaults(
  defineProps<{
    title?: string
    description?: string
  }>(),
  {
    title: '当前集群未启用 Metrics API',
    description: 'PodMetrics 与实时资源使用依赖 metrics-server。可先部署 metrics-server，再返回刷新页面。'
  }
)

const titleText = props.title
const descriptionText = props.description
const installCommand = 'kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml'
const patchCommand = `kubectl patch deployment metrics-server -n kube-system --type='json' -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'`
const verifyCommands = `kubectl get apiservice v1beta1.metrics.k8s.io
kubectl top nodes
kubectl top pods -A`

async function copySection(text: string, label: string) {
  try {
    await copyToClipboard(text)
    notifySuccess(`已复制${label}`)
  } catch (error) {
    const err = error as ApiError
    notifyError(err?.message ? `复制失败：${err.message}` : '复制失败')
  }
}
</script>

<style scoped>
.metrics-api-hint {
  width: 100%;
  max-width: 920px;
  margin: 0 auto;
  text-align: left;
}

.metrics-api-hint__desc {
  margin-bottom: 12px;
  color: var(--el-text-color-regular);
  line-height: 1.6;
}

.metrics-api-hint__section + .metrics-api-hint__section {
  margin-top: 12px;
}

.metrics-api-hint__section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.metrics-api-hint__label {
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}

.metrics-api-hint__code {
  margin: 6px 0 0;
  padding: 10px 12px;
  overflow-x: auto;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.24);
  background: rgba(248, 250, 252, 0.96);
  color: #0f172a;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

:global(html.dark) .metrics-api-hint__code {
  border-color: rgba(148, 163, 184, 0.2);
  background: rgba(15, 23, 42, 0.9);
  color: rgba(226, 232, 240, 0.92);
}
</style>