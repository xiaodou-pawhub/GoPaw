<template>
  <div class="market-root">
    <!-- 左侧分类面板 -->
    <aside class="market-panel">
      <div class="panel-header">
        <h3 class="panel-title">技能市场</h3>
      </div>
      <div class="category-list">
        <button
          v-for="cat in categories"
          :key="cat.key"
          class="category-item"
          :class="{ active: selectedCategory === cat.key }"
          @click="selectedCategory = cat.key"
        >
          <component :is="cat.icon" :size="14" />
          <span>{{ cat.label }}</span>
          <span class="cat-count">{{ getCategoryCount(cat.key) }}</span>
        </button>
      </div>
    </aside>

    <!-- 主内容区 -->
    <main class="market-main">
      <div class="market-header">
        <div class="search-box">
          <SearchIcon :size="13" class="search-icon" />
          <input v-model="searchQuery" placeholder="搜索技能..." class="search-input" />
        </div>
      </div>

      <div v-if="loading" class="market-loading">
        <div class="loading-spinner" />
        <span>加载技能列表...</span>
      </div>

      <div v-else-if="filteredSkills.length === 0" class="market-empty">
        <StorefrontIcon :size="48" class="empty-icon" />
        <p class="empty-text">{{ searchQuery ? '未找到匹配的技能' : '暂无可用技能' }}</p>
        <p class="empty-hint">技能存放在工作区的 skills/ 目录中</p>
      </div>

      <div v-else class="skill-grid">
        <div
          v-for="skill in filteredSkills"
          :key="skill.name"
          class="skill-card"
          :class="{ enabled: skill.enabled }"
        >
          <div class="skill-icon">
            {{ skill.display_name.charAt(0).toUpperCase() }}
          </div>
          <div class="skill-info">
            <div class="skill-name-row">
              <span class="skill-display-name">{{ skill.display_name }}</span>
              <span v-if="skill.enabled" class="enabled-badge">已安装</span>
            </div>
            <p class="skill-desc">{{ skill.description }}</p>
            <div class="skill-meta">
              <span class="meta-item">v{{ skill.version }}</span>
              <span class="meta-dot">·</span>
              <span class="meta-item">{{ skill.level }}</span>
              <template v-if="skill.author">
                <span class="meta-dot">·</span>
                <span class="meta-item">{{ skill.author }}</span>
              </template>
            </div>
          </div>
          <div class="skill-action">
            <label class="toggle" :title="skill.enabled ? '禁用技能' : '启用技能'">
              <input
                type="checkbox"
                :checked="skill.enabled"
                :disabled="!!loadingSkills[skill.name]"
                @change="handleToggle(skill.name, !skill.enabled)"
              />
              <span class="toggle-slider" />
            </label>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { SearchIcon, BrainIcon, ZapIcon, BookOpenIcon, GlobeIcon } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { getSkills, setSkillEnabled, type Skill } from '@/api/settings'

// 简单图标组件代替不存在的 StorefrontIcon
const StorefrontIcon = BrainIcon

const loading = ref(false)
const skills = ref<Skill[]>([])
const loadingSkills = reactive<Record<string, boolean>>({})
const searchQuery = ref('')
const selectedCategory = ref('all')

const categories = [
  { key: 'all', label: '全部', icon: GlobeIcon },
  { key: 'productivity', label: '效率工具', icon: ZapIcon },
  { key: 'knowledge', label: '知识库', icon: BookOpenIcon },
  { key: 'ai', label: 'AI 增强', icon: BrainIcon },
]

function getCategoryCount(key: string): number {
  if (key === 'all') return skills.value.length
  // 暂时所有技能都显示在全部下
  return 0
}

const filteredSkills = computed(() => {
  let list = skills.value
  if (selectedCategory.value !== 'all') {
    // 可以根据 skill.level 或其他字段过滤
    // 目前先显示全部
  }
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(s =>
      s.display_name.toLowerCase().includes(q) ||
      s.description?.toLowerCase().includes(q) ||
      s.name.toLowerCase().includes(q)
    )
  }
  return list
})

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

onMounted(loadSkills)
</script>

<style scoped>
.market-root {
  flex: 1;
  display: flex;
  overflow: hidden;
  height: 100%;
}

/* 左侧面板 */
.market-panel {
  width: 180px;
  background: var(--bg-panel);
  border-right: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.panel-header {
  padding: 14px 12px 8px;
  border-bottom: 1px solid var(--border-subtle);
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.category-list {
  padding: 8px 6px;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

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
  text-align: left;
  width: 100%;
}

.category-item:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}

.category-item.active {
  background: var(--accent-dim);
  color: var(--accent);
}

.cat-count {
  margin-left: auto;
  font-size: 10px;
  color: var(--text-tertiary);
  background: var(--bg-elevated);
  border-radius: 8px;
  padding: 1px 5px;
}

/* 主内容 */
.market-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg-app);
}

.market-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 7px 12px;
  max-width: 320px;
}

.search-icon { color: var(--text-tertiary); flex-shrink: 0; }

.search-input {
  flex: 1;
  background: none;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-size: 13px;
}

.search-input::placeholder { color: var(--text-tertiary); }

.market-loading {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-secondary);
  font-size: 13px;
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

.market-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
}

.empty-icon { color: var(--text-tertiary); }
.empty-text { font-size: 14px; color: var(--text-secondary); margin: 0; }
.empty-hint { font-size: 12px; color: var(--text-tertiary); margin: 0; }

.skill-grid {
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.skill-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  transition: border-color 0.15s;
}

.skill-card:hover { border-color: rgba(124, 106, 247, 0.3); }
.skill-card.enabled { border-color: rgba(34, 197, 94, 0.2); }

.skill-icon {
  width: 40px;
  height: 40px;
  background: var(--accent-dim);
  border: 1px solid rgba(124, 106, 247, 0.2);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 700;
  color: var(--accent);
  flex-shrink: 0;
}

.skill-info {
  flex: 1;
  min-width: 0;
}

.skill-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 3px;
}

.skill-display-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.enabled-badge {
  font-size: 10px;
  padding: 1px 6px;
  background: rgba(34, 197, 94, 0.1);
  color: var(--green);
  border-radius: 4px;
  font-weight: 500;
}

.skill-desc {
  font-size: 12px;
  color: var(--text-secondary);
  margin: 0 0 5px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-meta {
  display: flex;
  align-items: center;
  gap: 4px;
}

.meta-item { font-size: 11px; color: var(--text-tertiary); }
.meta-dot { color: var(--text-disabled); }

.skill-action { flex-shrink: 0; }

/* Toggle */
.toggle { position: relative; width: 34px; height: 18px; cursor: pointer; display: block; }
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
.toggle input:checked + .toggle-slider { background: rgba(34, 197, 94, 0.2); border-color: var(--green); }
.toggle input:checked + .toggle-slider::before { transform: translateX(16px); background: var(--green); }
</style>
