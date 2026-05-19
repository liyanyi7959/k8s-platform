<template>
  <el-drawer
    v-model="editVisible"
    :title="`编辑 ${workloadKind}`"
    size="72%"
    class="deployment-edit-drawer"
    :close-on-click-modal="false"
    :with-header="true"
  >
    <el-form v-if="editForm" :model="editForm" label-width="0" class="deployment-edit-form">
      <el-card shadow="never" class="edit-section-card edit-section-card--base">
        <template #header>
          <div class="edit-section-title-row">
            <div class="edit-title-stack">
              <div class="edit-section-title">基础信息</div>
              <div class="edit-meta-row">
                <div class="base-meta-row">
                  <span class="base-meta-pill base-meta-pill--cluster">cluster={{ clusterName || (clusterId != null ? String(clusterId) : '-') }}</span>
                  <span class="base-meta-pill base-meta-pill--kind">{{ workloadKind }}</span>
                  <span class="base-meta-pill base-meta-pill--target">{{ editTarget?.namespace }}/{{ editTarget?.name }}</span>
                </div>
              </div>
            </div>
            <el-tag size="small" :type="isEditChanged() ? 'warning' : 'info'" effect="light">
              {{ isEditChanged() ? '已修改' : '未修改' }}
            </el-tag>
          </div>
        </template>
        <div class="edit-grid4">
          <div class="edit-kv edit-kv--cluster">
            <div class="edit-k">集群</div>
            <div class="edit-v">
              <div class="edit-ro mono">{{ clusterName }}</div>
            </div>
          </div>
          <div class="edit-kv edit-kv--name">
            <div class="edit-k">名称</div>
            <div class="edit-v">
              <div class="edit-ro mono edit-ro--name">{{ editTarget?.name }}</div>
            </div>
          </div>
          <div class="edit-kv edit-kv--namespace">
            <div class="edit-k">命名空间</div>
            <div class="edit-v">
              <div class="edit-ro mono">{{ editTarget?.namespace }}</div>
            </div>
          </div>
          <div v-if="workloadKind !== 'DaemonSet'" class="edit-kv edit-kv--replicas">
            <div class="edit-k">副本数</div>
            <div class="edit-v">
              <el-input-number
                v-model="editForm.replicas"
                size="small"
                class="edit-field"
                :class="{ 'is-changed': isReplicasChanged() }"
                :controls="false"
                :min="0"
                :max="500"
              />
            </div>
          </div>
        </div>
      </el-card>

      <el-card shadow="never" class="edit-section-card edit-section-card--meta" style="margin-top: 10px">
        <template #header>
          <div class="edit-section-title-row">
            <div class="edit-title-stack">
              <div class="edit-section-title">标签 / 污点容忍</div>
            </div>
            <el-tag size="small" :type="isMetaChanged() ? 'warning' : 'info'" effect="light">
              {{ isMetaChanged() ? '已修改' : '未修改' }}
            </el-tag>
          </div>
        </template>
        <div class="meta-stack">
          <div class="meta-block meta-block--labels">
            <div class="meta-block-head">
              <div class="meta-block-title">
                <span class="meta-block-mark meta-block-mark--labels" />
                <span>Labels</span>
              </div>
              <el-button size="small" type="primary" plain :icon="Plus" @click="addLabelRow">添加标签</el-button>
            </div>

            <div v-if="labelsUiMode === 'form'" class="edit-label-grid">
              <div
                v-for="(it, idx) in labelRows"
                :key="`label-${idx}`"
                :class="['edit-label-item', `edit-label-tone-${idx % 6}`]"
              >
                <el-input v-model="it.key" size="small" class="edit-field edit-label-key" :class="{ 'is-changed': isLabelsChanged() }" placeholder="Key" />
                <el-input v-model="it.value" size="small" class="edit-field edit-label-value" :class="{ 'is-changed': isLabelsChanged() }" placeholder="Value" />
                <el-button
                  size="small"
                  type="info"
                  plain
                  circle
                  class="meta-del-btn"
                  aria-label="删除"
                  :icon="Close"
                  @click="removeLabelRow(idx)"
                />
              </div>
            </div>
            <el-input
              v-else
              v-model="editForm.labelsText"
              type="textarea"
              :rows="5"
              class="edit-field"
              :class="{ 'is-changed': isLabelsChanged() }"
              placeholder="{&quot;env&quot;:&quot;dev&quot;}"
            />
          </div>

          <div class="meta-block meta-block--tolerations">
            <div class="meta-block-head">
              <div class="meta-block-title">
                <span class="meta-block-mark meta-block-mark--tolerations" />
                <span>Tolerations</span>
              </div>
              <el-button size="small" type="primary" plain :icon="Plus" @click="addTolerationRow">添加容忍规则</el-button>
            </div>

            <div v-if="tolerationsUiMode === 'form'" class="edit-tol-list">
              <div
                v-for="(t, idx) in tolerationRows"
                :key="`tol-${idx}`"
                :class="['edit-tol-row', `edit-tol-tone-${idx % 6}`]"
              >
                <el-input v-model="t.key" size="small" class="edit-field edit-tol-key" :class="{ 'is-changed': isTolerationsChanged() }" placeholder="Key" />
                <el-select
                  v-model="t.operator"
                  size="small"
                  class="edit-field edit-tol-operator"
                  :class="{ 'is-changed': isTolerationsChanged() }"
                  @change="() => onTolerationOperatorChange(t)"
                >
                  <el-option label="Equal" value="Equal" />
                  <el-option label="Exists" value="Exists" />
                </el-select>
                <el-input
                  v-if="t.operator === 'Equal'"
                  v-model="t.value"
                  size="small"
                  class="edit-field edit-tol-value"
                  :class="{ 'is-changed': isTolerationsChanged() }"
                  placeholder="Value"
                />
                <el-select v-model="t.effect" size="small" clearable class="edit-field edit-tol-effect" :class="{ 'is-changed': isTolerationsChanged() }">
                  <el-option label="NoSchedule" value="NoSchedule" />
                  <el-option label="PreferNoSchedule" value="PreferNoSchedule" />
                  <el-option label="NoExecute" value="NoExecute" />
                </el-select>
                <el-button
                  size="small"
                  type="info"
                  plain
                  circle
                  class="meta-del-btn"
                  aria-label="删除"
                  :icon="Close"
                  @click="removeTolerationRow(idx)"
                />
              </div>
            </div>
            <el-input
              v-else
              v-model="editForm.tolerationsText"
              type="textarea"
              :rows="6"
              class="edit-field"
              :class="{ 'is-changed': isTolerationsChanged() }"
              placeholder="[{&quot;key&quot;:&quot;dedicated&quot;,&quot;operator&quot;:&quot;Equal&quot;,&quot;value&quot;:&quot;gpu&quot;,&quot;effect&quot;:&quot;NoSchedule&quot;}]"
            />
          </div>
        </div>
      </el-card>

      <!-- 更新策略 -->
      <el-card shadow="never" class="edit-section-card edit-section-card--strategy" style="margin-top: 10px">
        <template #header>
          <div class="edit-section-title-row">
            <div class="edit-section-title">更新策略 (Strategy)</div>
            <el-tag size="small" :type="isStrategyChanged() ? 'warning' : 'info'" effect="light">
              {{ isStrategyChanged() ? '已修改' : '未修改' }}
            </el-tag>
          </div>
        </template>
        <div class="edit-grid4">
          <div class="edit-kv">
            <div class="edit-k">类型</div>
            <div class="edit-v">
              <el-radio-group
                v-if="workloadKind === 'Deployment'"
                v-model="editForm!.strategyType"
                size="small"
                class="edit-strategy-radio"
                :class="{ 'is-changed': isStrategyChanged() }"
              >
                <el-radio-button value="RollingUpdate">RollingUpdate</el-radio-button>
                <el-radio-button value="Recreate">Recreate</el-radio-button>
              </el-radio-group>
              <el-select v-else v-model="editForm!.strategyType" size="small" clearable class="edit-field" :class="{ 'is-changed': isStrategyChanged() }">
                <el-option label="RollingUpdate" value="RollingUpdate" />
                <el-option label="Recreate" value="Recreate" />
                <el-option v-if="workloadKind === 'DaemonSet'" label="OnDelete" value="OnDelete" />
              </el-select>
            </div>
          </div>
          <div class="edit-kv" v-if="editForm!.strategyType === 'RollingUpdate'">
            <div class="edit-k">maxSurge</div>
            <div class="edit-v">
              <el-input v-model="editForm!.strategyMaxSurge" size="small" class="edit-field" placeholder="25% 或 1" />
            </div>
          </div>
          <div class="edit-kv" v-if="editForm!.strategyType === 'RollingUpdate'">
            <div class="edit-k">maxUnavailable</div>
            <div class="edit-v">
              <el-input v-model="editForm!.strategyMaxUnavailable" size="small" class="edit-field" placeholder="25% 或 1" />
            </div>
          </div>
        </div>
      </el-card>

      <!-- Pod级别 Volumes -->
      <el-card shadow="never" class="edit-section-card edit-section-card--volumes" style="margin-top: 10px">
        <template #header>
          <div class="edit-section-title-row">
            <div class="edit-section-title">Volumes</div>
            <el-tag size="small" :type="isVolumesChanged() ? 'warning' : 'info'" effect="light">
              {{ isVolumesChanged() ? '已修改' : '未修改' }}
            </el-tag>
          </div>
        </template>
        <el-input
          v-model="editForm!.volumesText"
          type="textarea"
          :autosize="{ minRows: 3, maxRows: 12 }"
          size="small"
          class="edit-field mono"
          :class="{ 'is-changed': isVolumesChanged() }"
          placeholder='[{"name":"data","emptyDir":{}}, {"name":"config","configMap":{"name":"my-cm"}}]'
        />
        <div class="edit-hint">JSON 数组，完整替换 Pod spec.volumes 字段。留空 = 不修改。</div>
      </el-card>

      <div class="edit-container-shell" style="margin-top: 10px">
        <div class="edit-container-nav">
          <div class="edit-container-nav-title">容器基础信息</div>
          <el-scrollbar class="edit-container-nav-scroll">
            <el-space :size="8" wrap class="edit-container-nav-list">
              <el-button
                v-for="c in allContainers"
                :key="containerKey(c.scope, c.name)"
                size="small"
                :type="editActiveContainer === containerKey(c.scope, c.name) ? 'primary' : 'default'"
                :plain="editActiveContainer !== containerKey(c.scope, c.name)"
                class="edit-container-pill"
                @click="editActiveContainer = containerKey(c.scope, c.name)"
              >
                <span class="mono">{{ c.scope === 'initContainers' ? `init/${c.name}` : c.name }}</span>
                <span v-if="isContainerChanged(c)" class="edit-dot edit-dot--changed" />
              </el-button>
            </el-space>
          </el-scrollbar>
        </div>

        <div v-for="c in allContainers" :key="containerKey(c.scope, c.name)" v-show="editActiveContainer === containerKey(c.scope, c.name)">
          <div class="edit-container-pane">
            <el-card shadow="never" class="edit-section-card edit-section-card--container">
              <template #header>
                <div class="edit-section-title-row">
                  <div class="edit-section-title">容器信息</div>
                  <el-space size="6">
                    <el-tag v-if="c.scope === 'initContainers'" size="small" type="info" effect="light">InitContainer</el-tag>
                    <el-tag size="small" :type="isContainerChanged(c) ? 'warning' : 'info'" effect="light">
                      {{ isContainerChanged(c) ? '已修改' : '未修改' }}
                    </el-tag>
                  </el-space>
                </div>
              </template>

              <div class="edit-grid4">
                <div class="edit-kv edit-kv--container-name">
                  <div class="edit-k">容器</div>
                  <div class="edit-v">
                    <div class="edit-ro mono">{{ c.name }}</div>
                  </div>
                </div>

                <div class="edit-kv edit-kv--span2 edit-kv--image">
                  <div class="edit-k">镜像信息</div>
                  <div class="edit-v">
                    <el-input
                      v-model="c.image"
                      size="small"
                      class="edit-field"
                      :class="{ 'is-changed': isImageChanged(c), 'is-invalid': isImageInvalid(c) }"
                      placeholder="例如：nginx:1.27.0"
                    />
                  </div>
                </div>

                <div class="edit-kv edit-kv--pull-policy">
                  <div class="edit-k">拉取策略</div>
                  <div class="edit-v">
                    <el-select v-model="c.imagePullPolicy" size="small" clearable class="edit-field" :class="{ 'is-changed': isImagePullPolicyChanged(c) }">
                      <el-option label="Always" value="Always" />
                      <el-option label="IfNotPresent" value="IfNotPresent" />
                      <el-option label="Never" value="Never" />
                    </el-select>
                  </div>
                </div>

              </div>
            </el-card>

            <el-card shadow="never" class="edit-section-card edit-section-card--resources">
              <template #header>
                <div class="edit-section-title-row">
                  <div class="edit-section-title">Resources</div>
                  <el-tag size="small" :type="isContainerResourcesChanged(c) ? 'warning' : 'info'" effect="light">
                    {{ isContainerResourcesChanged(c) ? '已修改' : '未修改' }}
                  </el-tag>
                </div>
              </template>

              <div class="edit-res-table">
                <div class="edit-res-head" />
                <div class="edit-res-head edit-res-head--cpu">CPU</div>
                <div class="edit-res-head edit-res-head--mem">内存</div>

                <div class="edit-res-rowlabel edit-res-rowlabel--requests">requests</div>
                <div class="edit-res-cell">
                  <el-input
                    v-model="c.requestsCpu"
                    size="small"
                    class="edit-field"
                    :class="{ 'is-changed': isRequestsCpuChanged(c) }"
                    placeholder="例如：100m"
                  />
                </div>
                <div class="edit-res-cell">
                  <el-input
                    v-model="c.requestsMemory"
                    size="small"
                    class="edit-field"
                    :class="{ 'is-changed': isRequestsMemoryChanged(c) }"
                    placeholder="例如：128Mi"
                  />
                </div>

                <div class="edit-res-rowlabel edit-res-rowlabel--limits">limits</div>
                <div class="edit-res-cell">
                  <el-input
                    v-model="c.limitsCpu"
                    size="small"
                    class="edit-field"
                    :class="{ 'is-changed': isLimitsCpuChanged(c) }"
                    placeholder="例如：500m"
                  />
                </div>
                <div class="edit-res-cell">
                  <el-input
                    v-model="c.limitsMemory"
                    size="small"
                    class="edit-field"
                    :class="{ 'is-changed': isLimitsMemoryChanged(c) }"
                    placeholder="例如：256Mi"
                  />
                </div>
              </div>
            </el-card>

            <el-card shadow="never" class="edit-section-card edit-section-card--probes">
              <template #header>
                <div class="edit-section-title-row">
                  <div class="edit-section-title">Probes（仅时序参数）</div>
                  <el-tag size="small" :type="isContainerProbesChanged(c) ? 'warning' : 'info'" effect="light">
                    {{ isContainerProbesChanged(c) ? '已修改' : '未修改' }}
                  </el-tag>
                </div>
              </template>

              <div class="edit-probe-stack">
                <el-card shadow="never" class="edit-section-card edit-section-card--liveness">
                  <template #header>
                    <div class="edit-section-title-row">
                      <div class="edit-section-title">Liveness</div>
                      <el-tag size="small" :type="isProbeChanged(c, 'liveness') ? 'warning' : 'info'" effect="light">
                        {{ isProbeChanged(c, 'liveness') ? '已修改' : '未修改' }}
                      </el-tag>
                    </div>
                  </template>
                  <div class="edit-grid4">
                    <div class="edit-kv">
                      <div class="edit-k">initialDelay</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.liveness.initialDelaySeconds"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'liveness', 'initialDelaySeconds') }"
                          :controls="false"
                          :min="0"
                          :max="3600"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">timeout</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.liveness.timeoutSeconds"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'liveness', 'timeoutSeconds') }"
                          :controls="false"
                          :min="0"
                          :max="3600"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">period</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.liveness.periodSeconds"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'liveness', 'periodSeconds') }"
                          :controls="false"
                          :min="0"
                          :max="3600"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">success</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.liveness.successThreshold"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'liveness', 'successThreshold') }"
                          :controls="false"
                          :min="0"
                          :max="100"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">failure</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.liveness.failureThreshold"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'liveness', 'failureThreshold') }"
                          :controls="false"
                          :min="0"
                          :max="100"
                        />
                      </div>
                    </div>
                  </div>
                </el-card>

                <el-card shadow="never" class="edit-section-card edit-section-card--readiness">
                  <template #header>
                    <div class="edit-section-title-row">
                      <div class="edit-section-title">Readiness</div>
                      <el-tag size="small" :type="isProbeChanged(c, 'readiness') ? 'warning' : 'info'" effect="light">
                        {{ isProbeChanged(c, 'readiness') ? '已修改' : '未修改' }}
                      </el-tag>
                    </div>
                  </template>
                  <div class="edit-grid4">
                    <div class="edit-kv">
                      <div class="edit-k">initialDelay</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.readiness.initialDelaySeconds"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'readiness', 'initialDelaySeconds') }"
                          :controls="false"
                          :min="0"
                          :max="3600"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">timeout</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.readiness.timeoutSeconds"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'readiness', 'timeoutSeconds') }"
                          :controls="false"
                          :min="0"
                          :max="3600"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">period</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.readiness.periodSeconds"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'readiness', 'periodSeconds') }"
                          :controls="false"
                          :min="0"
                          :max="3600"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">success</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.readiness.successThreshold"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'readiness', 'successThreshold') }"
                          :controls="false"
                          :min="0"
                          :max="100"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">failure</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.readiness.failureThreshold"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'readiness', 'failureThreshold') }"
                          :controls="false"
                          :min="0"
                          :max="100"
                        />
                      </div>
                    </div>
                  </div>
                </el-card>

                <el-card shadow="never" class="edit-section-card edit-section-card--startup">
                  <template #header>
                    <div class="edit-section-title-row">
                      <div class="edit-section-title">Startup</div>
                      <el-tag size="small" :type="isProbeChanged(c, 'startup') ? 'warning' : 'info'" effect="light">
                        {{ isProbeChanged(c, 'startup') ? '已修改' : '未修改' }}
                      </el-tag>
                    </div>
                  </template>
                  <div class="edit-grid4">
                    <div class="edit-kv">
                      <div class="edit-k">initialDelay</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.startup.initialDelaySeconds"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'startup', 'initialDelaySeconds') }"
                          :controls="false"
                          :min="0"
                          :max="3600"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">timeout</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.startup.timeoutSeconds"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'startup', 'timeoutSeconds') }"
                          :controls="false"
                          :min="0"
                          :max="3600"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">period</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.startup.periodSeconds"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'startup', 'periodSeconds') }"
                          :controls="false"
                          :min="0"
                          :max="3600"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">success</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.startup.successThreshold"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'startup', 'successThreshold') }"
                          :controls="false"
                          :min="0"
                          :max="100"
                        />
                      </div>
                    </div>
                    <div class="edit-kv">
                      <div class="edit-k">failure</div>
                      <div class="edit-v">
                        <el-input-number
                          v-model="c.startup.failureThreshold"
                          size="small"
                          class="edit-field"
                          :class="{ 'is-changed': isProbeValueChanged(c, 'startup', 'failureThreshold') }"
                          :controls="false"
                          :min="0"
                          :max="100"
                        />
                      </div>
                    </div>
                  </div>
                </el-card>
              </div>
            </el-card>

            <!-- 环境变量 -->
            <el-card shadow="never" class="edit-section-card edit-section-card--env">
              <template #header>
                <div class="edit-section-title-row">
                  <div class="edit-section-title">环境变量 (env)</div>
                  <el-tag size="small" :type="isEnvChanged(c) ? 'warning' : 'info'" effect="light">
                    {{ isEnvChanged(c) ? '已修改' : '未修改' }}
                  </el-tag>
                </div>
              </template>
              <el-input
                v-model="c.envText"
                type="textarea"
                :autosize="{ minRows: 3, maxRows: 12 }"
                size="small"
                class="edit-field mono"
                :class="{ 'is-changed': isEnvChanged(c) }"
                placeholder='[{"name":"KEY","value":"val"}, {"name":"K2","valueFrom":{"configMapKeyRef":{...}}}]'
              />
              <div class="edit-hint">JSON 数组，完整替换容器 env 字段。留空 = 不修改。</div>
            </el-card>

            <!-- envFrom -->
            <el-card shadow="never" class="edit-section-card edit-section-card--envfrom">
              <template #header>
                <div class="edit-section-title-row">
                  <div class="edit-section-title">envFrom</div>
                  <el-tag size="small" :type="isEnvFromChanged(c) ? 'warning' : 'info'" effect="light">
                    {{ isEnvFromChanged(c) ? '已修改' : '未修改' }}
                  </el-tag>
                </div>
              </template>
              <el-input
                v-model="c.envFromText"
                type="textarea"
                :autosize="{ minRows: 2, maxRows: 8 }"
                size="small"
                class="edit-field mono"
                :class="{ 'is-changed': isEnvFromChanged(c) }"
                placeholder='[{"configMapRef":{"name":"my-config"}}, {"secretRef":{"name":"my-secret"}}]'
              />
              <div class="edit-hint">JSON 数组，完整替换容器 envFrom 字段。留空 = 不修改。</div>
            </el-card>

            <!-- VolumeMounts -->
            <el-card shadow="never" class="edit-section-card edit-section-card--volmount">
              <template #header>
                <div class="edit-section-title-row">
                  <div class="edit-section-title">VolumeMounts</div>
                  <el-tag size="small" :type="isVolumeMountsChanged(c) ? 'warning' : 'info'" effect="light">
                    {{ isVolumeMountsChanged(c) ? '已修改' : '未修改' }}
                  </el-tag>
                </div>
              </template>
              <el-input
                v-model="c.volumeMountsText"
                type="textarea"
                :autosize="{ minRows: 2, maxRows: 10 }"
                size="small"
                class="edit-field mono"
                :class="{ 'is-changed': isVolumeMountsChanged(c) }"
                placeholder='[{"name":"data","mountPath":"/data"}, {"name":"config","mountPath":"/etc/config","readOnly":true}]'
              />
              <div class="edit-hint">JSON 数组，完整替换容器 volumeMounts 字段。留空 = 不修改。</div>
            </el-card>
          </div>
        </div>
      </div>
    </el-form>
    <div class="edit-drawer-footer">
      <el-button :disabled="editSaving" @click="closeEditDeployment">取消</el-button>
      <el-button type="primary" :loading="editSaving" @click="saveEditDeployment">保存</el-button>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Close, Plus } from '@element-plus/icons-vue'
import { useDeploymentEdit } from '@/features/k8s/composables/useDeploymentEdit'

const props = defineProps<{
  clusterId?: number
  clusterName?: string
  workloadKind?: 'Deployment' | 'StatefulSet' | 'DaemonSet'
}>()

const emit = defineEmits<{
  (e: 'saved'): void
}>()

const clusterId = computed(() => props.clusterId)
const clusterName = computed(() => String(props.clusterName ?? ''))
const workloadKind = computed(() => {
  if (props.workloadKind === 'StatefulSet') return 'StatefulSet'
  if (props.workloadKind === 'DaemonSet') return 'DaemonSet'
  return 'Deployment'
})

type LabelRow = { key: string; value: string }
type TolerationOperator = 'Equal' | 'Exists'
type TolerationEffect = '' | 'NoSchedule' | 'PreferNoSchedule' | 'NoExecute'
type TolerationRow = {
  key: string
  operator: TolerationOperator
  value: string
  effect: TolerationEffect
  tolerationSeconds: number | null
}

const labelsUiMode = ref<'form' | 'json'>('form')
const tolerationsUiMode = ref<'form' | 'json'>('form')
const labelRows = ref<LabelRow[]>([])
const tolerationRows = ref<TolerationRow[]>([])

function labelsTextToRows(text: string): LabelRow[] | null {
  const t = String(text ?? '').trim()
  if (!t) return []
  try {
    const v = JSON.parse(t)
    if (!v || typeof v !== 'object' || Array.isArray(v)) return null
    return Object.entries(v as Record<string, any>)
      .map(([k, vv]) => {
        const kk = String(k ?? '').trim()
        if (!kk) return null
        const ss = vv != null ? String(vv).trim() : ''
        return { key: kk, value: ss }
      })
      .filter(Boolean) as LabelRow[]
  } catch {
    return null
  }
}

function labelRowsToText(rows: LabelRow[]): string {
  const out: Record<string, string> = {}
  for (const it of rows) {
    const kk = String(it?.key ?? '').trim()
    if (!kk) continue
    out[kk] = String(it?.value ?? '').trim()
  }
  return JSON.stringify(out, null, 2)
}

function tolerationsTextToRows(text: string): TolerationRow[] | null {
  const t = String(text ?? '').trim()
  if (!t) return []
  try {
    const v = JSON.parse(t)
    if (!Array.isArray(v)) return null
    return v
      .map((it) => {
        if (!it || typeof it !== 'object' || Array.isArray(it)) return null
        const m = it as Record<string, any>
        const opRaw = String(m.operator ?? 'Equal').trim()
        const op: TolerationOperator = opRaw === 'Exists' ? 'Exists' : 'Equal'
        const secs = m.tolerationSeconds != null && m.tolerationSeconds !== '' ? Number(m.tolerationSeconds) : null
        return {
          key: String(m.key ?? ''),
          operator: op,
          value: String(m.value ?? ''),
          effect: (String(m.effect ?? '').trim() as TolerationEffect) || '',
          tolerationSeconds: secs != null && Number.isFinite(secs) ? secs : null
        }
      })
      .filter(Boolean) as TolerationRow[]
  } catch {
    return null
  }
}

function tolerationRowsToText(rows: TolerationRow[]): string {
  const out = rows
    .map((it) => {
      const key = String(it?.key ?? '').trim()
      const value = String(it?.value ?? '').trim()
      const effect = String(it?.effect ?? '').trim()
      const secs = it?.tolerationSeconds
      const hasAny = !!key || !!value || !!effect || secs != null
      if (!hasAny) return null
      const o: Record<string, any> = {}
      if (key !== '') o.key = key
      o.operator = it?.operator === 'Exists' ? 'Exists' : 'Equal'
      if (o.operator === 'Equal') o.value = value
      if (effect !== '') o.effect = effect
      if (secs != null && Number.isFinite(secs)) o.tolerationSeconds = secs
      return o
    })
    .filter(Boolean)
  return JSON.stringify(out, null, 2)
}

function addLabelRow() {
  labelRows.value.push({ key: '', value: '' })
}

function removeLabelRow(idx: number) {
  labelRows.value.splice(idx, 1)
}

function addTolerationRow() {
  tolerationRows.value.push({ key: '', operator: 'Equal', value: '', effect: '', tolerationSeconds: null })
}

function removeTolerationRow(idx: number) {
  tolerationRows.value.splice(idx, 1)
}

function onTolerationOperatorChange(t: TolerationRow) {
  if (t.operator === 'Exists') t.value = ''
}

const {
  editVisible,
  editSaving,
  editForm,
  editActiveContainer,
  editTarget,
  containerKey,
  openEditDeployment,
  closeEditDeployment,
  saveEditDeployment,
  isEditChanged,
  isMetaChanged,
  isLabelsChanged,
  isTolerationsChanged,
  isContainerChanged,
  isContainerResourcesChanged,
  isContainerProbesChanged,
  isContainerEnvChanged,
  isReplicasChanged,
  isImageChanged,
  isImagePullPolicyChanged,
  isImageInvalid,
  isRequestsCpuChanged,
  isRequestsMemoryChanged,
  isLimitsCpuChanged,
  isLimitsMemoryChanged,
  isProbeChanged,
  isProbeValueChanged,
  isEnvChanged,
  isEnvFromChanged,
  isVolumeMountsChanged,
  isStrategyChanged,
  isVolumesChanged
} = useDeploymentEdit({
  clusterId,
  clusterName,
  workloadKind: workloadKind.value,
  onSaved: async () => emit('saved')
})

watch(
  () => editForm.value,
  (f) => {
    if (!f) return
    const lr = labelsTextToRows(f.labelsText)
    if (lr == null) {
      labelsUiMode.value = 'json'
      labelRows.value = []
    } else {
      labelsUiMode.value = 'form'
      labelRows.value = lr
    }

    const tr = tolerationsTextToRows(f.tolerationsText)
    if (tr == null) {
      tolerationsUiMode.value = 'json'
      tolerationRows.value = []
    } else {
      tolerationsUiMode.value = 'form'
      tolerationRows.value = tr
    }
  },
  { immediate: true }
)

watch(
  labelRows,
  (rows) => {
    if (labelsUiMode.value !== 'form') return
    const f = editForm.value
    if (!f) return
    f.labelsText = labelRowsToText(rows)
  },
  { deep: true }
)

watch(
  tolerationRows,
  (rows) => {
    if (tolerationsUiMode.value !== 'form') return
    const f = editForm.value
    if (!f) return
    f.tolerationsText = tolerationRowsToText(rows)
  },
  { deep: true }
)

const allContainers = computed(() => {
  const f = editForm.value
  if (!f) return []
  return [...(f.containers ?? []), ...(f.initContainers ?? [])]
})

defineExpose<{ open: (row: any) => void }>({ open: openEditDeployment })
</script>

<style scoped>
.deployment-edit-form {
  margin: 0;
}

.edit-container-pill {
  border-radius: 999px;
  font-weight: 700;
}

.edit-container-pill.el-button--primary {
  --el-button-text-color: #fff;
  --el-button-bg-color: #1d4ed8;
  --el-button-border-color: #1d4ed8;
  --el-button-hover-text-color: #fff;
  --el-button-hover-bg-color: #1e40af;
  --el-button-hover-border-color: #1e40af;
  --el-button-active-text-color: #fff;
  --el-button-active-bg-color: #1e40af;
  --el-button-active-border-color: #1e40af;
  color: #fff !important;
  border-color: #1d4ed8 !important;
  background: #1d4ed8 !important;
  background-image: none !important;
  box-shadow: 0 1px 0 rgba(2, 6, 23, 0.12);
}

.edit-container-pill.el-button--default.is-plain {
  --el-button-text-color: #fff;
  --el-button-bg-color: rgba(15, 23, 42, 0.72);
  --el-button-border-color: rgba(15, 23, 42, 0.35);
  --el-button-hover-text-color: #fff;
  --el-button-hover-bg-color: rgba(15, 23, 42, 0.86);
  --el-button-hover-border-color: rgba(15, 23, 42, 0.45);
  --el-button-active-text-color: #fff;
  --el-button-active-bg-color: rgba(15, 23, 42, 0.86);
  --el-button-active-border-color: rgba(15, 23, 42, 0.45);
  color: #fff !important;
  border-color: rgba(15, 23, 42, 0.35) !important;
  background: rgba(15, 23, 42, 0.72) !important;
  background-image: none !important;
}

:global(html.dark) .edit-container-pill.el-button--default.is-plain {
  --el-button-text-color: #fff;
  --el-button-bg-color: rgba(148, 163, 184, 0.18);
  --el-button-border-color: rgba(226, 232, 240, 0.22);
  --el-button-hover-text-color: #fff;
  --el-button-hover-bg-color: rgba(148, 163, 184, 0.26);
  --el-button-hover-border-color: rgba(226, 232, 240, 0.32);
  --el-button-active-text-color: #fff;
  --el-button-active-bg-color: rgba(148, 163, 184, 0.26);
  --el-button-active-border-color: rgba(226, 232, 240, 0.32);
  color: #fff !important;
  border-color: rgba(226, 232, 240, 0.22) !important;
  background: rgba(148, 163, 184, 0.18) !important;
  background-image: none !important;
}

.edit-container-pill.el-button--default.is-plain :deep(.el-button__text),
.edit-container-pill.el-button--default.is-plain :deep(.el-icon) {
  color: #fff !important;
}

:global(html.dark) .edit-container-pill.el-button--default.is-plain :deep(.el-button__text),
:global(html.dark) .edit-container-pill.el-button--default.is-plain :deep(.el-icon) {
  color: #fff !important;
}

.edit-container-pill.el-button--primary :deep(.el-button__text) {
  color: #fff;
  text-shadow: 0 1px 0 rgba(2, 6, 23, 0.28);
}

.edit-container-pill.el-button--primary :deep(.el-icon) {
  color: #fff;
}

.edit-container-pill :deep(.el-icon) {
  margin-right: 4px;
}

.edit-kv--replicas::before {
  background: rgba(37, 99, 235, 0.8);
  opacity: 0.55;
}

.edit-kv--cluster::before {
  background: rgba(37, 99, 235, 0.8);
  opacity: 0.55;
}

.edit-kv--cluster .edit-k {
  color: rgba(37, 99, 235, 0.85);
}

:global(html.dark) .edit-kv--cluster .edit-k {
  color: rgba(96, 165, 250, 0.9);
}

.edit-kv--name::before {
  background: rgba(168, 85, 247, 0.85);
  opacity: 0.55;
}

.edit-kv--name .edit-k {
  color: rgba(168, 85, 247, 0.88);
}

:global(html.dark) .edit-kv--name .edit-k {
  color: rgba(216, 180, 254, 0.92);
}

.edit-kv--namespace::before {
  background: rgba(16, 185, 129, 0.85);
  opacity: 0.55;
}

.edit-kv--namespace .edit-k {
  color: rgba(16, 185, 129, 0.9);
}

:global(html.dark) .edit-kv--namespace .edit-k {
  color: rgba(52, 211, 153, 0.92);
}

.edit-kv--replicas .edit-k {
  color: rgba(37, 99, 235, 0.85);
}

:global(html.dark) .edit-kv--replicas .edit-k {
  color: rgba(96, 165, 250, 0.9);
}

.edit-kv--replicas :deep(.el-input-number) {
  --el-input-height: 32px;
}

.edit-kv--container-name::before {
  background: rgba(37, 99, 235, 0.8);
  opacity: 0.55;
}

.edit-kv--container-name .edit-k {
  color: rgba(37, 99, 235, 0.85);
}

:global(html.dark) .edit-kv--container-name .edit-k {
  color: rgba(96, 165, 250, 0.9);
}

.edit-kv--image::before {
  background: rgba(168, 85, 247, 0.85);
  opacity: 0.55;
}

.edit-kv--image .edit-k {
  color: rgba(168, 85, 247, 0.88);
}

:global(html.dark) .edit-kv--image .edit-k {
  color: rgba(216, 180, 254, 0.92);
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

.edit-ro {
  font-weight: 900;
  color: rgba(2, 6, 23, 0.82);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.edit-section-card--base .edit-ro {
  height: 32px;
  line-height: 32px;
}

.edit-section-card--container .edit-ro {
  display: flex;
  align-items: center;
  height: 28px;
}

.edit-section-card--container .edit-grid4 {
  grid-template-columns: 220px repeat(3, minmax(0, 1fr));
}

@media (max-width: 1280px) {
  .edit-section-card--container .edit-grid4 {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .edit-section-card--container .edit-grid4 {
    grid-template-columns: 1fr;
  }
}

.edit-section-card--container :deep(.el-input) {
  --el-input-height: 28px;
}

.edit-section-card--container :deep(.el-input__wrapper) {
  min-height: 28px;
  padding: 2px 8px;
}

.edit-section-card--container :deep(.el-input__inner) {
  height: 28px;
  line-height: 28px;
}

.edit-ro--name {
  font-size: 12px;
  font-weight: 800;
  opacity: 0.85;
}
:global(html.dark) .edit-ro {
  color: rgba(226, 232, 240, 0.9);
}

.edit-k {
  font-size: 12px;
  font-weight: 800;
  color: rgba(2, 6, 23, 0.62);
  white-space: nowrap;
}

:global(html.dark) .edit-k {
  color: rgba(226, 232, 240, 0.62);
}

.edit-v {
  min-width: 0;
}

.edit-kv :deep(.el-input__wrapper) {
  width: 100%;
}
.edit-kv :deep(.el-input-number) {
  width: 100%;
}

.edit-label-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 8px;
  align-items: start;
}

.edit-label-item {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 6px 8px;
  border-radius: 12px;
  border: 1px dashed rgba(2, 6, 23, 0.12);
  background: rgba(255, 255, 255, 0.5);
  min-width: 0;
}

:global(html.dark) .edit-label-item {
  border-color: rgba(226, 232, 240, 0.14);
  background: rgba(2, 6, 23, 0.18);
}

.edit-label-tone-0 {
  border-color: rgba(37, 99, 235, 0.22);
  background: rgba(37, 99, 235, 0.08);
}

.edit-label-tone-1 {
  border-color: rgba(16, 185, 129, 0.22);
  background: rgba(16, 185, 129, 0.08);
}

.edit-label-tone-2 {
  border-color: rgba(168, 85, 247, 0.22);
  background: rgba(168, 85, 247, 0.08);
}

.edit-label-tone-3 {
  border-color: rgba(245, 158, 11, 0.25);
  background: rgba(245, 158, 11, 0.08);
}

.edit-label-tone-4 {
  border-color: rgba(236, 72, 153, 0.22);
  background: rgba(236, 72, 153, 0.08);
}

.edit-label-tone-5 {
  border-color: rgba(14, 165, 233, 0.22);
  background: rgba(14, 165, 233, 0.08);
}

:global(html.dark) .edit-label-tone-0 {
  border-color: rgba(96, 165, 250, 0.22);
  background: rgba(96, 165, 250, 0.12);
}

:global(html.dark) .edit-label-tone-1 {
  border-color: rgba(52, 211, 153, 0.22);
  background: rgba(52, 211, 153, 0.12);
}

:global(html.dark) .edit-label-tone-2 {
  border-color: rgba(216, 180, 254, 0.22);
  background: rgba(216, 180, 254, 0.12);
}

:global(html.dark) .edit-label-tone-3 {
  border-color: rgba(252, 211, 77, 0.22);
  background: rgba(252, 211, 77, 0.12);
}

:global(html.dark) .edit-label-tone-4 {
  border-color: rgba(251, 113, 133, 0.22);
  background: rgba(251, 113, 133, 0.12);
}

:global(html.dark) .edit-label-tone-5 {
  border-color: rgba(56, 189, 248, 0.22);
  background: rgba(56, 189, 248, 0.12);
}

.edit-label-key {
  width: 180px;
}

.edit-label-value {
  flex: 1;
  min-width: 0;
}

.edit-tol-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.edit-tol-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-radius: 12px;
  border: 1px dashed rgba(2, 6, 23, 0.12);
  background: rgba(255, 255, 255, 0.5);
}

:global(html.dark) .edit-tol-row {
  border-color: rgba(226, 232, 240, 0.14);
  background: rgba(2, 6, 23, 0.18);
}

.edit-tol-tone-0 {
  border-color: rgba(37, 99, 235, 0.22);
  background: rgba(37, 99, 235, 0.08);
}

.edit-tol-tone-1 {
  border-color: rgba(16, 185, 129, 0.22);
  background: rgba(16, 185, 129, 0.08);
}

.edit-tol-tone-2 {
  border-color: rgba(168, 85, 247, 0.22);
  background: rgba(168, 85, 247, 0.08);
}

.edit-tol-tone-3 {
  border-color: rgba(245, 158, 11, 0.25);
  background: rgba(245, 158, 11, 0.08);
}

.edit-tol-tone-4 {
  border-color: rgba(236, 72, 153, 0.22);
  background: rgba(236, 72, 153, 0.08);
}

.edit-tol-tone-5 {
  border-color: rgba(14, 165, 233, 0.22);
  background: rgba(14, 165, 233, 0.08);
}

:global(html.dark) .edit-tol-tone-0 {
  border-color: rgba(96, 165, 250, 0.22);
  background: rgba(96, 165, 250, 0.12);
}

:global(html.dark) .edit-tol-tone-1 {
  border-color: rgba(52, 211, 153, 0.22);
  background: rgba(52, 211, 153, 0.12);
}

:global(html.dark) .edit-tol-tone-2 {
  border-color: rgba(216, 180, 254, 0.22);
  background: rgba(216, 180, 254, 0.12);
}

:global(html.dark) .edit-tol-tone-3 {
  border-color: rgba(252, 211, 77, 0.22);
  background: rgba(252, 211, 77, 0.12);
}

:global(html.dark) .edit-tol-tone-4 {
  border-color: rgba(251, 113, 133, 0.22);
  background: rgba(251, 113, 133, 0.12);
}

:global(html.dark) .edit-tol-tone-5 {
  border-color: rgba(56, 189, 248, 0.22);
  background: rgba(56, 189, 248, 0.12);
}

.edit-tol-key {
  width: 220px;
}

.edit-tol-operator {
  width: 130px;
}

.edit-tol-value {
  width: 200px;
}

.edit-tol-effect {
  width: 180px;
}

.meta-del-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  --el-button-text-color: rgba(148, 163, 184, 0.95);
  --el-button-border-color: rgba(148, 163, 184, 0.32);
  --el-button-bg-color: rgba(148, 163, 184, 0.1);
  --el-button-hover-text-color: rgba(239, 68, 68, 0.95);
  --el-button-hover-border-color: rgba(239, 68, 68, 0.35);
  --el-button-hover-bg-color: rgba(239, 68, 68, 0.1);
}

:global(html.dark) .meta-del-btn {
  --el-button-bg-color: rgba(148, 163, 184, 0.08);
  --el-button-border-color: rgba(148, 163, 184, 0.22);
}

@media (max-width: 640px) {
  .edit-label-grid {
    grid-template-columns: 1fr;
  }

  .edit-label-key {
    width: 160px;
  }

  .edit-tol-key {
    width: 100%;
  }
}

.edit-probe-stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.edit-section-card--probes :deep(.el-input-number) {
  --el-input-height: 26px;
}

.edit-section-card--probes :deep(.el-input-number .el-input__wrapper) {
  min-height: 26px;
  padding: 1px 6px;
}

.edit-section-card--probes :deep(.el-input-number .el-input__inner) {
  height: 26px;
  line-height: 26px;
}

.edit-section-card--probes .edit-grid4 {
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 6px 8px;
}

.edit-section-card--probes .edit-kv {
  grid-template-columns: auto minmax(0, 1fr);
  column-gap: 8px;
  padding: 6px 8px 6px 12px;
}

.edit-section-card--probes .edit-k {
  font-size: 11px;
}

@media (max-width: 1280px) {
  .edit-section-card--probes .edit-grid4 {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .edit-section-card--probes .edit-grid4 {
    grid-template-columns: 1fr;
  }
}

.edit-res-table {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr) minmax(0, 1fr);
  gap: 8px 10px;
  align-items: center;
}

.edit-res-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 900;
  color: rgba(2, 6, 23, 0.78);
}

.edit-strategy-radio {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 8px;
}

:deep(.edit-strategy-radio .el-radio-button__inner) {
  min-width: 132px;
}

.edit-res-head--cpu::before,
.edit-res-head--mem::before {
  content: '';
  width: 8px;
  height: 8px;
  border-radius: 999px;
  box-shadow: 0 0 0 1px rgba(2, 6, 23, 0.08) inset;
}

.edit-res-head--cpu::before {
  background: rgba(245, 158, 11, 0.9);
}

.edit-res-head--mem::before {
  background: rgba(168, 85, 247, 0.9);
}

:global(html.dark) .edit-res-head {
  color: rgba(226, 232, 240, 0.82);
}

.edit-res-head--cpu::before,
.edit-res-head--mem::before {
  box-shadow: 0 0 0 1px rgba(226, 232, 240, 0.08) inset;
}

.edit-res-rowlabel {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 800;
  color: rgba(2, 6, 23, 0.62);
  white-space: nowrap;
}

.edit-res-rowlabel::before {
  content: '';
  width: 8px;
  height: 8px;
  border-radius: 3px;
  background: rgba(148, 163, 184, 0.65);
}

.edit-res-rowlabel--requests::before {
  background: rgba(16, 185, 129, 0.9);
}

.edit-res-rowlabel--limits::before {
  background: rgba(239, 68, 68, 0.9);
}

:global(html.dark) .edit-res-rowlabel {
  color: rgba(226, 232, 240, 0.62);
}

.edit-section-card--resources :deep(.el-input) {
  --el-input-height: 28px;
}

.edit-section-card--resources :deep(.el-input__wrapper) {
  min-height: 28px;
  padding: 2px 8px;
}

.edit-section-card--resources :deep(.el-input__inner) {
  height: 28px;
  line-height: 28px;
}

.edit-res-cell :deep(.el-input__wrapper) {
  width: 100%;
}

.edit-probe-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 18px;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  opacity: 0.9;
}

.edit-hint {
  margin-top: 4px;
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  line-height: 1.5;
}

.deployment-edit-drawer :deep(.el-button.el-button--primary.is-plain) {
  --el-button-text-color: #fff;
  --el-button-hover-text-color: #fff;
  --el-button-active-text-color: #fff;
  color: #fff !important;
}

:global(html.dark) .deployment-edit-drawer :deep(.el-button.el-button--primary.is-plain) {
  --el-button-text-color: #fff;
  --el-button-hover-text-color: #fff;
  --el-button-active-text-color: #fff;
  color: #fff !important;
}

.deployment-edit-drawer :deep(.el-button.el-button--primary.is-plain .el-button__text),
.deployment-edit-drawer :deep(.el-button.el-button--primary.is-plain .el-icon) {
  color: #fff !important;
}
</style>
