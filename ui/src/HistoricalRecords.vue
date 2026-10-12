<script setup lang="ts">
import { computed, ref } from 'vue'
import type { BotAccount } from '@rayleabot/plugin-ui'
import { botAccountPath, historicalRecords, type APIRequest, type HistoricalRecord } from './bots'

const props = defineProps<{ invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T>; apiRequest: APIRequest }>()
const rows = ref<HistoricalRecord[]>([]), total = ref(0), loaded = ref(false), busy = ref(false), error = ref(''), notice = ref('')
const accounts = ref<BotAccount[]>([]), cursor = ref<string | null>(null), query = ref(''), selected = ref('')
const candidate = ref<HistoricalRecord | null>(null), operation = ref<'remove' | 'assign'>('remove'), offset = ref(0)
const viewing = ref<HistoricalRecord | null>(null), detail = ref<{text: string; offset: number; next_offset: number | null} | null>(null)
const kinds: Record<string, string> = { group: '群设置与排行', reminder: '提醒与任务', sync_task: '每日抽卡同步', subscription: '群推送', interaction: '互动偏好', pay_log: '充值记录', simulation: '聊天模拟档案', panel_history: '面板历史', monthly: '月报档案', billing: '资产档案', cloud_archive: '面板交换档案' }
const pending = computed(() => rows.value.filter(row => !row.source_protocol || !row.bot_id).length)

async function run(work: () => Promise<void>) {
  if (busy.value) return
  busy.value = true; error.value = ''; notice.value = ''
  try { await work() } catch (cause) { error.value = cause instanceof Error ? cause.message : '历史记录操作未完成。' }
  finally { busy.value = false }
}
async function load() {
  const records: HistoricalRecord[] = []
  let next: number | null = 0, count = 0
  while (next !== null) {
    const page: { items: HistoricalRecord[]; total: number; next_offset: number | null } = await props.invoke('history.records.list', { offset: next })
    records.push(...page.items); count = page.total; next = page.next_offset
  }
  const history = await historicalRecords(records, props.apiRequest)
  rows.value = history; total.value = count; offset.value = 0; loaded.value = true; candidate.value = null; viewing.value = null; detail.value = null
}
async function read(row: HistoricalRecord, start = 0) {
  await run(async () => {
    const result = await props.invoke<{text: string; offset: number; next_offset: number | null}>('history.records.get', {kind: row.kind, ref: row.ref, revision: row.revision, offset: start})
    viewing.value = row; detail.value = result
  })
}
async function loadAccounts(more = false) {
  const params = new URLSearchParams({ source_protocol: 'onebot11', limit: '100', query: query.value })
  if (more && cursor.value) params.set('cursor', cursor.value)
  const result = await props.apiRequest<{ items: BotAccount[]; next_cursor?: string }>(`/api/bot-accounts?${params}`)
  accounts.value = more ? [...accounts.value, ...result.items] : result.items
  cursor.value = result.next_cursor ?? null
  if (!accounts.value.some(bot => bot.bot_id === selected.value)) selected.value = ''
}
async function choose(row: HistoricalRecord, action: 'remove' | 'assign') {
  await run(async () => {
    if (action === 'assign') await loadAccounts()
    candidate.value = row; operation.value = action; selected.value = ''
  })
}
async function apply() {
  const row = candidate.value
  if (!row) return
  await run(async () => {
    const payload: Record<string, unknown> = { kind: row.kind, ref: row.ref, revision: row.revision, confirm: true }
    if (operation.value === 'assign') {
      const bot = await props.apiRequest<BotAccount>(botAccountPath('onebot11', selected.value))
      payload.bot = { source_protocol: bot.source_protocol, bot_id: bot.bot_id }
    }
    const result = await props.invoke<{ scheduler_removed: boolean }>(`history.records.${operation.value}`, payload)
    const assigned = operation.value === 'assign'
    await load()
    notice.value = assigned ? '归属已保存；委托任务保持暂停，请重新创建。' : '历史记录已删除。'
    if (!result.scheduler_removed) notice.value += '调度任务待下次触发时清理。'
  })
}
</script>

<template>
  <section class="history-records separated">
    <div class="section-heading"><h2>历史数据与归属</h2><button :disabled="busy" @click="run(load)">{{ loaded ? '刷新历史记录' : '读取历史记录' }}</button></div>
    <p class="hint">缺少账号的旧记录可删除；仅 OneBot11 可选择归属。目录外的完整账号记录可查看和删除，不能改挂。</p>
    <p v-if="error" class="feedback danger" role="alert">{{ error }}</p>
    <p v-if="notice" class="feedback" role="status">{{ notice }}</p>
    <template v-if="loaded">
      <p class="hint">待归属 {{ pending }} 条 · 目录外 {{ rows.length - pending }} 条 · 插件共 {{ total }} 条归属记录</p>
      <p v-if="!rows.length" class="empty">没有待处理的历史记录。</p>
      <ul class="history-list"><li v-for="row in rows.slice(offset, offset + 50)" :key="`${row.kind}:${row.ref}`">
        <strong>{{ kinds[row.kind] }} · {{ row.label || row.ref }}</strong>
        <p class="hint">{{ row.source_protocol && row.bot_id ? `${row.source_protocol} / ${row.bot_id} · 账号不在目录中` : '待选择机器人账号' }}</p>
        <p v-if="(!row.source_protocol || !row.bot_id) && !row.assignable" class="hint">{{ row.source_protocol === 'qqofficial' ? 'QQ 官方身份不能改挂。' : '身份信息不足，仅可查看和删除。' }}</p>
        <div class="actions"><button :disabled="busy" @click="read(row)">查看记录</button><button v-if="row.assignable" :disabled="busy" @click="choose(row, 'assign')">选择归属</button><button :disabled="busy" @click="choose(row, 'remove')">删除记录</button></div>
      </li></ul>
      <div v-if="rows.length > 50" class="actions"><button :disabled="busy || offset === 0" @click="offset -= 50">上一页</button><button :disabled="busy || offset + 50 >= rows.length" @click="offset += 50">下一页</button></div>
    </template>
    <section v-if="viewing && detail"><h3>{{ kinds[viewing.kind] }} · {{ viewing.label }}</h3><pre>{{ detail.text }}</pre><div class="actions"><button :disabled="busy || detail.offset === 0" @click="read(viewing, Math.max(0, detail.offset - 12000))">上一段</button><button :disabled="busy || detail.next_offset === null" @click="read(viewing, detail.next_offset!)">下一段</button><button @click="viewing = null; detail = null">关闭记录</button></div></section>
    <section v-if="candidate" class="feedback">
      <h3>{{ operation === 'assign' ? '选择旧记录归属' : '删除历史记录' }} · {{ candidate.label }}</h3>
      <template v-if="operation === 'assign'">
        <form class="inline-form" @submit.prevent="run(() => loadAccounts())"><label>搜索 OneBot11 账号<input v-model="query" :disabled="busy"></label><button :disabled="busy">查询账号</button></form>
        <label>机器人账号<select v-model="selected" :disabled="busy"><option value="">选择账号</option><option v-for="bot in accounts" :key="bot.bot_id" :value="bot.bot_id">{{ bot.nickname || bot.bot_id }} · {{ bot.bot_id }}{{ bot.state === 'online' ? '' : ' · 离线或无连接' }}</option></select></label>
        <button v-if="cursor" :disabled="busy" @click="run(() => loadAccounts(true))">更多账号</button>
        <p class="hint">委托任务归属后保持暂停，需在账号页重新创建。</p>
      </template>
      <p v-else>将删除本插件保存的这条记录；提醒和推送同时停止。</p>
      <div class="actions"><button :disabled="busy" @click="candidate = null">取消</button><button :disabled="busy || operation === 'assign' && !selected" @click="apply">{{ operation === 'assign' ? '确认归属' : '确认删除' }}</button></div>
    </section>
  </section>
</template>

<style scoped>
.history-records,.history-list{display:grid;gap:16px}.history-list{padding:0;list-style:none}.history-list li{display:grid;gap:10px;padding-block:16px;border-bottom:1px solid var(--raylea-color-border)}pre{max-height:360px;overflow:auto;white-space:pre-wrap;overflow-wrap:anywhere}label{display:grid;gap:8px;min-width:0}select{width:100%;min-width:0}
</style>
