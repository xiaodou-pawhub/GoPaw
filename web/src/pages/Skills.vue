<template>
  <div class="skills-root">
    <!-- 顶部栏 -->
    <div class="skills-topbar">
      <div class="topbar-left">
        <h1 class="page-title">技能</h1>
        <div class="view-tabs">
          <button class="view-tab" :class="{ active: view === 'installed' }" @click="view = 'installed'">已安装</button>
          <button class="view-tab" :class="{ active: view === 'market' }" @click="switchToMarket">技能市场</button>
        </div>
      </div>
      <div class="topbar-right">
        <template v-if="view === 'installed'">
          <!-- 隐藏文件选择 -->
          <input ref="fileInputRef" type="file" accept=".zip" class="hidden-input" @change="handleFileSelect" />
          <button class="btn-secondary" :disabled="importing" @click="triggerImport">
            <UploadIcon :size="13" />
            {{ importing ? '导入中...' : '导入 ZIP' }}
          </button>
          <button class="btn-secondary" :disabled="reloading" @click="handleReload">
            <RefreshCwIcon :size="13" class="reload-icon" :class="{ spinning: reloading }" />
            {{ reloading ? '重载中...' : '重新加载' }}
          </button>
        </template>
        <template v-else>
          <div class="market-search">
            <SearchIcon :size="13" class="search-icon" />
            <input v-model="marketQuery" placeholder="搜索技能市场..." class="search-input" @input="debouncedMarketSearch" />
          </div>
        </template>
      </div>
    </div>

    <!-- 已安装视图 -->
    <div v-if="view === 'installed'" class="view-content">
      <div v-if="loading" class="center-state">
        <div class="loading-spinner" />
        <span>加载中...</span>
      </div>

      <div v-else-if="skills.length === 0" class="center-state">
        <BrainIcon :size="48" class="empty-icon" />
        <p class="empty-text">暂无已安装的技能</p>
        <p class="empty-hint">从技能市场安装，或导入本地 ZIP 包</p>
        <div class="empty-actions">
          <button class="btn-primary" @click="switchToMarket">
            <StoreIcon :size="14" />浏览技能市场
          </button>
          <button class="btn-secondary" @click="triggerImport">
            <UploadIcon :size="14" />导入 ZIP
          </button>
        </div>
      </div>

      <div v-else class="skill-grid">
        <div
          v-for="skill in skills"
          :key="skill.name"
          class="skill-card"
          :class="{ enabled: skill.enabled }"
        >
          <div class="skill-card-header">
            <div class="skill-avatar">{{ skill.display_name.charAt(0).toUpperCase() }}</div>
            <div class="skill-info">
              <div class="skill-name-row">
                <span class="skill-display-name">{{ skill.display_name }}</span>
                <span class="level-badge" :class="skill.level === 2 ? 'level-code' : 'level-prompt'">
                  {{ skill.level === 2 ? '代码' : '提示词' }}
                </span>
              </div>
              <span class="skill-id">{{ skill.name }}</span>
            </div>
            <label class="toggle" :title="skill.enabled ? '禁用' : '启用'">
              <input
                type="checkbox"
                :checked="skill.enabled"
                :disabled="!!loadingSkills[skill.name]"
                @change="handleToggle(skill.name, !skill.enabled)"
              />
              <span class="toggle-slider" />
            </label>
          </div>
          <p class="skill-desc">{{ skill.description }}</p>
          <div class="skill-meta">
            <span class="meta-item">v{{ skill.version }}</span>
            <template v-if="skill.author">
              <span class="meta-dot">·</span>
              <span class="meta-item">{{ skill.author }}</span>
            </template>
          </div>
        </div>
      </div>
    </div>

    <!-- 技能市场视图 -->
    <div v-else class="market-view">
      <!-- 左侧分类 -->
      <aside class="market-panel">
        <div class="panel-header"><h3 class="panel-title">分类</h3></div>
        <div class="category-list">
          <button
            v-for="cat in categories"
            :key="cat.key"
            class="category-item"
            :class="{ active: selectedCategory === cat.key }"
            @click="selectCategory(cat.key)"
          >
            <component :is="cat.icon" :size="14" />
            <span>{{ cat.label }}</span>
          </button>
        </div>
      </aside>

      <!-- 主内容 -->
      <main class="market-main">
        <div v-if="marketLoading" class="center-state">
          <div class="loading-spinner" />
          <span>加载技能市场...</span>
        </div>

        <div v-else-if="marketError" class="center-state">
          <WifiOffIcon :size="40" class="empty-icon" />
          <p class="empty-text">无法连接到技能市场</p>
          <p class="empty-hint">请检查网络连接后重试</p>
          <button class="btn-secondary" @click="loadMarketSkills">重新加载</button>
        </div>

        <div v-else-if="marketSkills.length === 0" class="center-state">
          <BrainIcon :size="48" class="empty-icon" />
          <p class="empty-text">{{ marketQuery ? '未找到匹配的技能' : '暂无可用技能' }}</p>
        </div>

        <template v-else>
          <div class="market-skill-list">
            <div
              v-for="ms in marketSkills"
              :key="ms.name"
              class="market-skill-card"
              :class="{ installed: isInstalled(ms.name) }"
            >
              <div class="skill-avatar">{{ ms.display_name.charAt(0).toUpperCase() }}</div>
              <div class="skill-info">
                <div class="skill-name-row">
                  <span class="skill-display-name">{{ ms.display_name }}</span>
                  <span v-if="isInstalled(ms.name)" class="installed-badge">已安装</span>
                  <span class="level-badge" :class="ms.level === 'L2' ? 'level-code' : 'level-prompt'">
                    {{ ms.level === 'L2' ? '代码' : '提示词' }}
                  </span>
                  <span v-if="ms.featured" class="featured-badge">精选</span>
                </div>
                <p class="skill-desc">{{ ms.description }}</p>
                <div class="skill-meta">
                  <span class="meta-item">{{ ms.author }}</span>
                  <span class="meta-dot">·</span>
                  <span class="meta-item">v{{ ms.latest_version }}</span>
                  <span class="meta-dot">·</span>
                  <span class="meta-item">{{ ms.install_count.toLocaleString() }} 安装</span>
                </div>
              </div>
              <button
                class="install-btn"
                :class="{ installed: isInstalled(ms.name) }"
                :disabled="!!installingSkills[ms.name]"
                @click="handleInstall(ms)"
              >
                <CheckIcon v-if="isInstalled(ms.name)" :size="13" />
                <DownloadIcon v-else-if="!installingSkills[ms.name]" :size="13" />
                <div v-else class="btn-spinner" />
                {{ isInstalled(ms.name) ? '已安装' : installingSkills[ms.name] ? '安装中...' : '安装' }}
              </button>
            </div>
          </div>

          <!-- 分页 -->
          <div v-if="marketTotal > marketPageSize" class="pagination">
            <button class="page-btn" :disabled="marketPage <= 1" @click="changePage(marketPage - 1)">
              <ChevronLeftIcon :size="14" />
            </button>
            <span class="page-info">{{ marketPage }} / {{ totalPages }}</span>
            <button class="page-btn" :disabled="marketPage >= totalPages" @click="changePage(marketPage + 1)">
              <ChevronRightIcon :size="14" />
            </button>
          </div>
        </template>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import {
  RefreshCwIcon, BrainIcon, SearchIcon,
  ZapIcon, BookOpenIcon, GlobeIcon,
  Store as StoreIcon, UploadIcon, DownloadIcon,
  CheckIcon, WifiOffIcon, ChevronLeftIcon, ChevronRightIcon,
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import {
  getSkills, setSkillEnabled, reloadSkills, installSkill, importSkillZip,
  getMarketSkills,
  type Skill, type MarketSkill,
} from '@/api/skills'

const view = ref<'installed' | 'market'>('installed')
const loading = ref(false)
const reloading = ref(false)
const skills = ref<Skill[]>([])
const loadingSkills = reactive<Record<string, boolean>>({})
const fileInputRef = ref<HTMLInputElement | null>(null)
const importing = ref(false)

// Market state
const marketLoading = ref(false)
const marketError = ref(false)
const marketSkills = ref<MarketSkill[]>([])
const marketQuery = ref('')
const marketPage = ref(1)
const marketPageSize = 20
const marketTotal = ref(0)
const selectedCategory = ref('all')
const installingSkills = reactive<Record<string, boolean>>({})
let marketSearchTimer: ReturnType<typeof setTimeout> | null = null

const categories = [
  { key: 'all', label: '全部', icon: GlobeIcon },
  { key: 'productivity', label: '效率工具', icon: ZapIcon },
  { key: 'knowledge', label: '知识库', icon: BookOpenIcon },
  { key: 'ai', label: 'AI 增强', icon: BrainIcon },
]

const totalPages = computed(() => Math.max(1, Math.ceil(marketTotal.value / marketPageSize)))

function isInstalled(name: string): boolean {
  return skills.value.some(s => s.name === name)
}

// ---- 已安装视图 ----

async function loadSkills() {
  loading.value = true
  try {
    skills.value = await getSkills()
  } catch {
    toast.error('加载技能列表失败')
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
    toast.success(enabled ? '技能已启用' : '技能已禁用')
  } catch {
    toast.error('操作失败')
  } finally {
    loadingSkills[name] = false
  }
}

function triggerImport() {
  fileInputRef.value?.click()
}

async function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  input.value = '' // reset for re-selection

  importing.value = true
  const tid = toast.loading(`正在导入 ${file.name}，AI 分析中（约 30-60 秒）...`)
  try {
    const result = await importSkillZip(file)
    toast.dismiss(tid)
    toast.success(`技能 "${result.name}" 导入成功`)
    await loadSkills()
  } catch (err: any) {
    toast.dismiss(tid)
    if (err?.response?.status === 409) {
      const skillName = err.response.data?.name || file.name
      const confirmed = window.confirm(`技能 "${skillName}" 已存在，是否覆写？`)
      if (confirmed) {
        importing.value = true
        const tid2 = toast.loading(`正在覆写技能 "${skillName}"...`)
        try {
          const result = await importSkillZip(file, true)
          toast.dismiss(tid2)
          toast.success(`技能 "${result.name}" 覆写成功`)
          await loadSkills()
        } catch (err2: any) {
          toast.dismiss(tid2)
          toast.error('覆写失败：' + (err2?.response?.data?.message || err2.message || '未知错误'))
        } finally {
          importing.value = false
        }
        return
      }
    } else {
      toast.error('导入失败：' + (err?.response?.data?.message || err.message || '未知错误'))
    }
  } finally {
    importing.value = false
  }
}

// ---- 技能市场视图 ----

async function loadMarketSkills() {
  marketLoading.value = true
  marketError.value = false
  try {
    const result = await getMarketSkills({
      q: marketQuery.value || undefined,
      featured: selectedCategory.value === 'featured' ? true : undefined,
      page: marketPage.value,
      page_size: marketPageSize,
    })
    marketSkills.value = result.items || []
    marketTotal.value = result.total || 0
  } catch {
    marketError.value = true
  } finally {
    marketLoading.value = false
  }
}

function switchToMarket() {
  view.value = 'market'
  if (marketSkills.value.length === 0 && !marketLoading.value) {
    loadMarketSkills()
  }
}

function selectCategory(key: string) {
  selectedCategory.value = key
  marketPage.value = 1
  loadMarketSkills()
}

function debouncedMarketSearch() {
  if (marketSearchTimer) clearTimeout(marketSearchTimer)
  marketSearchTimer = setTimeout(() => {
    marketPage.value = 1
    loadMarketSkills()
  }, 400)
}

function changePage(page: number) {
  marketPage.value = page
  loadMarketSkills()
}

async function handleInstall(ms: MarketSkill) {
  if (isInstalled(ms.name)) return
  installingSkills[ms.name] = true
  try {
    await installSkill(ms.name, ms.latest_version)
    toast.success(`"${ms.display_name}" 安装成功`)
    await loadSkills() // refresh local list so badge shows
  } catch (err: any) {
    toast.error('安装失败：' + (err?.response?.data?.message || err.message || '未知错误'))
  } finally {
    installingSkills[ms.name] = false
  }
}

onMounted(loadSkills)
</script>

<style scoped>
.skills-root {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
}

/* 顶部栏 */
.skills-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 20px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
  gap: 12px;
}

.topbar-left { display: flex; align-items: center; gap: 16px; }
.topbar-right { display: flex; align-items: center; gap: 8px; }

.page-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  white-space: nowrap;
}

.view-tabs {
  display: flex;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 3px;
  gap: 2px;
}

.view-tab {
  padding: 5px 14px;
  background: transparent;
  border: none;
  border-radius: 5px;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.view-tab:hover { color: var(--text-primary); }
.view-tab.active {
  background: var(--accent-dim);
  color: var(--accent);
  font-weight: 600;
}

.hidden-input { display: none; }

/* 市场搜索框（顶部） */
.market-search {
  display: flex;
  align-items: center;
  gap: 7px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 6px 12px;
  width: 240px;
}
.search-icon { color: var(--text-tertiary); flex-shrink: 0; }
.search-input {
  flex: 1;
  background: none;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-size: 12px;
}
.search-input::placeholder { color: var(--text-tertiary); }

/* 已安装内容区 */
.view-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
}

.center-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 60px 0;
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

.empty-icon { color: var(--text-tertiary); }
.empty-text { font-size: 14px; color: var(--text-secondary); margin: 0; }
.empty-hint { font-size: 12px; color: var(--text-tertiary); margin: 0; text-align: center; }

.empty-actions { display: flex; gap: 8px; margin-top: 4px; }

/* 已安装卡片网格 */
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
.skill-card.enabled { border-color: rgba(124, 106, 247, 0.25); }

.skill-card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.skill-avatar {
  width: 36px;
  height: 36px;
  background: var(--accent-dim);
  border: 1px solid rgba(124, 106, 247, 0.2);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 700;
  color: var(--accent);
  flex-shrink: 0;
}

.skill-info { flex: 1; min-width: 0; }

.skill-name-row {
  display: flex;
  align-items: center;
  gap: 5px;
  flex-wrap: wrap;
  margin-bottom: 2px;
}

.skill-display-name { font-size: 13px; font-weight: 600; color: var(--text-primary); }

.skill-id {
  font-size: 10px;
  color: var(--text-tertiary);
  font-family: monospace;
  background: var(--bg-overlay);
  border-radius: 3px;
  padding: 1px 5px;
}

.level-badge {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  font-weight: 500;
}
.level-prompt { background: rgba(59, 130, 246, 0.12); color: #3b82f6; }
.level-code { background: rgba(124, 106, 247, 0.12); color: var(--accent); }

.skill-desc {
  font-size: 12px;
  color: var(--text-secondary);
  margin: 0 0 10px;
  line-height: 1.5;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.skill-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  padding-top: 8px;
  border-top: 1px solid var(--border-subtle);
}

.meta-item { font-size: 11px; color: var(--text-tertiary); }
.meta-dot { color: var(--text-disabled); font-size: 11px; }

/* 市场视图 */
.market-view {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.market-panel {
  width: 150px;
  background: var(--bg-panel);
  border-right: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.panel-header {
  padding: 10px 12px 8px;
  border-bottom: 1px solid var(--border-subtle);
}

.panel-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.category-list { padding: 6px; display: flex; flex-direction: column; gap: 1px; }

.category-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 8px;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
  width: 100%;
  text-align: left;
}
.category-item:hover { background: var(--bg-overlay); color: var(--text-primary); }
.category-item.active { background: var(--accent-dim); color: var(--accent); }

.market-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.market-skill-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.market-skill-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  transition: border-color 0.15s;
}
.market-skill-card:hover { border-color: rgba(124, 106, 247, 0.3); }
.market-skill-card.installed { border-color: rgba(124, 106, 247, 0.25); }

.market-skill-card .skill-info { flex: 1; min-width: 0; }
.market-skill-card .skill-desc { margin: 3px 0 5px; white-space: nowrap; text-overflow: ellipsis; overflow: hidden; }

.installed-badge {
  font-size: 10px;
  padding: 1px 6px;
  background: rgba(34, 197, 94, 0.1);
  color: var(--green);
  border-radius: 4px;
  font-weight: 500;
}

.featured-badge {
  font-size: 10px;
  padding: 1px 6px;
  background: rgba(251, 191, 36, 0.15);
  color: #f59e0b;
  border-radius: 4px;
  font-weight: 500;
}

.install-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 6px 14px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
  transition: opacity 0.15s;
}
.install-btn:hover:not(:disabled) { opacity: 0.85; }
.install-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.install-btn.installed { background: var(--bg-overlay); color: var(--text-secondary); border: 1px solid var(--border); }

.btn-spinner {
  width: 12px;
  height: 12px;
  border: 1.5px solid rgba(255,255,255,0.4);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
  flex-shrink: 0;
}

/* 分页 */
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 12px;
  border-top: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.page-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: background 0.15s;
}
.page-btn:hover:not(:disabled) { background: var(--bg-overlay); color: var(--text-primary); }
.page-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.page-info { font-size: 12px; color: var(--text-secondary); }

/* Buttons */
.btn-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
  white-space: nowrap;
}
.btn-secondary:hover { background: var(--bg-elevated); color: var(--text-primary); }
.btn-secondary:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-primary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--accent);
  border: none;
  border-radius: 8px;
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}
.btn-primary:hover { opacity: 0.85; }

.reload-icon { transition: transform 0.3s; }
.reload-icon.spinning { animation: spin 0.8s linear infinite; }

/* Toggle */
.toggle { position: relative; width: 34px; height: 18px; cursor: pointer; display: block; flex-shrink: 0; }
.toggle input { display: none; }
.toggle-slider {
  position: absolute; inset: 0;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 9px;
  transition: background 0.2s;
}
.toggle-slider::before {
  content: '';
  position: absolute;
  width: 12px; height: 12px;
  background: var(--text-tertiary);
  border-radius: 50%;
  top: 2px; left: 2px;
  transition: transform 0.2s, background 0.2s;
}
.toggle input:checked + .toggle-slider { background: rgba(124, 106, 247, 0.2); border-color: var(--accent); }
.toggle input:checked + .toggle-slider::before { transform: translateX(16px); background: var(--accent); }
.toggle input:disabled + .toggle-slider { opacity: 0.5; cursor: not-allowed; }

/* Scrollbar */
.view-content::-webkit-scrollbar,
.market-skill-list::-webkit-scrollbar { width: 4px; }
.view-content::-webkit-scrollbar-track,
.market-skill-list::-webkit-scrollbar-track { background: transparent; }
.view-content::-webkit-scrollbar-thumb,
.market-skill-list::-webkit-scrollbar-thumb { background: var(--border); border-radius: 2px; }
</style>
