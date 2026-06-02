<template>
  <div class="settings-page">
    <section class="settings-hero">
      <div>
        <p class="eyebrow">Model Hub</p>
        <h1>AI 模型与提供商配置</h1>
        <p>
          统一维护模型接入、能力标签和默认排序。当前版本支持多提供商、多模型配置，为后续聊天、诊断、图片理解与工具调用做准备。
        </p>
      </div>
      <div class="hero-side">
        <div class="hero-box">
          <span>提供商</span>
          <strong>{{ providers.length }}</strong>
        </div>
        <div class="hero-box">
          <span>模型</span>
          <strong>{{ models.length }}</strong>
        </div>
      </div>
    </section>

    <section class="grid-layout">
      <div class="panel-card">
        <div class="panel-head">
          <div>
            <h2>提供商</h2>
            <p>保存 API 地址、密钥与扩展参数。</p>
          </div>
          <el-button type="primary" @click="openProviderDialog()">新增提供商</el-button>
        </div>

        <el-table :data="providers" stripe>
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="provider_type" label="类型" min-width="120" />
          <el-table-column prop="base_url" label="Base URL" min-width="220" />
          <el-table-column label="密钥" width="120">
            <template #default="{ row }">
              <el-tag size="small" :type="row.has_api_key ? 'success' : 'info'">
                {{ row.has_api_key ? '已配置' : '未配置' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="row.enabled ? 'success' : 'info'">
                {{ row.enabled ? '启用' : '停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button text @click="openProviderDialog(row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="panel-card">
        <div class="panel-head">
          <div>
            <h2>模型</h2>
            <p>标记模型的用途、视觉能力与工具能力。</p>
          </div>
          <el-button type="primary" @click="openModelDialog()">新增模型</el-button>
        </div>

        <el-table :data="models" stripe>
          <el-table-column prop="name" label="模型名称" min-width="150" />
          <el-table-column prop="provider_name" label="提供商" min-width="140" />
          <el-table-column prop="model_code" label="模型编码" min-width="180" />
          <el-table-column prop="model_type" label="类型" width="100" />
          <el-table-column label="能力" min-width="160">
            <template #default="{ row }">
              <div class="capability-tags">
                <el-tag size="small" effect="plain">{{ row.enabled ? '启用' : '停用' }}</el-tag>
                <el-tag v-if="row.supports_tools" size="small" type="warning" effect="plain">工具</el-tag>
                <el-tag v-if="row.supports_vision" size="small" type="success" effect="plain">视觉</el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button text @click="openModelDialog(row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </section>

    <el-dialog v-model="providerDialogVisible" :title="providerDialogTitle" width="620px">
      <el-form label-position="top">
        <el-form-item label="名称">
          <el-input v-model="providerForm.name" placeholder="例如 OpenAI Production" />
        </el-form-item>
        <el-form-item label="类型">
          <el-input v-model="providerForm.provider_type" placeholder="例如 openai / azure-openai / ollama / qwen" />
        </el-form-item>
        <el-form-item label="Base URL">
          <el-input v-model="providerForm.base_url" placeholder="例如 https://api.openai.com/v1" />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input v-model="providerForm.api_key" type="password" show-password placeholder="留空表示不修改；输入空字符串可在后端清空" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="providerForm.priority" :min="1" :max="9999" />
        </el-form-item>
        <el-form-item label="扩展配置 JSON">
          <el-input v-model="providerForm.meta_text" type="textarea" :rows="4" placeholder='例如 {"organization":"team-a"}' />
        </el-form-item>
        <el-form-item>
          <el-switch v-model="providerForm.enabled" active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="providerDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingProvider" @click="submitProvider">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="modelDialogVisible" :title="modelDialogTitle" width="620px">
      <el-form label-position="top">
        <el-form-item label="所属提供商">
          <el-select v-model="modelForm.provider_id" style="width: 100%" placeholder="请选择提供商">
            <el-option v-for="item in providers" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型名称">
          <el-input v-model="modelForm.name" placeholder="例如 GPT-4.1" />
        </el-form-item>
        <el-form-item label="模型编码">
          <el-input v-model="modelForm.model_code" placeholder="例如 gpt-4.1" />
        </el-form-item>
        <el-form-item label="模型类型">
          <el-input v-model="modelForm.model_type" placeholder="例如 chat / vision / embedding / image" />
        </el-form-item>
        <el-form-item label="最大输入 Token">
          <el-input-number v-model="modelForm.max_input_tokens" :min="0" :max="2000000" />
        </el-form-item>
        <el-form-item label="最大输出 Token">
          <el-input-number v-model="modelForm.max_output_tokens" :min="0" :max="2000000" />
        </el-form-item>
        <el-form-item label="扩展配置 JSON">
          <el-input v-model="modelForm.meta_text" type="textarea" :rows="4" placeholder='例如 {"temperature":0.2}' />
        </el-form-item>
        <div class="switch-row">
          <el-switch v-model="modelForm.enabled" active-text="启用" inactive-text="停用" />
          <el-switch v-model="modelForm.supports_tools" active-text="支持工具" inactive-text="不支持工具" />
          <el-switch v-model="modelForm.supports_vision" active-text="支持视觉" inactive-text="不支持视觉" />
        </div>
      </el-form>
      <template #footer>
        <el-button @click="modelDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingModel" @click="submitModel">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

import { createAIModel, createAIProvider, getAIModels, getAIProviders, patchAIModel, patchAIProvider, type AIModelItem, type AIProviderItem } from '@/features/ai/api/ai'

const providers = ref<AIProviderItem[]>([])
const models = ref<AIModelItem[]>([])
const providerDialogVisible = ref(false)
const modelDialogVisible = ref(false)
const savingProvider = ref(false)
const savingModel = ref(false)
const editingProviderId = ref<number>()
const editingModelId = ref<number>()

const providerForm = reactive({
  name: '',
  provider_type: 'openai',
  base_url: '',
  api_key: '',
  enabled: true,
  priority: 100,
  meta_text: ''
})

const modelForm = reactive({
  provider_id: undefined as number | undefined,
  name: '',
  model_code: '',
  model_type: 'chat',
  enabled: true,
  supports_tools: false,
  supports_vision: false,
  max_input_tokens: 0,
  max_output_tokens: 0,
  meta_text: ''
})

const providerDialogTitle = computed(() => editingProviderId.value ? '编辑提供商' : '新增提供商')
const modelDialogTitle = computed(() => editingModelId.value ? '编辑模型' : '新增模型')

async function loadData() {
  const [providerItems, modelItems] = await Promise.all([
    getAIProviders(),
    getAIModels()
  ])
  providers.value = providerItems
  models.value = modelItems
}

function openProviderDialog(item?: AIProviderItem) {
  editingProviderId.value = item?.id
  providerForm.name = item?.name ?? ''
  providerForm.provider_type = item?.provider_type ?? 'openai'
  providerForm.base_url = item?.base_url ?? ''
  providerForm.api_key = ''
  providerForm.enabled = item?.enabled ?? true
  providerForm.priority = item?.priority ?? 100
  providerForm.meta_text = item?.meta ? JSON.stringify(item.meta, null, 2) : ''
  providerDialogVisible.value = true
}

function openModelDialog(item?: AIModelItem) {
  editingModelId.value = item?.id
  modelForm.provider_id = item?.provider_id
  modelForm.name = item?.name ?? ''
  modelForm.model_code = item?.model_code ?? ''
  modelForm.model_type = item?.model_type ?? 'chat'
  modelForm.enabled = item?.enabled ?? true
  modelForm.supports_tools = item?.supports_tools ?? false
  modelForm.supports_vision = item?.supports_vision ?? false
  modelForm.max_input_tokens = item?.max_input_tokens ?? 0
  modelForm.max_output_tokens = item?.max_output_tokens ?? 0
  modelForm.meta_text = item?.meta ? JSON.stringify(item.meta, null, 2) : ''
  modelDialogVisible.value = true
}

function parseMeta(text: string) {
  const raw = text.trim()
  if (!raw) return undefined
  return JSON.parse(raw) as Record<string, unknown>
}

async function submitProvider() {
  savingProvider.value = true
  try {
    const payload = {
      name: providerForm.name,
      provider_type: providerForm.provider_type,
      base_url: providerForm.base_url,
      api_key: providerForm.api_key || undefined,
      enabled: providerForm.enabled,
      priority: providerForm.priority,
      meta: parseMeta(providerForm.meta_text)
    }
    if (editingProviderId.value) {
      await patchAIProvider(editingProviderId.value, payload)
    } else {
      await createAIProvider(payload)
    }
    providerDialogVisible.value = false
    await loadData()
    ElMessage.success('提供商已保存')
  } catch (error) {
    if (error instanceof SyntaxError) {
      ElMessage.error('提供商扩展配置 JSON 格式不正确')
      return
    }
    throw error
  } finally {
    savingProvider.value = false
  }
}

async function submitModel() {
  if (!modelForm.provider_id) {
    ElMessage.warning('请先选择提供商')
    return
  }
  savingModel.value = true
  try {
    const payload = {
      provider_id: modelForm.provider_id,
      name: modelForm.name,
      model_code: modelForm.model_code,
      model_type: modelForm.model_type,
      enabled: modelForm.enabled,
      supports_tools: modelForm.supports_tools,
      supports_vision: modelForm.supports_vision,
      max_input_tokens: modelForm.max_input_tokens,
      max_output_tokens: modelForm.max_output_tokens,
      meta: parseMeta(modelForm.meta_text)
    }
    if (editingModelId.value) {
      await patchAIModel(editingModelId.value, payload)
    } else {
      await createAIModel(payload)
    }
    modelDialogVisible.value = false
    await loadData()
    ElMessage.success('模型已保存')
  } catch (error) {
    if (error instanceof SyntaxError) {
      ElMessage.error('模型扩展配置 JSON 格式不正确')
      return
    }
    throw error
  } finally {
    savingModel.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 18px;
}

.settings-hero,
.panel-card {
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 24px;
  background:
    radial-gradient(circle at top right, rgba(191, 219, 254, 0.25), transparent 28%),
    linear-gradient(180deg, #ffffff, #f5f8ff);
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.06);
}

.settings-hero {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  padding: 28px;
}

.settings-hero h1 {
  margin: 0;
  color: #1f2937;
}

.settings-hero p {
  max-width: 760px;
  line-height: 1.7;
  color: #4b5563;
}

.eyebrow {
  margin: 0 0 8px;
  font-size: 12px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: #2563eb;
}

.hero-side {
  display: grid;
  gap: 12px;
  min-width: 180px;
}

.hero-box {
  padding: 16px 18px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  border: 1px solid rgba(15, 23, 42, 0.08);
}

.hero-box span {
  display: block;
  color: #6b7280;
  font-size: 12px;
}

.hero-box strong {
  display: block;
  margin-top: 6px;
  font-size: 26px;
  color: #1f2937;
}

.grid-layout {
  display: grid;
  grid-template-columns: 1fr;
  gap: 18px;
}

.panel-card {
  padding: 20px;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  margin-bottom: 16px;
}

.panel-head h2 {
  margin: 0;
  color: #1f2937;
}

.panel-head p {
  margin: 6px 0 0;
  color: #6b7280;
}

.capability-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.switch-row {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

@media (max-width: 1080px) {
  .settings-hero {
    flex-direction: column;
  }
}
</style>
