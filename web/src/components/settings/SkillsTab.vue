<template>
  <div class="tab-root">
    <div class="tab-header">
      <div>
        <h2 class="tab-title">技能管理</h2>
        <p class="tab-desc">查看和管理 Agent 可用的技能模块，技能文件存放于工作区 skills/ 目录</p>
      </div>
      <button class="btn-secondary" :disabled="reloading" @click="handleReload">
        <RefreshCwIcon :size="13" class="reload-icon" :class="{ spinning: reloading }" />
        {{ reloading ? '重载中...' : '重新加载' }}
      </button>
    </div>

    <div v-if="loading" class="loading-state">
      <div class="loading-spinner" />
      <span>加载中...</span>
    </div>

    <div v-else-if="skills.length === 0" class="empty-state">
      <BrainIcon :size="40" class="empty-icon" />
      <p class="empty-text">暂无可用的技能</p>
      <p class="empty-tip">将技能文件夹放入工作区 skills/ 目录后，点击「重新加载」即可生效</p>
    </div>

    <div v-else class="skill-grid">
      <div v-for="skill in skills" :key="skill.name" class="skill-card">
        <div class="skill-header">
          <label class="toggle">
            <input
              type="checkbox"
              :checked="skill.enabled"
              :disabled="!!loadingSkills[skill.name]"
              @change="handleToggle(skill.name, !skill.enabled)"
            />
            <span class="toggle-slider" />
          </label>
          <div class="skill-info">
            <div class="skill-title">
              <span class="skill-name">{{ skill.display_name }}</span>
              <span class="skill-tag">{{ skill.name }}</span>
            </div>
            <p class="skill-desc">{{ skill.description }}</p>
          </div>
        </div>
        <div class="skill-meta">
          <span class="meta-item"><span class="meta-label">版本</span> v{{ skill.version }}</span>
          <span class="meta-item"><span class="meta-label">等级</span> {{ skill.level }}</span>
          <span v-if="skill.author" class="meta-item"><span class="meta-label">作者</span> {{ skill.author }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { RefreshCwIcon, BrainIcon } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { getSkills, setSkillEnabled, reloadSkills, type Skill } from '@/api/settings'

const loading = ref(false)
const reloading = ref(false)
const skills = ref<Skill[]>([])
const loadingSkills = reactive<Record<string, boolean>>({})

async function loadSkills() {
  loading.value = true
  try {
    skills.value = await getSkills()
  } catch {
    toast.error('加载失败')
  } finally {
    loading.value = false
  }
}

async function handleReload() {
  reloading.value = true
  try {
    await reloadSkills()
    await loadSkills()
    toast.success('技能已重新加载')
  } catch {
    toast.error('重载失败')
  } finally {
    reloading.value = false
  }
}

async function handleToggle(name: string, enabled: boolean) {
  loadingSkills[name] = true
  try {
    await setSkillEnabled(name, enabled)
    const skill = skills.value.find(s => s.name === name)
    if (skill) skill.enabled = enabled
    toast.success(enabled ? '已启用' : '已禁用')
  } catch {
    toast.error('操作失败')
  } finally {
    loadingSkills[name] = false
  }
}

onMounted(loadSkills)
</script>

<style scoped>
.tab-root { display: flex; flex-direction: column; gap: 20px; }

.tab-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.tab-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin: 0 0 4px; }
.tab-desc { font-size: 12px; color: var(--text-secondary); margin: 0; }

.loading-state {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-secondary);
  font-size: 13px;
  padding: 20px 0;
}

.loading-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 48px 0;
}

.empty-icon { color: var(--text-tertiary); }
.empty-text { font-size: 14px; color: var(--text-secondary); margin: 0; }
.empty-tip { font-size: 12px; color: var(--text-tertiary); margin: 0; text-align: center; }

.skill-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 10px;
}

.skill-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 14px 16px;
  transition: border-color 0.15s;
}

.skill-card:hover { border-color: rgba(124, 106, 247, 0.3); }

.skill-header {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.skill-info { flex: 1; }

.skill-title {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-bottom: 4px;
}

.skill-name { font-size: 13px; font-weight: 600; color: var(--text-primary); }

.skill-tag {
  font-size: 10px;
  color: var(--text-tertiary);
  background: var(--bg-overlay);
  border-radius: 3px;
  padding: 1px 5px;
  font-family: monospace;
}

.skill-desc { font-size: 12px; color: var(--text-secondary); margin: 0; line-height: 1.5; }

.skill-meta {
  display: flex;
  gap: 12px;
  padding-top: 10px;
  border-top: 1px solid var(--border-subtle);
}

.meta-item { font-size: 11px; color: var(--text-tertiary); }
.meta-label { color: var(--text-disabled); margin-right: 4px; }

/* Toggle */
.toggle { position: relative; width: 34px; height: 18px; cursor: pointer; display: block; flex-shrink: 0; }
.toggle input { display: none; }
.toggle-slider {
  position: absolute;
  inset: 0;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 9px;
  transition: background 0.2s;
}
.toggle-slider::before {
  content: '';
  position: absolute;
  width: 12px;
  height: 12px;
  background: var(--text-tertiary);
  border-radius: 50%;
  top: 2px;
  left: 2px;
  transition: transform 0.2s, background 0.2s;
}
.toggle input:checked + .toggle-slider { background: rgba(124, 106, 247, 0.2); border-color: var(--accent); }
.toggle input:checked + .toggle-slider::before { transform: translateX(16px); background: var(--accent); }

.reload-icon { transition: transform 0.3s; }
.reload-icon.spinning { animation: spin 0.8s linear infinite; }

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
  white-space: nowrap;
  flex-shrink: 0;
}

.btn-secondary:hover { background: var(--bg-elevated); color: var(--text-primary); }
.btn-secondary:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
