<template>
  <el-drawer
    v-model="jobEditVisible"
    :title="`编辑 Job`"
    size="72%"
    class="deployment-edit-drawer"
    :close-on-click-modal="false"
    :with-header="true"
  >
    <el-form label-width="0" class="deployment-edit-form">
      <el-card shadow="never" class="edit-section-card edit-section-card--base">
        <template #header>
          <div class="edit-section-title-row">
            <div class="edit-title-stack">
              <div class="edit-section-title">基础信息</div>
              <div class="edit-meta-row">
                <div class="base-meta-row">
                  <span class="base-meta-pill base-meta-pill--cluster">cluster={{ props.clusterName || (props.clusterId != null ? String(props.clusterId) : '-') }}</span>
                  <span class="base-meta-pill base-meta-pill--kind">Job</span>
                  <span class="base-meta-pill base-meta-pill--target">{{ jobEditNamespace }}/{{ jobEditName }}</span>
                </div>
              </div>
            </div>
            <el-tag size="small" :type="isJobEditChanged ? 'warning' : 'info'" effect="light">{{ isJobEditChanged ? '已修改' : '未修改' }}</el-tag>
          </div>
        </template>
        <div class="edit-grid4">
          <div class="edit-kv">
            <div class="edit-k">命名空间</div>
            <div class="edit-v"><div class="edit-ro mono">{{ jobEditNamespace }}</div></div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">名称</div>
            <div class="edit-v"><div class="edit-ro mono">{{ jobEditName }}</div></div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">parallelism</div>
            <div class="edit-v">
              <el-input-number v-model="jobEditParallelism" size="small" class="edit-field" :class="{ 'is-changed': isJobParallelismChanged }" :min="0" :controls="false" />
            </div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">completions</div>
            <div class="edit-v">
              <el-input-number v-model="jobEditCompletions" size="small" class="edit-field" :class="{ 'is-changed': isJobCompletionsChanged }" :min="0" :controls="false" />
            </div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">backoffLimit</div>
            <div class="edit-v">
              <el-input-number v-model="jobEditBackoffLimit" size="small" class="edit-field" :class="{ 'is-changed': isJobBackoffLimitChanged }" :min="0" :controls="false" />
            </div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">ttlSecondsAfterFinished</div>
            <div class="edit-v">
              <el-input-number
                v-model="jobEditTtlSecondsAfterFinished"
                size="small"
                class="edit-field"
                :class="{ 'is-changed': isJobTtlChanged }"
                :min="0"
                :controls="false"
              />
            </div>
          </div>
          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Labels(JSON)</div>
            <div class="edit-v">
              <el-input v-model="jobEditLabelsText" type="textarea" :rows="6" class="edit-field" :class="{ 'is-changed': isJobLabelsChanged }" />
            </div>
          </div>
        </div>
      </el-card>

      <div class="edit-drawer-footer">
        <el-button @click="jobEditVisible = false">取消</el-button>
        <el-button type="primary" :loading="jobEditSaving" @click="saveEditJob">保存</el-button>
      </div>
    </el-form>
  </el-drawer>

  <el-drawer
    v-model="cronJobEditVisible"
    :title="`编辑 CronJob`"
    size="72%"
    class="deployment-edit-drawer"
    :close-on-click-modal="false"
    :with-header="true"
  >
    <el-form label-width="0" class="deployment-edit-form">
      <el-card shadow="never" class="edit-section-card edit-section-card--base">
        <template #header>
          <div class="edit-section-title-row">
            <div class="edit-title-stack">
              <div class="edit-section-title">基础信息</div>
              <div class="edit-meta-row">
                <div class="base-meta-row">
                  <span class="base-meta-pill base-meta-pill--cluster">cluster={{ props.clusterName || (props.clusterId != null ? String(props.clusterId) : '-') }}</span>
                  <span class="base-meta-pill base-meta-pill--kind">CronJob</span>
                  <span class="base-meta-pill base-meta-pill--target">{{ cronJobEditNamespace }}/{{ cronJobEditName }}</span>
                </div>
              </div>
            </div>
            <el-tag size="small" :type="isCronJobEditChanged ? 'warning' : 'info'" effect="light">
              {{ isCronJobEditChanged ? '已修改' : '未修改' }}
            </el-tag>
          </div>
        </template>

        <div class="edit-grid4">
          <div class="edit-kv">
            <div class="edit-k">命名空间</div>
            <div class="edit-v"><div class="edit-ro mono">{{ cronJobEditNamespace }}</div></div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">名称</div>
            <div class="edit-v"><div class="edit-ro mono">{{ cronJobEditName }}</div></div>
          </div>
          <div class="edit-kv edit-kv--span2">
            <div class="edit-k">Schedule</div>
            <div class="edit-v">
              <el-input v-model="cronJobEditSchedule" size="small" class="edit-field" :class="{ 'is-changed': isCronJobScheduleChanged }" placeholder="*/5 * * * *" />
            </div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">Suspend</div>
            <div class="edit-v"><el-switch v-model="cronJobEditSuspend" /></div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">concurrencyPolicy</div>
            <div class="edit-v">
              <el-select v-model="cronJobEditConcurrencyPolicy" size="small" clearable class="edit-field" :class="{ 'is-changed': isCronJobConcurrencyPolicyChanged }">
                <el-option label="Allow" value="Allow" />
                <el-option label="Forbid" value="Forbid" />
                <el-option label="Replace" value="Replace" />
              </el-select>
            </div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">successfulJobsHistoryLimit</div>
            <div class="edit-v">
              <el-input-number
                v-model="cronJobEditSuccessfulJobsHistoryLimit"
                size="small"
                class="edit-field"
                :class="{ 'is-changed': isCronJobSuccessHistoryChanged }"
                :min="0"
                :controls="false"
              />
            </div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">failedJobsHistoryLimit</div>
            <div class="edit-v">
              <el-input-number
                v-model="cronJobEditFailedJobsHistoryLimit"
                size="small"
                class="edit-field"
                :class="{ 'is-changed': isCronJobFailedHistoryChanged }"
                :min="0"
                :controls="false"
              />
            </div>
          </div>
          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Labels(JSON)</div>
            <div class="edit-v">
              <el-input v-model="cronJobEditLabelsText" type="textarea" :rows="6" class="edit-field" :class="{ 'is-changed': isCronJobLabelsChanged }" />
            </div>
          </div>
        </div>
      </el-card>

      <div class="edit-drawer-footer">
        <el-button @click="cronJobEditVisible = false">取消</el-button>
        <el-button type="primary" :loading="cronJobEditSaving" :disabled="!cronJobEditSchedule.trim()" @click="saveEditCronJob">保存</el-button>
      </div>
    </el-form>
  </el-drawer>

  <el-drawer v-model="serviceEditVisible" :title="`编辑 Service`" size="72%" class="deployment-edit-drawer" :close-on-click-modal="false" :with-header="true">
    <el-form label-width="0" class="deployment-edit-form">
      <el-card shadow="never" class="edit-section-card edit-section-card--base">
        <template #header>
          <div class="edit-section-title-row">
            <div class="edit-title-stack">
              <div class="edit-section-title">基础信息</div>
              <div class="edit-meta-row">
                <div class="base-meta-row">
                  <span class="base-meta-pill base-meta-pill--cluster">cluster={{ props.clusterName || (props.clusterId != null ? String(props.clusterId) : '-') }}</span>
                  <span class="base-meta-pill base-meta-pill--kind">Service</span>
                  <span class="base-meta-pill base-meta-pill--target">{{ serviceEditNamespace }}/{{ serviceEditName }}</span>
                </div>
              </div>
            </div>
            <el-tag size="small" :type="isServiceEditChanged ? 'warning' : 'info'" effect="light">{{ isServiceEditChanged ? '已修改' : '未修改' }}</el-tag>
          </div>
        </template>

        <div class="edit-grid4">
          <div class="edit-kv">
            <div class="edit-k">命名空间</div>
            <div class="edit-v"><div class="edit-ro mono">{{ serviceEditNamespace }}</div></div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">名称</div>
            <div class="edit-v"><div class="edit-ro mono">{{ serviceEditName }}</div></div>
          </div>
          <div class="edit-kv edit-kv--span2">
            <div class="edit-k">Type</div>
            <div class="edit-v">
              <el-select v-model="serviceEditType" size="small" clearable class="edit-field" :class="{ 'is-changed': isServiceTypeChanged }">
                <el-option label="ClusterIP" value="ClusterIP" />
                <el-option label="NodePort" value="NodePort" />
                <el-option label="LoadBalancer" value="LoadBalancer" />
                <el-option label="ExternalName" value="ExternalName" />
              </el-select>
            </div>
          </div>

          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Labels(JSON)</div>
            <div class="edit-v">
              <div class="k8s-pane-toolbar">
                <el-space :size="8" wrap>
                  <el-tooltip content="搜索" placement="top"><el-button size="small" :icon="Search" circle @click="openServiceEditLabelsSearch" /></el-tooltip>
                  <el-tooltip content="折叠全部" placement="top"><el-button size="small" :icon="Fold" circle @click="foldServiceEditLabelsAll" /></el-tooltip>
                  <el-tooltip content="展开全部" placement="top"><el-button size="small" :icon="Expand" circle @click="unfoldServiceEditLabelsAll" /></el-tooltip>
                  <el-switch v-model="serviceEditShowDiff" inline-prompt active-text="Diff" inactive-text="编辑" />
                  <el-switch v-model="serviceEditWrap" inline-prompt active-text="换行" inactive-text="单行" />
                  <el-switch v-model="serviceEditLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
                  <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                    <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
                  </el-tooltip>
                </el-space>
              </div>
              <CodeMirrorViewer
                ref="serviceEditLabelsViewerRef"
                v-model:text="serviceEditLabelsText"
                :compare-text="serviceEditLabelsOriginalText"
                :show-diff="serviceEditShowDiff"
                language="json"
                :theme="props.editorTheme"
                :read-only="false"
                :wrap="serviceEditWrap"
                :line-numbers="serviceEditLineNumbers"
                height="220px"
                class="k8s-detail-box k8s-detail-box--fill"
              />
            </div>
          </div>

          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Annotations(JSON)</div>
            <div class="edit-v">
              <div class="k8s-pane-toolbar">
                <el-space :size="8" wrap>
                  <el-tooltip content="搜索" placement="top"><el-button size="small" :icon="Search" circle @click="openServiceEditAnnotationsSearch" /></el-tooltip>
                  <el-tooltip content="折叠全部" placement="top"><el-button size="small" :icon="Fold" circle @click="foldServiceEditAnnotationsAll" /></el-tooltip>
                  <el-tooltip content="展开全部" placement="top"><el-button size="small" :icon="Expand" circle @click="unfoldServiceEditAnnotationsAll" /></el-tooltip>
                  <el-switch v-model="serviceEditShowDiff" inline-prompt active-text="Diff" inactive-text="编辑" />
                  <el-switch v-model="serviceEditWrap" inline-prompt active-text="换行" inactive-text="单行" />
                  <el-switch v-model="serviceEditLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
                  <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                    <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
                  </el-tooltip>
                </el-space>
              </div>
              <CodeMirrorViewer
                ref="serviceEditAnnotationsViewerRef"
                v-model:text="serviceEditAnnotationsText"
                :compare-text="serviceEditAnnotationsOriginalText"
                :show-diff="serviceEditShowDiff"
                language="json"
                :theme="props.editorTheme"
                :read-only="false"
                :wrap="serviceEditWrap"
                :line-numbers="serviceEditLineNumbers"
                height="220px"
                class="k8s-detail-box k8s-detail-box--fill"
              />
            </div>
          </div>

          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Selector(JSON)</div>
            <div class="edit-v">
              <div class="k8s-pane-toolbar">
                <el-space :size="8" wrap>
                  <el-tooltip content="搜索" placement="top"><el-button size="small" :icon="Search" circle @click="openServiceEditSelectorSearch" /></el-tooltip>
                  <el-tooltip content="折叠全部" placement="top"><el-button size="small" :icon="Fold" circle @click="foldServiceEditSelectorAll" /></el-tooltip>
                  <el-tooltip content="展开全部" placement="top"><el-button size="small" :icon="Expand" circle @click="unfoldServiceEditSelectorAll" /></el-tooltip>
                  <el-switch v-model="serviceEditShowDiff" inline-prompt active-text="Diff" inactive-text="编辑" />
                  <el-switch v-model="serviceEditWrap" inline-prompt active-text="换行" inactive-text="单行" />
                  <el-switch v-model="serviceEditLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
                  <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                    <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
                  </el-tooltip>
                </el-space>
              </div>
              <CodeMirrorViewer
                ref="serviceEditSelectorViewerRef"
                v-model:text="serviceEditSelectorText"
                :compare-text="serviceEditSelectorOriginalText"
                :show-diff="serviceEditShowDiff"
                language="json"
                :theme="props.editorTheme"
                :read-only="false"
                :wrap="serviceEditWrap"
                :line-numbers="serviceEditLineNumbers"
                height="220px"
                class="k8s-detail-box k8s-detail-box--fill"
              />
            </div>
          </div>
        </div>
      </el-card>

      <div class="edit-drawer-footer">
        <el-button @click="serviceEditVisible = false">取消</el-button>
        <el-button type="primary" :loading="serviceEditSaving" :disabled="!isServiceEditChanged" @click="saveEditService">保存</el-button>
      </div>
    </el-form>
  </el-drawer>

  <el-drawer v-model="ingressEditVisible" :title="`编辑 Ingress`" size="72%" class="deployment-edit-drawer" :close-on-click-modal="false" :with-header="true">
    <el-form label-width="0" class="deployment-edit-form">
      <el-card shadow="never" class="edit-section-card edit-section-card--base">
        <template #header>
          <div class="edit-section-title-row">
            <div class="edit-title-stack">
              <div class="edit-section-title">基础信息</div>
              <div class="edit-meta-row">
                <div class="base-meta-row">
                  <span class="base-meta-pill base-meta-pill--cluster">cluster={{ props.clusterName || (props.clusterId != null ? String(props.clusterId) : '-') }}</span>
                  <span class="base-meta-pill base-meta-pill--kind">Ingress</span>
                  <span class="base-meta-pill base-meta-pill--target">{{ ingressEditNamespace }}/{{ ingressEditName }}</span>
                </div>
              </div>
            </div>
            <el-tag size="small" :type="isIngressEditChanged ? 'warning' : 'info'" effect="light">{{ isIngressEditChanged ? '已修改' : '未修改' }}</el-tag>
          </div>
        </template>

        <div class="edit-grid4">
          <div class="edit-kv">
            <div class="edit-k">命名空间</div>
            <div class="edit-v"><div class="edit-ro mono">{{ ingressEditNamespace }}</div></div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">名称</div>
            <div class="edit-v"><div class="edit-ro mono">{{ ingressEditName }}</div></div>
          </div>
          <div class="edit-kv edit-kv--span2">
            <div class="edit-k">ingressClassName</div>
            <div class="edit-v">
              <el-input v-model="ingressEditClassName" size="small" clearable class="edit-field" :class="{ 'is-changed': isIngressClassChanged }" placeholder="可选" />
            </div>
          </div>

          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Labels(JSON)</div>
            <div class="edit-v">
              <div class="k8s-pane-toolbar">
                <el-space :size="8" wrap>
                  <el-tooltip content="搜索" placement="top"><el-button size="small" :icon="Search" circle @click="openIngressEditLabelsSearch" /></el-tooltip>
                  <el-tooltip content="折叠全部" placement="top"><el-button size="small" :icon="Fold" circle @click="foldIngressEditLabelsAll" /></el-tooltip>
                  <el-tooltip content="展开全部" placement="top"><el-button size="small" :icon="Expand" circle @click="unfoldIngressEditLabelsAll" /></el-tooltip>
                  <el-switch v-model="ingressEditShowDiff" inline-prompt active-text="Diff" inactive-text="编辑" />
                  <el-switch v-model="ingressEditWrap" inline-prompt active-text="换行" inactive-text="单行" />
                  <el-switch v-model="ingressEditLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
                  <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                    <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
                  </el-tooltip>
                </el-space>
              </div>
              <CodeMirrorViewer
                ref="ingressEditLabelsViewerRef"
                v-model:text="ingressEditLabelsText"
                :compare-text="ingressEditLabelsOriginalText"
                :show-diff="ingressEditShowDiff"
                language="json"
                :theme="props.editorTheme"
                :read-only="false"
                :wrap="ingressEditWrap"
                :line-numbers="ingressEditLineNumbers"
                height="220px"
                class="k8s-detail-box k8s-detail-box--fill"
              />
            </div>
          </div>

          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Annotations(JSON)</div>
            <div class="edit-v">
              <div class="k8s-pane-toolbar">
                <el-space :size="8" wrap>
                  <el-tooltip content="搜索" placement="top"><el-button size="small" :icon="Search" circle @click="openIngressEditAnnotationsSearch" /></el-tooltip>
                  <el-tooltip content="折叠全部" placement="top"><el-button size="small" :icon="Fold" circle @click="foldIngressEditAnnotationsAll" /></el-tooltip>
                  <el-tooltip content="展开全部" placement="top"><el-button size="small" :icon="Expand" circle @click="unfoldIngressEditAnnotationsAll" /></el-tooltip>
                  <el-switch v-model="ingressEditShowDiff" inline-prompt active-text="Diff" inactive-text="编辑" />
                  <el-switch v-model="ingressEditWrap" inline-prompt active-text="换行" inactive-text="单行" />
                  <el-switch v-model="ingressEditLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
                  <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                    <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
                  </el-tooltip>
                </el-space>
              </div>
              <CodeMirrorViewer
                ref="ingressEditAnnotationsViewerRef"
                v-model:text="ingressEditAnnotationsText"
                :compare-text="ingressEditAnnotationsOriginalText"
                :show-diff="ingressEditShowDiff"
                language="json"
                :theme="props.editorTheme"
                :read-only="false"
                :wrap="ingressEditWrap"
                :line-numbers="ingressEditLineNumbers"
                height="260px"
                class="k8s-detail-box k8s-detail-box--fill"
              />
            </div>
          </div>
        </div>
      </el-card>

      <div class="edit-drawer-footer">
        <el-button @click="ingressEditVisible = false">取消</el-button>
        <el-button type="primary" :loading="ingressEditSaving" :disabled="!isIngressEditChanged" @click="saveEditIngress">保存</el-button>
      </div>
    </el-form>
  </el-drawer>

  <el-drawer v-model="ingressClassEditVisible" :title="`编辑 IngressClass`" size="72%" class="deployment-edit-drawer" :close-on-click-modal="false" :with-header="true">
    <el-form label-width="0" class="deployment-edit-form">
      <el-card shadow="never" class="edit-section-card edit-section-card--base">
        <template #header>
          <div class="edit-section-title-row">
            <div class="edit-title-stack">
              <div class="edit-section-title">基础信息</div>
              <div class="edit-meta-row">
                <div class="base-meta-row">
                  <span class="base-meta-pill base-meta-pill--cluster">cluster={{ props.clusterName || (props.clusterId != null ? String(props.clusterId) : '-') }}</span>
                  <span class="base-meta-pill base-meta-pill--kind">IngressClass</span>
                  <span class="base-meta-pill base-meta-pill--target">{{ ingressClassEditName }}</span>
                </div>
              </div>
            </div>
            <el-tag size="small" :type="isIngressClassEditChanged ? 'warning' : 'info'" effect="light">{{ isIngressClassEditChanged ? '已修改' : '未修改' }}</el-tag>
          </div>
        </template>

        <div class="edit-grid4">
          <div class="edit-kv edit-kv--span2">
            <div class="edit-k">名称</div>
            <div class="edit-v"><div class="edit-ro mono">{{ ingressClassEditName }}</div></div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">Default</div>
            <div class="edit-v"><el-switch v-model="ingressClassEditDefault" /></div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">controller</div>
            <div class="edit-v">
              <el-input v-model="ingressClassEditController" size="small" class="edit-field" :class="{ 'is-changed': isIngressClassControllerChanged }" placeholder="k8s.io/ingress-nginx" />
            </div>
          </div>

          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Labels(JSON)</div>
            <div class="edit-v">
              <div class="k8s-pane-toolbar">
                <el-space :size="8" wrap>
                  <el-tooltip content="搜索" placement="top"><el-button size="small" :icon="Search" circle @click="openIngressClassEditLabelsSearch" /></el-tooltip>
                  <el-tooltip content="折叠全部" placement="top"><el-button size="small" :icon="Fold" circle @click="foldIngressClassEditLabelsAll" /></el-tooltip>
                  <el-tooltip content="展开全部" placement="top"><el-button size="small" :icon="Expand" circle @click="unfoldIngressClassEditLabelsAll" /></el-tooltip>
                  <el-switch v-model="ingressClassEditShowDiff" inline-prompt active-text="Diff" inactive-text="编辑" />
                  <el-switch v-model="ingressClassEditWrap" inline-prompt active-text="换行" inactive-text="单行" />
                  <el-switch v-model="ingressClassEditLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
                  <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                    <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
                  </el-tooltip>
                </el-space>
              </div>
              <CodeMirrorViewer
                ref="ingressClassEditLabelsViewerRef"
                v-model:text="ingressClassEditLabelsText"
                :compare-text="ingressClassEditLabelsOriginalText"
                :show-diff="ingressClassEditShowDiff"
                language="json"
                :theme="props.editorTheme"
                :read-only="false"
                :wrap="ingressClassEditWrap"
                :line-numbers="ingressClassEditLineNumbers"
                height="220px"
                class="k8s-detail-box k8s-detail-box--fill"
              />
            </div>
          </div>

          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Annotations(JSON)</div>
            <div class="edit-v">
              <div class="k8s-pane-toolbar">
                <el-space :size="8" wrap>
                  <el-tooltip content="搜索" placement="top"><el-button size="small" :icon="Search" circle @click="openIngressClassEditAnnotationsSearch" /></el-tooltip>
                  <el-tooltip content="折叠全部" placement="top"><el-button size="small" :icon="Fold" circle @click="foldIngressClassEditAnnotationsAll" /></el-tooltip>
                  <el-tooltip content="展开全部" placement="top"><el-button size="small" :icon="Expand" circle @click="unfoldIngressClassEditAnnotationsAll" /></el-tooltip>
                  <el-switch v-model="ingressClassEditShowDiff" inline-prompt active-text="Diff" inactive-text="编辑" />
                  <el-switch v-model="ingressClassEditWrap" inline-prompt active-text="换行" inactive-text="单行" />
                  <el-switch v-model="ingressClassEditLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
                  <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                    <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
                  </el-tooltip>
                </el-space>
              </div>
              <CodeMirrorViewer
                ref="ingressClassEditAnnotationsViewerRef"
                v-model:text="ingressClassEditAnnotationsText"
                :compare-text="ingressClassEditAnnotationsOriginalText"
                :show-diff="ingressClassEditShowDiff"
                language="json"
                :theme="props.editorTheme"
                :read-only="false"
                :wrap="ingressClassEditWrap"
                :line-numbers="ingressClassEditLineNumbers"
                height="260px"
                class="k8s-detail-box k8s-detail-box--fill"
              />
            </div>
          </div>
        </div>
      </el-card>

      <div class="edit-drawer-footer">
        <el-button @click="ingressClassEditVisible = false">取消</el-button>
        <el-button type="primary" :loading="ingressClassEditSaving" :disabled="!isIngressClassEditChanged || !ingressClassEditController.trim()" @click="saveEditIngressClass">
          保存
        </el-button>
      </div>
    </el-form>
  </el-drawer>

  <el-drawer
    v-model="configMapEditVisible"
    :title="`编辑 ConfigMap`"
    size="72%"
    class="deployment-edit-drawer"
    :close-on-click-modal="false"
    :with-header="true"
  >
    <el-form label-width="0" class="deployment-edit-form">
      <el-card shadow="never" class="edit-section-card edit-section-card--base">
        <template #header>
          <div class="edit-section-title-row">
            <div class="edit-title-stack">
              <div class="edit-section-title">基础信息</div>
              <div class="edit-meta-row">
                <div class="base-meta-row">
                  <span class="base-meta-pill base-meta-pill--cluster">cluster={{ props.clusterName || (props.clusterId != null ? String(props.clusterId) : '-') }}</span>
                  <span class="base-meta-pill base-meta-pill--kind">ConfigMap</span>
                  <span class="base-meta-pill base-meta-pill--target">{{ configMapEditNamespace }}/{{ configMapEditName }}</span>
                </div>
              </div>
            </div>
            <el-tag size="small" :type="isConfigMapEditChanged ? 'warning' : 'info'" effect="light">
              {{ isConfigMapEditChanged ? '已修改' : '未修改' }}
            </el-tag>
          </div>
        </template>

        <div class="edit-grid4">
          <div class="edit-kv">
            <div class="edit-k">命名空间</div>
            <div class="edit-v"><div class="edit-ro mono">{{ configMapEditNamespace }}</div></div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">名称</div>
            <div class="edit-v"><div class="edit-ro mono">{{ configMapEditName }}</div></div>
          </div>
          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Labels(JSON)</div>
            <div class="edit-v">
              <div class="k8s-pane-toolbar">
                <el-space :size="8" wrap>
                  <el-tooltip content="搜索" placement="top">
                    <el-button size="small" :icon="Search" circle @click="openConfigMapEditLabelsSearch" />
                  </el-tooltip>
                  <el-tooltip content="折叠全部" placement="top">
                    <el-button size="small" :icon="Fold" circle @click="foldConfigMapEditLabelsAll" />
                  </el-tooltip>
                  <el-tooltip content="展开全部" placement="top">
                    <el-button size="small" :icon="Expand" circle @click="unfoldConfigMapEditLabelsAll" />
                  </el-tooltip>
                  <el-switch v-model="configMapEditShowDiff" inline-prompt active-text="Diff" inactive-text="编辑" />
                  <el-switch v-model="configMapEditWrap" inline-prompt active-text="换行" inactive-text="单行" />
                  <el-switch v-model="configMapEditLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
                  <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                    <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
                  </el-tooltip>
                </el-space>
              </div>
              <CodeMirrorViewer
                ref="configMapEditLabelsViewerRef"
                v-model:text="configMapEditLabelsText"
                :compare-text="configMapEditLabelsOriginalText"
                :show-diff="configMapEditShowDiff"
                language="json"
                :theme="props.editorTheme"
                :read-only="false"
                :wrap="configMapEditWrap"
                :line-numbers="configMapEditLineNumbers"
                height="220px"
                class="k8s-detail-box k8s-detail-box--fill"
              />
            </div>
          </div>
          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Data(JSON)</div>
            <div class="edit-v">
              <div class="k8s-pane-toolbar">
                <el-space :size="8" wrap>
                  <el-tooltip content="搜索" placement="top">
                    <el-button size="small" :icon="Search" circle @click="openConfigMapEditDataSearch" />
                  </el-tooltip>
                  <el-tooltip content="折叠全部" placement="top">
                    <el-button size="small" :icon="Fold" circle @click="foldConfigMapEditDataAll" />
                  </el-tooltip>
                  <el-tooltip content="展开全部" placement="top">
                    <el-button size="small" :icon="Expand" circle @click="unfoldConfigMapEditDataAll" />
                  </el-tooltip>
                  <el-switch v-model="configMapEditShowDiff" inline-prompt active-text="Diff" inactive-text="编辑" />
                  <el-switch v-model="configMapEditWrap" inline-prompt active-text="换行" inactive-text="单行" />
                  <el-switch v-model="configMapEditLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
                  <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                    <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
                  </el-tooltip>
                </el-space>
              </div>
              <CodeMirrorViewer
                ref="configMapEditDataViewerRef"
                v-model:text="configMapEditDataText"
                :compare-text="configMapEditDataOriginalText"
                :show-diff="configMapEditShowDiff"
                language="json"
                :theme="props.editorTheme"
                :read-only="false"
                :wrap="configMapEditWrap"
                :line-numbers="configMapEditLineNumbers"
                height="360px"
                class="k8s-detail-box k8s-detail-box--fill"
              />
            </div>
          </div>
        </div>
      </el-card>

      <div class="edit-drawer-footer">
        <el-button @click="configMapEditVisible = false">取消</el-button>
        <el-button type="primary" :loading="configMapEditSaving" :disabled="!isConfigMapEditChanged" @click="saveEditConfigMap">保存</el-button>
      </div>
    </el-form>
  </el-drawer>

  <el-drawer
    v-model="secretEditVisible"
    :title="`编辑 Secret`"
    size="72%"
    class="deployment-edit-drawer"
    :close-on-click-modal="false"
    :with-header="true"
  >
    <el-form label-width="0" class="deployment-edit-form">
      <el-card shadow="never" class="edit-section-card edit-section-card--base">
        <template #header>
          <div class="edit-section-title-row">
            <div class="edit-title-stack">
              <div class="edit-section-title">基础信息</div>
              <div class="edit-meta-row">
                <div class="base-meta-row">
                  <span class="base-meta-pill base-meta-pill--cluster">cluster={{ props.clusterName || (props.clusterId != null ? String(props.clusterId) : '-') }}</span>
                  <span class="base-meta-pill base-meta-pill--kind">Secret</span>
                  <span class="base-meta-pill base-meta-pill--target">{{ secretEditNamespace }}/{{ secretEditName }}</span>
                </div>
              </div>
            </div>
            <el-tag size="small" :type="isSecretEditChanged ? 'warning' : 'info'" effect="light">{{ isSecretEditChanged ? '已修改' : '未修改' }}</el-tag>
          </div>
        </template>

        <div class="edit-grid4">
          <div class="edit-kv">
            <div class="edit-k">命名空间</div>
            <div class="edit-v"><div class="edit-ro mono">{{ secretEditNamespace }}</div></div>
          </div>
          <div class="edit-kv">
            <div class="edit-k">名称</div>
            <div class="edit-v"><div class="edit-ro mono">{{ secretEditName }}</div></div>
          </div>
          <div class="edit-kv edit-kv--span2">
            <div class="edit-k">Type</div>
            <div class="edit-v">
              <el-input v-model="secretEditType" size="small" class="edit-field" :class="{ 'is-changed': isSecretTypeChanged }" placeholder="Opaque" />
            </div>
          </div>
          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Labels(JSON)</div>
            <div class="edit-v">
              <div class="k8s-pane-toolbar">
                <el-space :size="8" wrap>
                  <el-tooltip content="搜索" placement="top">
                    <el-button size="small" :icon="Search" circle @click="openSecretEditLabelsSearch" />
                  </el-tooltip>
                  <el-tooltip content="折叠全部" placement="top">
                    <el-button size="small" :icon="Fold" circle @click="foldSecretEditLabelsAll" />
                  </el-tooltip>
                  <el-tooltip content="展开全部" placement="top">
                    <el-button size="small" :icon="Expand" circle @click="unfoldSecretEditLabelsAll" />
                  </el-tooltip>
                  <el-switch v-model="secretEditShowDiff" inline-prompt active-text="Diff" inactive-text="编辑" />
                  <el-switch v-model="secretEditWrap" inline-prompt active-text="换行" inactive-text="单行" />
                  <el-switch v-model="secretEditLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
                  <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                    <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
                  </el-tooltip>
                </el-space>
              </div>
              <CodeMirrorViewer
                ref="secretEditLabelsViewerRef"
                v-model:text="secretEditLabelsText"
                :compare-text="secretEditLabelsOriginalText"
                :show-diff="secretEditShowDiff"
                language="json"
                :theme="props.editorTheme"
                :read-only="false"
                :wrap="secretEditWrap"
                :line-numbers="secretEditLineNumbers"
                height="220px"
                class="k8s-detail-box k8s-detail-box--fill"
              />
            </div>
          </div>
          <div class="edit-kv edit-kv--span4">
            <div class="edit-k">Data(JSON)</div>
            <div class="edit-v">
              <div class="k8s-pane-toolbar">
                <el-space :size="8" wrap>
                  <el-tooltip content="搜索" placement="top">
                    <el-button size="small" :icon="Search" circle @click="openSecretEditDataSearch" />
                  </el-tooltip>
                  <el-tooltip content="折叠全部" placement="top">
                    <el-button size="small" :icon="Fold" circle @click="foldSecretEditDataAll" />
                  </el-tooltip>
                  <el-tooltip content="展开全部" placement="top">
                    <el-button size="small" :icon="Expand" circle @click="unfoldSecretEditDataAll" />
                  </el-tooltip>
                  <el-switch v-model="secretEditShowDiff" inline-prompt active-text="Diff" inactive-text="编辑" />
                  <el-switch v-model="secretEditWrap" inline-prompt active-text="换行" inactive-text="单行" />
                  <el-switch v-model="secretEditLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
                  <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                    <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
                  </el-tooltip>
                </el-space>
              </div>
              <CodeMirrorViewer
                ref="secretEditDataViewerRef"
                v-model:text="secretEditDataText"
                :compare-text="secretEditDataOriginalText"
                :show-diff="secretEditShowDiff"
                language="json"
                :theme="props.editorTheme"
                :read-only="false"
                :wrap="secretEditWrap"
                :line-numbers="secretEditLineNumbers"
                height="360px"
                class="k8s-detail-box k8s-detail-box--fill"
              />
            </div>
          </div>
        </div>
      </el-card>

      <div class="edit-drawer-footer">
        <el-button @click="secretEditVisible = false">取消</el-button>
        <el-button type="primary" :loading="secretEditSaving" :disabled="!isSecretEditChanged" @click="saveEditSecret">保存</el-button>
      </div>
    </el-form>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Expand, Fold, Moon, Search, Sunny } from '@element-plus/icons-vue'

import * as k8sApi from '@/features/k8s/api/k8s'
import { getRowNamespace } from '@/features/k8s/pages/ClusterManageView.utils'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'

const props = defineProps<{
  clusterId: number
  clusterName: string
  editorTheme: 'auto' | 'light' | 'dark'
  editorThemeEffectiveDark: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle-editor-theme'): void
  (e: 'saved'): void
}>()

function parseLabelsText(text: string): Record<string, string> {
  const raw = String(text ?? '').trim()
  if (!raw) return {}
  let value: any
  try {
    value = JSON.parse(raw)
  } catch {
    throw new Error('Labels JSON 格式错误')
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  const out: Record<string, string> = {}
  for (const [key, item] of Object.entries(value as Record<string, unknown>)) {
    const normalizedKey = String(key ?? '').trim()
    if (!normalizedKey) continue
    out[normalizedKey] = String(item ?? '')
  }
  return out
}

function normalizeIntOrNull(value: any): number | null {
  if (value == null) return null
  const normalized = Number(value)
  if (!Number.isFinite(normalized)) return null
  return Math.trunc(normalized)
}

function normalizeRecord(record: Record<string, string>): string {
  const entries = Object.entries(record ?? {})
    .map(([key, value]) => [String(key).trim(), String(value ?? '')] as const)
    .filter(([key]) => Boolean(key))
    .sort(([left], [right]) => left.localeCompare(right))
  return JSON.stringify(Object.fromEntries(entries))
}

function parseStringOrNullMapText(text: string, title: string): Record<string, string | null> {
  const raw = String(text ?? '').trim()
  if (!raw) return {}
  let value: any
  try {
    value = JSON.parse(raw)
  } catch {
    throw new Error(`${title} JSON 格式错误`)
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  const out: Record<string, string | null> = {}
  for (const [key, item] of Object.entries(value as Record<string, unknown>)) {
    const normalizedKey = String(key ?? '').trim()
    if (!normalizedKey) continue
    out[normalizedKey] = item === null ? null : String(item ?? '')
  }
  return out
}

function normalizeNullableRecord(record: Record<string, string | null>): string {
  const entries = Object.entries(record ?? {})
    .map(([key, value]) => [String(key).trim(), value === null ? null : String(value ?? '')] as const)
    .filter(([key]) => Boolean(key))
    .sort(([left], [right]) => left.localeCompare(right))
  return JSON.stringify(Object.fromEntries(entries))
}

function buildPatchMap(orig: Record<string, string | null>, next: Record<string, string | null>): Record<string, string | null> | undefined {
  const keys = new Set<string>([...Object.keys(orig ?? {}), ...Object.keys(next ?? {})])
  const out: Record<string, string | null> = {}
  for (const key of keys) {
    const origValue = orig?.[key]
    const hasNext = Object.prototype.hasOwnProperty.call(next ?? {}, key)
    if (!hasNext) {
      if (origValue !== undefined) out[key] = null
      continue
    }
    const nextValue = next?.[key]
    const normalizedNext = nextValue === null ? null : String(nextValue ?? '')
    const normalizedOrig = origValue === null ? null : origValue !== undefined ? String(origValue ?? '') : undefined
    if (normalizedOrig === undefined && normalizedNext === null) {
      out[key] = null
      continue
    }
    if (normalizedOrig !== normalizedNext) out[key] = normalizedNext
  }
  return Object.keys(out).length ? out : undefined
}

const codeMirrorExpose = {
  openSearch: () => undefined,
  foldAll: () => undefined,
  unfoldAll: () => undefined
}

type CodeMirrorExpose = typeof codeMirrorExpose

const jobEditVisible = ref(false)
const jobEditSaving = ref(false)
const jobEditNamespace = ref('')
const jobEditName = ref('')
const jobEditLabelsText = ref('{}')
const jobEditParallelism = ref<number | null>(null)
const jobEditCompletions = ref<number | null>(null)
const jobEditBackoffLimit = ref<number | null>(null)
const jobEditTtlSecondsAfterFinished = ref<number | null>(null)
const jobEditOrig = ref({ labels: {}, parallelism: null, completions: null, backoffLimit: null, ttlSecondsAfterFinished: null } as {
  labels: Record<string, string>
  parallelism: number | null
  completions: number | null
  backoffLimit: number | null
  ttlSecondsAfterFinished: number | null
})

const isJobLabelsChanged = computed(() => {
  try {
    return normalizeRecord(parseLabelsText(jobEditLabelsText.value)) !== normalizeRecord(jobEditOrig.value.labels)
  } catch {
    return true
  }
})
const isJobParallelismChanged = computed(() => normalizeIntOrNull(jobEditParallelism.value) !== jobEditOrig.value.parallelism)
const isJobCompletionsChanged = computed(() => normalizeIntOrNull(jobEditCompletions.value) !== jobEditOrig.value.completions)
const isJobBackoffLimitChanged = computed(() => normalizeIntOrNull(jobEditBackoffLimit.value) !== jobEditOrig.value.backoffLimit)
const isJobTtlChanged = computed(() => normalizeIntOrNull(jobEditTtlSecondsAfterFinished.value) !== jobEditOrig.value.ttlSecondsAfterFinished)
const isJobEditChanged = computed(() => isJobLabelsChanged.value || isJobParallelismChanged.value || isJobCompletionsChanged.value || isJobBackoffLimitChanged.value || isJobTtlChanged.value)

function openEditJob(row: any) {
  const namespace = getRowNamespace(row)
  if (!props.clusterId || !namespace) return
  const name = String(row?.metadata?.name ?? '')
  if (!name) return
  jobEditOrig.value = {
    labels: (row?.metadata?.labels ?? {}) as Record<string, string>,
    parallelism: row?.spec?.parallelism != null ? normalizeIntOrNull(row.spec.parallelism) : null,
    completions: row?.spec?.completions != null ? normalizeIntOrNull(row.spec.completions) : null,
    backoffLimit: row?.spec?.backoffLimit != null ? normalizeIntOrNull(row.spec.backoffLimit) : null,
    ttlSecondsAfterFinished: row?.spec?.ttlSecondsAfterFinished != null ? normalizeIntOrNull(row.spec.ttlSecondsAfterFinished) : null
  }
  jobEditNamespace.value = namespace
  jobEditName.value = name
  jobEditLabelsText.value = JSON.stringify(row?.metadata?.labels ?? {}, null, 2)
  jobEditParallelism.value = row?.spec?.parallelism != null ? Number(row.spec.parallelism) : null
  jobEditCompletions.value = row?.spec?.completions != null ? Number(row.spec.completions) : null
  jobEditBackoffLimit.value = row?.spec?.backoffLimit != null ? Number(row.spec.backoffLimit) : null
  jobEditTtlSecondsAfterFinished.value = row?.spec?.ttlSecondsAfterFinished != null ? Number(row.spec.ttlSecondsAfterFinished) : null
  jobEditVisible.value = true
}

async function saveEditJob() {
  if (!props.clusterId || !jobEditNamespace.value || !jobEditName.value) return
  try {
    jobEditSaving.value = true
    const labels = parseLabelsText(jobEditLabelsText.value)
    const req: any = { namespace: jobEditNamespace.value, name: jobEditName.value, labels }
    if (jobEditParallelism.value != null && Number.isFinite(jobEditParallelism.value)) req.parallelism = Math.max(0, Math.trunc(jobEditParallelism.value))
    if (jobEditCompletions.value != null && Number.isFinite(jobEditCompletions.value)) req.completions = Math.max(0, Math.trunc(jobEditCompletions.value))
    if (jobEditBackoffLimit.value != null && Number.isFinite(jobEditBackoffLimit.value)) req.backoffLimit = Math.max(0, Math.trunc(jobEditBackoffLimit.value))
    if (jobEditTtlSecondsAfterFinished.value != null && Number.isFinite(jobEditTtlSecondsAfterFinished.value)) req.ttlSecondsAfterFinished = Math.max(0, Math.trunc(jobEditTtlSecondsAfterFinished.value))
    await k8sApi.editJob(props.clusterId, req)
    notifySuccess('已保存')
    jobEditVisible.value = false
    emit('saved')
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    jobEditSaving.value = false
  }
}

const cronJobEditVisible = ref(false)
const cronJobEditSaving = ref(false)
const cronJobEditNamespace = ref('')
const cronJobEditName = ref('')
const cronJobEditLabelsText = ref('{}')
const cronJobEditSchedule = ref('')
const cronJobEditSuspend = ref(false)
const cronJobEditConcurrencyPolicy = ref<string | undefined>(undefined)
const cronJobEditSuccessfulJobsHistoryLimit = ref<number | null>(null)
const cronJobEditFailedJobsHistoryLimit = ref<number | null>(null)
const cronJobEditOrig = ref({
  labels: {},
  schedule: '',
  suspend: false,
  concurrencyPolicy: undefined,
  successfulJobsHistoryLimit: null,
  failedJobsHistoryLimit: null
} as {
  labels: Record<string, string>
  schedule: string
  suspend: boolean
  concurrencyPolicy: string | undefined
  successfulJobsHistoryLimit: number | null
  failedJobsHistoryLimit: number | null
})

const isCronJobLabelsChanged = computed(() => {
  try {
    return normalizeRecord(parseLabelsText(cronJobEditLabelsText.value)) !== normalizeRecord(cronJobEditOrig.value.labels)
  } catch {
    return true
  }
})
const isCronJobScheduleChanged = computed(() => String(cronJobEditSchedule.value ?? '').trim() !== String(cronJobEditOrig.value.schedule ?? '').trim())
const isCronJobConcurrencyPolicyChanged = computed(() => (cronJobEditConcurrencyPolicy.value ?? undefined) !== (cronJobEditOrig.value.concurrencyPolicy ?? undefined))
const isCronJobSuccessHistoryChanged = computed(() => normalizeIntOrNull(cronJobEditSuccessfulJobsHistoryLimit.value) !== cronJobEditOrig.value.successfulJobsHistoryLimit)
const isCronJobFailedHistoryChanged = computed(() => normalizeIntOrNull(cronJobEditFailedJobsHistoryLimit.value) !== cronJobEditOrig.value.failedJobsHistoryLimit)
const isCronJobEditChanged = computed(() => isCronJobLabelsChanged.value || isCronJobScheduleChanged.value || cronJobEditSuspend.value !== cronJobEditOrig.value.suspend || isCronJobConcurrencyPolicyChanged.value || isCronJobSuccessHistoryChanged.value || isCronJobFailedHistoryChanged.value)

function openEditCronJob(row: any) {
  const namespace = getRowNamespace(row)
  if (!props.clusterId || !namespace) return
  const name = String(row?.metadata?.name ?? '')
  if (!name) return
  cronJobEditOrig.value = {
    labels: (row?.metadata?.labels ?? {}) as Record<string, string>,
    schedule: String(row?.spec?.schedule ?? '').trim(),
    suspend: Boolean(row?.spec?.suspend),
    concurrencyPolicy: row?.spec?.concurrencyPolicy != null ? String(row.spec.concurrencyPolicy) : undefined,
    successfulJobsHistoryLimit: row?.spec?.successfulJobsHistoryLimit != null ? normalizeIntOrNull(row.spec.successfulJobsHistoryLimit) : null,
    failedJobsHistoryLimit: row?.spec?.failedJobsHistoryLimit != null ? normalizeIntOrNull(row.spec.failedJobsHistoryLimit) : null
  }
  cronJobEditNamespace.value = namespace
  cronJobEditName.value = name
  cronJobEditLabelsText.value = JSON.stringify(row?.metadata?.labels ?? {}, null, 2)
  cronJobEditSchedule.value = String(row?.spec?.schedule ?? '').trim()
  cronJobEditSuspend.value = Boolean(row?.spec?.suspend)
  cronJobEditConcurrencyPolicy.value = row?.spec?.concurrencyPolicy != null ? String(row.spec.concurrencyPolicy) : undefined
  cronJobEditSuccessfulJobsHistoryLimit.value = row?.spec?.successfulJobsHistoryLimit != null ? Number(row.spec.successfulJobsHistoryLimit) : null
  cronJobEditFailedJobsHistoryLimit.value = row?.spec?.failedJobsHistoryLimit != null ? Number(row.spec.failedJobsHistoryLimit) : null
  cronJobEditVisible.value = true
}

async function saveEditCronJob() {
  if (!props.clusterId || !cronJobEditNamespace.value || !cronJobEditName.value) return
  try {
    cronJobEditSaving.value = true
    const labels = parseLabelsText(cronJobEditLabelsText.value)
    const req: any = {
      namespace: cronJobEditNamespace.value,
      name: cronJobEditName.value,
      labels,
      schedule: cronJobEditSchedule.value.trim(),
      suspend: cronJobEditSuspend.value
    }
    if (cronJobEditConcurrencyPolicy.value) req.concurrencyPolicy = cronJobEditConcurrencyPolicy.value
    if (cronJobEditSuccessfulJobsHistoryLimit.value != null && Number.isFinite(cronJobEditSuccessfulJobsHistoryLimit.value)) req.successfulJobsHistoryLimit = Math.max(0, Math.trunc(cronJobEditSuccessfulJobsHistoryLimit.value))
    if (cronJobEditFailedJobsHistoryLimit.value != null && Number.isFinite(cronJobEditFailedJobsHistoryLimit.value)) req.failedJobsHistoryLimit = Math.max(0, Math.trunc(cronJobEditFailedJobsHistoryLimit.value))
    await k8sApi.editCronJob(props.clusterId, req)
    notifySuccess('已保存')
    cronJobEditVisible.value = false
    emit('saved')
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    cronJobEditSaving.value = false
  }
}

const serviceEditVisible = ref(false)
const serviceEditSaving = ref(false)
const serviceEditNamespace = ref('')
const serviceEditName = ref('')
const serviceEditType = ref<string | undefined>(undefined)
const serviceEditLabelsText = ref('{}')
const serviceEditAnnotationsText = ref('{}')
const serviceEditSelectorText = ref('{}')
const serviceEditLabelsOriginalText = ref('{}')
const serviceEditAnnotationsOriginalText = ref('{}')
const serviceEditSelectorOriginalText = ref('{}')
const serviceEditWrap = ref(true)
const serviceEditLineNumbers = ref(true)
const serviceEditShowDiff = ref(false)
const serviceEditLabelsViewerRef = ref<CodeMirrorExpose | null>(null)
const serviceEditAnnotationsViewerRef = ref<CodeMirrorExpose | null>(null)
const serviceEditSelectorViewerRef = ref<CodeMirrorExpose | null>(null)
const serviceEditOrig = ref({ type: undefined, labels: {}, annotations: {}, selector: {} } as {
  type: string | undefined
  labels: Record<string, string | null>
  annotations: Record<string, string | null>
  selector: Record<string, string | null>
})

const isServiceTypeChanged = computed(() => (serviceEditType.value ?? undefined) !== (serviceEditOrig.value.type ?? undefined))
const isServiceLabelsChanged = computed(() => {
  try {
    return normalizeNullableRecord(parseStringOrNullMapText(serviceEditLabelsText.value, 'Labels')) !== normalizeNullableRecord(serviceEditOrig.value.labels)
  } catch {
    return true
  }
})
const isServiceAnnotationsChanged = computed(() => {
  try {
    return normalizeNullableRecord(parseStringOrNullMapText(serviceEditAnnotationsText.value, 'Annotations')) !== normalizeNullableRecord(serviceEditOrig.value.annotations)
  } catch {
    return true
  }
})
const isServiceSelectorChanged = computed(() => {
  try {
    return normalizeNullableRecord(parseStringOrNullMapText(serviceEditSelectorText.value, 'Selector')) !== normalizeNullableRecord(serviceEditOrig.value.selector)
  } catch {
    return true
  }
})
const isServiceEditChanged = computed(() => isServiceTypeChanged.value || isServiceLabelsChanged.value || isServiceAnnotationsChanged.value || isServiceSelectorChanged.value)

function openServiceEditLabelsSearch() { serviceEditLabelsViewerRef.value?.openSearch() }
function foldServiceEditLabelsAll() { serviceEditLabelsViewerRef.value?.foldAll() }
function unfoldServiceEditLabelsAll() { serviceEditLabelsViewerRef.value?.unfoldAll() }
function openServiceEditAnnotationsSearch() { serviceEditAnnotationsViewerRef.value?.openSearch() }
function foldServiceEditAnnotationsAll() { serviceEditAnnotationsViewerRef.value?.foldAll() }
function unfoldServiceEditAnnotationsAll() { serviceEditAnnotationsViewerRef.value?.unfoldAll() }
function openServiceEditSelectorSearch() { serviceEditSelectorViewerRef.value?.openSearch() }
function foldServiceEditSelectorAll() { serviceEditSelectorViewerRef.value?.foldAll() }
function unfoldServiceEditSelectorAll() { serviceEditSelectorViewerRef.value?.unfoldAll() }

function openEditService(row: any) {
  const namespace = getRowNamespace(row)
  if (!props.clusterId || !namespace) return
  const name = String(row?.metadata?.name ?? '')
  if (!name) return
  const labels = (row?.metadata?.labels ?? {}) as Record<string, unknown>
  const annotations = (row?.metadata?.annotations ?? {}) as Record<string, unknown>
  const selector = (row?.spec?.selector ?? {}) as Record<string, unknown>
  const type = row?.spec?.type != null ? String(row.spec.type).trim() : undefined
  serviceEditOrig.value = {
    type: type || undefined,
    labels: Object.fromEntries(Object.entries(labels).map(([key, value]) => [String(key), value == null ? '' : String(value)])),
    annotations: Object.fromEntries(Object.entries(annotations).map(([key, value]) => [String(key), value == null ? '' : String(value)])),
    selector: Object.fromEntries(Object.entries(selector).map(([key, value]) => [String(key), value == null ? '' : String(value)]))
  }
  serviceEditNamespace.value = namespace
  serviceEditName.value = name
  serviceEditType.value = type || undefined
  serviceEditLabelsOriginalText.value = JSON.stringify(labels ?? {}, null, 2)
  serviceEditAnnotationsOriginalText.value = JSON.stringify(annotations ?? {}, null, 2)
  serviceEditSelectorOriginalText.value = JSON.stringify(selector ?? {}, null, 2)
  serviceEditLabelsText.value = serviceEditLabelsOriginalText.value
  serviceEditAnnotationsText.value = serviceEditAnnotationsOriginalText.value
  serviceEditSelectorText.value = serviceEditSelectorOriginalText.value
  serviceEditShowDiff.value = false
  serviceEditVisible.value = true
}

async function saveEditService() {
  if (!props.clusterId || !serviceEditNamespace.value || !serviceEditName.value) return
  try {
    serviceEditSaving.value = true
    const labels = parseStringOrNullMapText(serviceEditLabelsText.value, 'Labels')
    const annotations = parseStringOrNullMapText(serviceEditAnnotationsText.value, 'Annotations')
    const selector = parseStringOrNullMapText(serviceEditSelectorText.value, 'Selector')
    const labelsPatch = buildPatchMap(serviceEditOrig.value.labels, labels)
    const annotationsPatch = buildPatchMap(serviceEditOrig.value.annotations, annotations)
    const selectorPatch = buildPatchMap(serviceEditOrig.value.selector, selector)
    const typePatch = isServiceTypeChanged.value ? (serviceEditType.value ?? '') : undefined
    if (!labelsPatch && !annotationsPatch && !selectorPatch && typePatch === undefined) {
      notifySuccess('未修改')
      serviceEditVisible.value = false
      return
    }
    const req: any = { namespace: serviceEditNamespace.value, name: serviceEditName.value }
    if (labelsPatch) req.labels = labelsPatch
    if (annotationsPatch) req.annotations = annotationsPatch
    if (selectorPatch) req.selector = selectorPatch
    if (typePatch !== undefined) req.type = typePatch
    await k8sApi.editService(props.clusterId, req)
    notifySuccess('已保存')
    serviceEditVisible.value = false
    emit('saved')
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    serviceEditSaving.value = false
  }
}

const ingressEditVisible = ref(false)
const ingressEditSaving = ref(false)
const ingressEditNamespace = ref('')
const ingressEditName = ref('')
const ingressEditClassName = ref('')
const ingressEditLabelsText = ref('{}')
const ingressEditAnnotationsText = ref('{}')
const ingressEditLabelsOriginalText = ref('{}')
const ingressEditAnnotationsOriginalText = ref('{}')
const ingressEditWrap = ref(true)
const ingressEditLineNumbers = ref(true)
const ingressEditShowDiff = ref(false)
const ingressEditLabelsViewerRef = ref<CodeMirrorExpose | null>(null)
const ingressEditAnnotationsViewerRef = ref<CodeMirrorExpose | null>(null)
const ingressEditOrig = ref({ className: '', labels: {}, annotations: {} } as {
  className: string
  labels: Record<string, string | null>
  annotations: Record<string, string | null>
})

const isIngressClassChanged = computed(() => String(ingressEditClassName.value ?? '').trim() !== String(ingressEditOrig.value.className ?? '').trim())
const isIngressLabelsChanged = computed(() => {
  try {
    return normalizeNullableRecord(parseStringOrNullMapText(ingressEditLabelsText.value, 'Labels')) !== normalizeNullableRecord(ingressEditOrig.value.labels)
  } catch {
    return true
  }
})
const isIngressAnnotationsChanged = computed(() => {
  try {
    return normalizeNullableRecord(parseStringOrNullMapText(ingressEditAnnotationsText.value, 'Annotations')) !== normalizeNullableRecord(ingressEditOrig.value.annotations)
  } catch {
    return true
  }
})
const isIngressEditChanged = computed(() => isIngressClassChanged.value || isIngressLabelsChanged.value || isIngressAnnotationsChanged.value)

function openIngressEditLabelsSearch() { ingressEditLabelsViewerRef.value?.openSearch() }
function foldIngressEditLabelsAll() { ingressEditLabelsViewerRef.value?.foldAll() }
function unfoldIngressEditLabelsAll() { ingressEditLabelsViewerRef.value?.unfoldAll() }
function openIngressEditAnnotationsSearch() { ingressEditAnnotationsViewerRef.value?.openSearch() }
function foldIngressEditAnnotationsAll() { ingressEditAnnotationsViewerRef.value?.foldAll() }
function unfoldIngressEditAnnotationsAll() { ingressEditAnnotationsViewerRef.value?.unfoldAll() }

function openEditIngress(row: any) {
  const namespace = getRowNamespace(row)
  if (!props.clusterId || !namespace) return
  const name = String(row?.metadata?.name ?? '')
  if (!name) return
  const labels = (row?.metadata?.labels ?? {}) as Record<string, unknown>
  const annotations = (row?.metadata?.annotations ?? {}) as Record<string, unknown>
  const className = row?.spec?.ingressClassName != null ? String(row.spec.ingressClassName).trim() : ''
  ingressEditOrig.value = {
    className,
    labels: Object.fromEntries(Object.entries(labels).map(([key, value]) => [String(key), value == null ? '' : String(value)])),
    annotations: Object.fromEntries(Object.entries(annotations).map(([key, value]) => [String(key), value == null ? '' : String(value)]))
  }
  ingressEditNamespace.value = namespace
  ingressEditName.value = name
  ingressEditClassName.value = className
  ingressEditLabelsOriginalText.value = JSON.stringify(labels ?? {}, null, 2)
  ingressEditAnnotationsOriginalText.value = JSON.stringify(annotations ?? {}, null, 2)
  ingressEditLabelsText.value = ingressEditLabelsOriginalText.value
  ingressEditAnnotationsText.value = ingressEditAnnotationsOriginalText.value
  ingressEditShowDiff.value = false
  ingressEditVisible.value = true
}

async function saveEditIngress() {
  if (!props.clusterId || !ingressEditNamespace.value || !ingressEditName.value) return
  try {
    ingressEditSaving.value = true
    const labels = parseStringOrNullMapText(ingressEditLabelsText.value, 'Labels')
    const annotations = parseStringOrNullMapText(ingressEditAnnotationsText.value, 'Annotations')
    const labelsPatch = buildPatchMap(ingressEditOrig.value.labels, labels)
    const annotationsPatch = buildPatchMap(ingressEditOrig.value.annotations, annotations)
    const classPatch = isIngressClassChanged.value ? ingressEditClassName.value.trim() : undefined
    if (!labelsPatch && !annotationsPatch && classPatch === undefined) {
      notifySuccess('未修改')
      ingressEditVisible.value = false
      return
    }
    const req: any = { namespace: ingressEditNamespace.value, name: ingressEditName.value }
    if (labelsPatch) req.labels = labelsPatch
    if (annotationsPatch) req.annotations = annotationsPatch
    if (classPatch !== undefined) req.ingressClassName = classPatch
    await k8sApi.editIngress(props.clusterId, req)
    notifySuccess('已保存')
    ingressEditVisible.value = false
    emit('saved')
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    ingressEditSaving.value = false
  }
}

const ingressClassEditVisible = ref(false)
const ingressClassEditSaving = ref(false)
const ingressClassEditName = ref('')
const ingressClassEditController = ref('')
const ingressClassEditDefault = ref(false)
const ingressClassEditLabelsText = ref('{}')
const ingressClassEditAnnotationsText = ref('{}')
const ingressClassEditLabelsOriginalText = ref('{}')
const ingressClassEditAnnotationsOriginalText = ref('{}')
const ingressClassEditWrap = ref(true)
const ingressClassEditLineNumbers = ref(true)
const ingressClassEditShowDiff = ref(false)
const ingressClassEditLabelsViewerRef = ref<CodeMirrorExpose | null>(null)
const ingressClassEditAnnotationsViewerRef = ref<CodeMirrorExpose | null>(null)
const ingressClassEditOrig = ref({ controller: '', isDefault: false, labels: {}, annotations: {} } as {
  controller: string
  isDefault: boolean
  labels: Record<string, string | null>
  annotations: Record<string, string | null>
})

const isIngressClassControllerChanged = computed(() => String(ingressClassEditController.value ?? '').trim() !== String(ingressClassEditOrig.value.controller ?? '').trim())
const isIngressClassDefaultChanged = computed(() => Boolean(ingressClassEditDefault.value) !== Boolean(ingressClassEditOrig.value.isDefault))
const isIngressClassLabelsChanged = computed(() => {
  try {
    return normalizeNullableRecord(parseStringOrNullMapText(ingressClassEditLabelsText.value, 'Labels')) !== normalizeNullableRecord(ingressClassEditOrig.value.labels)
  } catch {
    return true
  }
})
const isIngressClassAnnotationsChanged = computed(() => {
  try {
    return normalizeNullableRecord(parseStringOrNullMapText(ingressClassEditAnnotationsText.value, 'Annotations')) !== normalizeNullableRecord(ingressClassEditOrig.value.annotations)
  } catch {
    return true
  }
})
const isIngressClassEditChanged = computed(() => isIngressClassControllerChanged.value || isIngressClassDefaultChanged.value || isIngressClassLabelsChanged.value || isIngressClassAnnotationsChanged.value)

function openIngressClassEditLabelsSearch() { ingressClassEditLabelsViewerRef.value?.openSearch() }
function foldIngressClassEditLabelsAll() { ingressClassEditLabelsViewerRef.value?.foldAll() }
function unfoldIngressClassEditLabelsAll() { ingressClassEditLabelsViewerRef.value?.unfoldAll() }
function openIngressClassEditAnnotationsSearch() { ingressClassEditAnnotationsViewerRef.value?.openSearch() }
function foldIngressClassEditAnnotationsAll() { ingressClassEditAnnotationsViewerRef.value?.foldAll() }
function unfoldIngressClassEditAnnotationsAll() { ingressClassEditAnnotationsViewerRef.value?.unfoldAll() }

function openEditIngressClass(row: any) {
  if (!props.clusterId) return
  const name = String(row?.metadata?.name ?? '')
  if (!name) return
  const labels = (row?.metadata?.labels ?? {}) as Record<string, unknown>
  const annotations = (row?.metadata?.annotations ?? {}) as Record<string, unknown>
  const controller = row?.spec?.controller != null ? String(row.spec.controller).trim() : ''
  const rawAnnotations = row?.metadata?.annotations
  const defaultValue = rawAnnotations?.['ingressclass.kubernetes.io/is-default-class'] ?? rawAnnotations?.['ingressclass.k8s.io/is-default-class'] ?? rawAnnotations?.['is-default-class']
  const isDefault = String(defaultValue ?? '').toLowerCase() === 'true'
  ingressClassEditOrig.value = {
    controller,
    isDefault,
    labels: Object.fromEntries(Object.entries(labels).map(([key, value]) => [String(key), value == null ? '' : String(value)])),
    annotations: Object.fromEntries(Object.entries(annotations).map(([key, value]) => [String(key), value == null ? '' : String(value)]))
  }
  ingressClassEditName.value = name
  ingressClassEditController.value = controller
  ingressClassEditDefault.value = isDefault
  ingressClassEditLabelsOriginalText.value = JSON.stringify(labels ?? {}, null, 2)
  ingressClassEditAnnotationsOriginalText.value = JSON.stringify(annotations ?? {}, null, 2)
  ingressClassEditLabelsText.value = ingressClassEditLabelsOriginalText.value
  ingressClassEditAnnotationsText.value = ingressClassEditAnnotationsOriginalText.value
  ingressClassEditShowDiff.value = false
  ingressClassEditVisible.value = true
}

async function saveEditIngressClass() {
  if (!props.clusterId || !ingressClassEditName.value) return
  try {
    ingressClassEditSaving.value = true
    const labels = parseStringOrNullMapText(ingressClassEditLabelsText.value, 'Labels')
    const annotations = parseStringOrNullMapText(ingressClassEditAnnotationsText.value, 'Annotations')
    const labelsPatch = buildPatchMap(ingressClassEditOrig.value.labels, labels)
    const annotationsPatch = buildPatchMap(ingressClassEditOrig.value.annotations, annotations)
    const controllerPatch = isIngressClassControllerChanged.value ? ingressClassEditController.value.trim() : undefined
    const defaultPatch = isIngressClassDefaultChanged.value ? Boolean(ingressClassEditDefault.value) : undefined
    if (!labelsPatch && !annotationsPatch && controllerPatch === undefined && defaultPatch === undefined) {
      notifySuccess('未修改')
      ingressClassEditVisible.value = false
      return
    }
    const req: any = { name: ingressClassEditName.value }
    if (labelsPatch) req.labels = labelsPatch
    if (annotationsPatch) req.annotations = annotationsPatch
    if (controllerPatch !== undefined) req.controller = controllerPatch
    if (defaultPatch !== undefined) req.isDefault = defaultPatch
    await k8sApi.editIngressClass(props.clusterId, req)
    notifySuccess('已保存')
    ingressClassEditVisible.value = false
    emit('saved')
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    ingressClassEditSaving.value = false
  }
}

const configMapEditVisible = ref(false)
const configMapEditSaving = ref(false)
const configMapEditNamespace = ref('')
const configMapEditName = ref('')
const configMapEditLabelsText = ref('{}')
const configMapEditDataText = ref('{}')
const configMapEditLabelsOriginalText = ref('{}')
const configMapEditDataOriginalText = ref('{}')
const configMapEditWrap = ref(true)
const configMapEditLineNumbers = ref(true)
const configMapEditShowDiff = ref(false)
const configMapEditLabelsViewerRef = ref<CodeMirrorExpose | null>(null)
const configMapEditDataViewerRef = ref<CodeMirrorExpose | null>(null)
const configMapEditOrig = ref({ labels: {}, data: {} } as { labels: Record<string, string | null>; data: Record<string, string | null> })

const isConfigMapLabelsChanged = computed(() => {
  try {
    return normalizeNullableRecord(parseStringOrNullMapText(configMapEditLabelsText.value, 'Labels')) !== normalizeNullableRecord(configMapEditOrig.value.labels)
  } catch {
    return true
  }
})
const isConfigMapDataChanged = computed(() => {
  try {
    return normalizeNullableRecord(parseStringOrNullMapText(configMapEditDataText.value, 'Data')) !== normalizeNullableRecord(configMapEditOrig.value.data)
  } catch {
    return true
  }
})
const isConfigMapEditChanged = computed(() => isConfigMapLabelsChanged.value || isConfigMapDataChanged.value)

function openConfigMapEditLabelsSearch() { configMapEditLabelsViewerRef.value?.openSearch() }
function foldConfigMapEditLabelsAll() { configMapEditLabelsViewerRef.value?.foldAll() }
function unfoldConfigMapEditLabelsAll() { configMapEditLabelsViewerRef.value?.unfoldAll() }
function openConfigMapEditDataSearch() { configMapEditDataViewerRef.value?.openSearch() }
function foldConfigMapEditDataAll() { configMapEditDataViewerRef.value?.foldAll() }
function unfoldConfigMapEditDataAll() { configMapEditDataViewerRef.value?.unfoldAll() }

function openEditConfigMap(row: any) {
  const namespace = getRowNamespace(row)
  if (!props.clusterId || !namespace) return
  const name = String(row?.metadata?.name ?? '')
  if (!name) return
  const labels = (row?.metadata?.labels ?? {}) as Record<string, unknown>
  const data = (row?.data ?? {}) as Record<string, unknown>
  configMapEditOrig.value = {
    labels: Object.fromEntries(Object.entries(labels).map(([key, value]) => [String(key), value == null ? '' : String(value)])),
    data: Object.fromEntries(Object.entries(data).map(([key, value]) => [String(key), value == null ? '' : String(value)]))
  }
  configMapEditNamespace.value = namespace
  configMapEditName.value = name
  configMapEditLabelsOriginalText.value = JSON.stringify(labels ?? {}, null, 2)
  configMapEditDataOriginalText.value = JSON.stringify(data ?? {}, null, 2)
  configMapEditLabelsText.value = configMapEditLabelsOriginalText.value
  configMapEditDataText.value = configMapEditDataOriginalText.value
  configMapEditShowDiff.value = false
  configMapEditVisible.value = true
}

async function saveEditConfigMap() {
  if (!props.clusterId || !configMapEditNamespace.value || !configMapEditName.value) return
  try {
    configMapEditSaving.value = true
    const labels = parseStringOrNullMapText(configMapEditLabelsText.value, 'Labels')
    const data = parseStringOrNullMapText(configMapEditDataText.value, 'Data')
    const labelsPatch = buildPatchMap(configMapEditOrig.value.labels, labels)
    const dataPatch = buildPatchMap(configMapEditOrig.value.data, data)
    if (!labelsPatch && !dataPatch) {
      notifySuccess('未修改')
      configMapEditVisible.value = false
      return
    }
    const req: any = { namespace: configMapEditNamespace.value, name: configMapEditName.value }
    if (labelsPatch) req.labels = labelsPatch
    if (dataPatch) req.data = dataPatch
    await k8sApi.editConfigMap(props.clusterId, req)
    notifySuccess('已保存')
    configMapEditVisible.value = false
    emit('saved')
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    configMapEditSaving.value = false
  }
}

const secretEditVisible = ref(false)
const secretEditSaving = ref(false)
const secretEditNamespace = ref('')
const secretEditName = ref('')
const secretEditType = ref('')
const secretEditLabelsText = ref('{}')
const secretEditDataText = ref('{}')
const secretEditLabelsOriginalText = ref('{}')
const secretEditDataOriginalText = ref('{}')
const secretEditWrap = ref(true)
const secretEditLineNumbers = ref(true)
const secretEditShowDiff = ref(false)
const secretEditLabelsViewerRef = ref<CodeMirrorExpose | null>(null)
const secretEditDataViewerRef = ref<CodeMirrorExpose | null>(null)
const secretEditOrig = ref({ type: '', labels: {}, data: {} } as { type: string; labels: Record<string, string | null>; data: Record<string, string | null> })

const isSecretTypeChanged = computed(() => String(secretEditType.value ?? '').trim() !== String(secretEditOrig.value.type ?? '').trim())
const isSecretLabelsChanged = computed(() => {
  try {
    return normalizeNullableRecord(parseStringOrNullMapText(secretEditLabelsText.value, 'Labels')) !== normalizeNullableRecord(secretEditOrig.value.labels)
  } catch {
    return true
  }
})
const isSecretDataChanged = computed(() => {
  try {
    return normalizeNullableRecord(parseStringOrNullMapText(secretEditDataText.value, 'Data')) !== normalizeNullableRecord(secretEditOrig.value.data)
  } catch {
    return true
  }
})
const isSecretEditChanged = computed(() => isSecretTypeChanged.value || isSecretLabelsChanged.value || isSecretDataChanged.value)

function openSecretEditLabelsSearch() { secretEditLabelsViewerRef.value?.openSearch() }
function foldSecretEditLabelsAll() { secretEditLabelsViewerRef.value?.foldAll() }
function unfoldSecretEditLabelsAll() { secretEditLabelsViewerRef.value?.unfoldAll() }
function openSecretEditDataSearch() { secretEditDataViewerRef.value?.openSearch() }
function foldSecretEditDataAll() { secretEditDataViewerRef.value?.foldAll() }
function unfoldSecretEditDataAll() { secretEditDataViewerRef.value?.unfoldAll() }

function openEditSecret(row: any) {
  const namespace = getRowNamespace(row)
  if (!props.clusterId || !namespace) return
  const name = String(row?.metadata?.name ?? '')
  if (!name) return
  const labels = (row?.metadata?.labels ?? {}) as Record<string, unknown>
  const data = (row?.data ?? {}) as Record<string, unknown>
  const type = String(row?.type ?? '').trim()
  secretEditOrig.value = {
    type,
    labels: Object.fromEntries(Object.entries(labels).map(([key, value]) => [String(key), value == null ? '' : String(value)])),
    data: Object.fromEntries(Object.entries(data).map(([key, value]) => [String(key), value == null ? '' : String(value)]))
  }
  secretEditNamespace.value = namespace
  secretEditName.value = name
  secretEditType.value = type
  secretEditLabelsOriginalText.value = JSON.stringify(labels ?? {}, null, 2)
  secretEditDataOriginalText.value = JSON.stringify(data ?? {}, null, 2)
  secretEditLabelsText.value = secretEditLabelsOriginalText.value
  secretEditDataText.value = secretEditDataOriginalText.value
  secretEditShowDiff.value = false
  secretEditVisible.value = true
}

async function saveEditSecret() {
  if (!props.clusterId || !secretEditNamespace.value || !secretEditName.value) return
  try {
    secretEditSaving.value = true
    const labels = parseStringOrNullMapText(secretEditLabelsText.value, 'Labels')
    const data = parseStringOrNullMapText(secretEditDataText.value, 'Data')
    const labelsPatch = buildPatchMap(secretEditOrig.value.labels, labels)
    const dataPatch = buildPatchMap(secretEditOrig.value.data, data)
    const typeText = String(secretEditType.value ?? '').trim()
    const typeChanged = typeText !== String(secretEditOrig.value.type ?? '').trim()
    if (!labelsPatch && !dataPatch && !typeChanged) {
      notifySuccess('未修改')
      secretEditVisible.value = false
      return
    }
    const req: any = { namespace: secretEditNamespace.value, name: secretEditName.value }
    if (typeChanged && typeText) req.type = typeText
    if (labelsPatch) req.labels = labelsPatch
    if (dataPatch) req.data = dataPatch
    if (!req.type && !req.labels && !req.data) {
      notifySuccess('未修改')
      secretEditVisible.value = false
      return
    }
    await k8sApi.editSecret(props.clusterId, req)
    notifySuccess('已保存')
    secretEditVisible.value = false
    emit('saved')
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    secretEditSaving.value = false
  }
}

defineExpose({
  openEditJob,
  openEditCronJob,
  openEditService,
  openEditIngress,
  openEditIngressClass,
  openEditConfigMap,
  openEditSecret
})
</script>

<style scoped>
.deployment-edit-form {
  margin: 0;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
}

.edit-section-card {
  border-radius: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: rgba(255, 255, 255, 0.96);
}

:global(html.dark) .edit-section-card {
  border-color: rgba(226, 232, 240, 0.08);
  background: rgba(2, 6, 23, 0.35);
}

.edit-section-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.edit-title-stack {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.edit-section-title {
  font-size: 14px;
  font-weight: 800;
  color: rgba(15, 23, 42, 0.88);
}

:global(html.dark) .edit-section-title {
  color: rgba(226, 232, 240, 0.92);
}

.edit-meta-row,
.base-meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.base-meta-pill {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  border: 1px solid rgba(148, 163, 184, 0.24);
  background: rgba(248, 250, 252, 0.92);
  color: rgba(51, 65, 85, 0.9);
}

:global(html.dark) .base-meta-pill {
  border-color: rgba(226, 232, 240, 0.12);
  background: rgba(15, 23, 42, 0.78);
  color: rgba(226, 232, 240, 0.9);
}

.base-meta-pill--cluster {
  color: rgba(37, 99, 235, 0.92);
}

.base-meta-pill--kind {
  color: rgba(168, 85, 247, 0.92);
}

.base-meta-pill--target {
  color: rgba(16, 185, 129, 0.92);
}

.edit-grid4 {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.edit-kv {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  padding: 12px 14px 12px 16px;
  border-radius: 14px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: rgba(248, 250, 252, 0.85);
}

.edit-kv::before {
  content: '';
  position: absolute;
  left: 0;
  top: 9px;
  bottom: 9px;
  width: 3px;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.48);
}

:global(html.dark) .edit-kv {
  border-color: rgba(226, 232, 240, 0.08);
  background: rgba(2, 6, 23, 0.35);
}

.edit-kv--span4 {
  grid-column: 1 / -1;
}

.edit-kv--span2 {
  grid-column: span 2;
}

.edit-k {
  font-size: 12px;
  font-weight: 800;
  color: rgba(2, 6, 23, 0.62);
}

:global(html.dark) .edit-k {
  color: rgba(226, 232, 240, 0.62);
}

.edit-v {
  min-width: 0;
}

.edit-ro {
  height: 32px;
  line-height: 32px;
  font-weight: 900;
  color: rgba(2, 6, 23, 0.82);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

:global(html.dark) .edit-ro {
  color: rgba(226, 232, 240, 0.9);
}

.edit-field :deep(.el-input__wrapper),
.edit-field :deep(.el-textarea__inner),
.edit-field :deep(.el-select__wrapper),
.edit-field :deep(.el-input-number .el-input__wrapper) {
  border-radius: 10px;
}

.edit-field.is-changed :deep(.el-input__wrapper),
.edit-field.is-changed :deep(.el-textarea__inner),
.edit-field.is-changed :deep(.el-select__wrapper),
.edit-field.is-changed :deep(.el-input-number .el-input__wrapper) {
  box-shadow: 0 0 0 1px rgba(245, 158, 11, 0.6) inset;
}

.edit-drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 14px;
}

@media (max-width: 1280px) {
  .edit-grid4 {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .edit-grid4 {
    grid-template-columns: 1fr;
  }

  .edit-kv--span2,
  .edit-kv--span4 {
    grid-column: auto;
  }
}
</style>
