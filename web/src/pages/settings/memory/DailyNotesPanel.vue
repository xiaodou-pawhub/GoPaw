<template>
  <div class="panel">
    <div class="panel-header">
      <h2 class="panel-title">{{ t('settings.memory.dailyNotes') }}</h2>
      <div class="header-actions">
        <!-- Month navigation -->
        <n-button-group size="small">
          <n-button @click="prevMonth">‹</n-button>
          <n-button disabled>{{ currentMonthLabel }}</n-button>
          <n-button @click="nextMonth" :disabled="isCurrentMonth">›</n-button>
        </n-button-group>
      </div>
    </div>

    <div class="panel-body">
      <div class="notes-layout">
        <!-- Calendar / date list -->
        <div class="date-list">
          <div class="date-list-header">{{ currentMonthLabel }}</div>
          <div
            v-for="day in daysInMonth"
            :key="day.date"
            class="day-item"
            :class="{
              active: selectedDate === day.date,
              'has-note': day.hasNote,
              today: day.isToday
            }"
            @click="selectDate(day.date)"
          >
            <span class="day-num">{{ day.day }}</span>
            <span v-if="day.hasNote" class="note-dot" />
          </div>
        </div>

        <!-- Note editor -->
        <div class="note-editor">
          <div class="editor-header">
            <span class="editor-date">{{ formatSelectedDate }}</span>
            <div class="editor-actions">
              <n-button
                v-if="noteModified"
                size="small"
                type="primary"
                :loading="saving"
                @click="saveNote"
              >
                {{ t('settings.memory.save') }}
              </n-button>
              <n-popconfirm
                v-if="noteContent && !noteModified"
                @positive-click="deleteCurrentNote"
              >
                <template #trigger>
                  <n-button size="small" type="error" text>
                    <template #icon><n-icon><TrashOutline /></n-icon></template>
                  </n-button>
                </template>
                {{ t('settings.memory.deleteNoteConfirm') }}
              </n-popconfirm>
            </div>
          </div>

          <n-input
            v-model:value="noteContent"
            type="textarea"
            class="note-textarea"
            :placeholder="t('settings.memory.noNote')"
            :autosize="{ minRows: 12 }"
            @input="noteModified = true"
          />

          <!-- Quick append bar (today only) -->
          <div v-if="selectedDate === todayDate" class="append-bar">
            <n-input
              v-model:value="appendText"
              :placeholder="t('settings.memory.appendNote')"
              size="small"
              clearable
              @keydown.enter="handleAppend"
            />
            <n-button size="small" type="primary" :disabled="!appendText.trim()" @click="handleAppend">
              {{ t('settings.memory.append') }}
            </n-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { TrashOutline } from '@vicons/ionicons5'
import { listNotes, getNote, putNote, appendNote, deleteNote, type NoteFileInfo } from '@/api/memory'

const { t } = useI18n()
const message = useMessage()

const notes = ref<NoteFileInfo[]>([])
const selectedDate = ref('')
const noteContent = ref('')
const noteModified = ref(false)
const saving = ref(false)
const appendText = ref('')

const now = new Date()
const viewYear = ref(now.getFullYear())
const viewMonth = ref(now.getMonth() + 1) // 1-12

const todayDate = computed(() => {
  const d = new Date()
  return [
    d.getFullYear(),
    String(d.getMonth() + 1).padStart(2, '0'),
    String(d.getDate()).padStart(2, '0'),
  ].join('')
})

const currentMonthLabel = computed(() => {
  return `${viewYear.value}-${String(viewMonth.value).padStart(2, '0')}`
})

const isCurrentMonth = computed(() => {
  const d = new Date()
  return viewYear.value === d.getFullYear() && viewMonth.value === d.getMonth() + 1
})

const noteSet = computed(() => new Set(notes.value.map((n) => n.date)))

const daysInMonth = computed(() => {
  const year = viewYear.value
  const month = viewMonth.value
  const count = new Date(year, month, 0).getDate()
  const today = todayDate.value
  const result = []
  for (let d = 1; d <= count; d++) {
    const date =
      year +
      String(month).padStart(2, '0') +
      String(d).padStart(2, '0')
    result.push({
      day: d,
      date,
      hasNote: noteSet.value.has(date),
      isToday: date === today,
    })
  }
  return result.reverse() // latest first
})

const formatSelectedDate = computed(() => {
  if (!selectedDate.value) return ''
  const s = selectedDate.value
  return `${s.slice(0, 4)}-${s.slice(4, 6)}-${s.slice(6, 8)}`
})

function prevMonth() {
  if (viewMonth.value === 1) {
    viewMonth.value = 12
    viewYear.value--
  } else {
    viewMonth.value--
  }
}

function nextMonth() {
  if (isCurrentMonth.value) return
  if (viewMonth.value === 12) {
    viewMonth.value = 1
    viewYear.value++
  } else {
    viewMonth.value++
  }
}

async function selectDate(date: string) {
  if (noteModified.value) {
    // Auto-save unsaved changes
    await saveNote()
  }
  selectedDate.value = date
  await loadNote(date)
}

async function loadNote(date: string) {
  try {
    const res = await getNote(date)
    // @ts-ignore
    noteContent.value = res.content || ''
    noteModified.value = false
  } catch {
    noteContent.value = ''
    noteModified.value = false
  }
}

async function saveNote() {
  if (!selectedDate.value) return
  saving.value = true
  try {
    await putNote(selectedDate.value, noteContent.value)
    noteModified.value = false
    await loadNoteList()
  } catch (e: any) {
    message.error(e?.message || t('common.error'))
  } finally {
    saving.value = false
  }
}

async function handleAppend() {
  if (!appendText.value.trim()) return
  try {
    await appendNote(todayDate.value, appendText.value.trim())
    appendText.value = ''
    await loadNote(todayDate.value)
    await loadNoteList()
  } catch (e: any) {
    message.error(e?.message || t('common.error'))
  }
}

async function deleteCurrentNote() {
  if (!selectedDate.value) return
  try {
    await deleteNote(selectedDate.value)
    noteContent.value = ''
    noteModified.value = false
    await loadNoteList()
  } catch (e: any) {
    message.error(e?.message || t('common.error'))
  }
}

async function loadNoteList() {
  try {
    const res = await listNotes()
    // @ts-ignore
    notes.value = res.notes || []
  } catch {}
}

onMounted(async () => {
  await loadNoteList()
  // Default to today
  selectedDate.value = todayDate.value
  await loadNote(todayDate.value)
})
</script>

<style scoped lang="scss">
@use '@/styles/variables.scss' as *;

.panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-4 $spacing-6;
  border-bottom: 1px solid $color-border-light;
  flex-shrink: 0;
}

.panel-title {
  font-size: $font-size-lg;
  font-weight: $font-weight-semibold;
  color: $color-text-primary;
  margin: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: $spacing-2;
}

.panel-body {
  flex: 1;
  overflow: hidden;
}

.notes-layout {
  display: flex;
  height: 100%;
}

.date-list {
  width: 160px;
  flex-shrink: 0;
  border-right: 1px solid $color-border-light;
  overflow-y: auto;
  padding: $spacing-2;
}

.date-list-header {
  font-size: $font-size-xs;
  font-weight: $font-weight-semibold;
  color: $color-text-secondary;
  padding: $spacing-2;
  text-align: center;
}

.day-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-2 $spacing-3;
  border-radius: $radius-md;
  cursor: pointer;
  transition: background 0.15s;
  font-size: $font-size-sm;
  color: $color-text-secondary;

  &:hover {
    background: $color-bg-tertiary;
  }

  &.active {
    background: $color-gray-100;
    color: $color-primary;
    font-weight: $font-weight-medium;
  }

  &.today {
    color: $color-primary;
  }

  &.has-note .day-num {
    font-weight: $font-weight-semibold;
    color: $color-text-primary;
  }
}

.note-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: $color-primary;
}

.note-editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: $spacing-4 $spacing-6;
  gap: $spacing-3;
}

.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.editor-date {
  font-size: $font-size-base;
  font-weight: $font-weight-semibold;
  color: $color-text-primary;
}

.editor-actions {
  display: flex;
  align-items: center;
  gap: $spacing-2;
}

.note-textarea {
  flex: 1;

  :deep(.n-input__border),
  :deep(.n-input__state-border) {
    border: none !important;
  }

  :deep(textarea) {
    font-family: $font-family-mono;
    font-size: $font-size-sm;
    line-height: $line-height-relaxed;
  }
}

.append-bar {
  display: flex;
  gap: $spacing-2;
  flex-shrink: 0;
  padding-top: $spacing-2;
  border-top: 1px solid $color-border-light;
}
</style>
