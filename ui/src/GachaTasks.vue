<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
interface Round {
  ref: string; role: { uid: string; nickname: string }; full: boolean; daily: boolean
  state: 'running' | 'completed' | 'failed' | 'canceled'; last_code?: string; started_ms: number
  progress: { pages: number; fetched: number; result?: { added: number; total: number } }
}
interface Task {
  ref: string; role: { uid: string; nickname: string }; owner: { actor_id: string }
  hour: number; full: boolean; notify: boolean
  state: string; last_code: string; next_check_ms: number; expires_at_ms: number
  result?: { added: number; total: number }
}
const props = defineProps<{ choices: { key: string; label: string; account: { ref: string }; role: { ref: string } }[]; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const selected = ref(''), kind = ref('once'), days = ref(30), hour = ref(8), full = ref(false), notify = ref(false)
const rounds = ref<Round[]>([]), tasks = ref<Task[]>([]), busy = ref(false), loading = ref(false), error = ref(''), loadError = ref(''), notice = ref('')
let disposed = false, revision = 0
const roundStates: Record<string, string> = { running: '同步中', completed: '已完成', failed: '未完成', canceled: '已取消' }
const states: Record<string, string> = { creating: '创建中', waiting: '等待调度', running: '等待下一次调度读取', paused: '已暂停', expired: '已到期' }
const reasons: Record<string, string> = {
  queued: '已排队', sync_completed: '本轮同步完成', canceled: '本轮已取消', expired: '授权已到期，请移除后重新创建', archive_removed: '档案已移除，同步已停止',
  'sync_completed.notification_failed': '记录已保存，结果通知未能发送', 'plugin.event_timeout': '超过后台时限，本轮未合并', 'plugin.game_sync_conflict': '档案发生变化，请确认后重新运行', 'plugin.game_sync_invalid': '官方分页或记录异常，原档案保留',
  'plugin.account_delegation_denied': '委托已撤销或失效，请重新创建', 'plugin.account_caller_denied': '本游戏的账号授权已撤销', 'plugin.account_not_found': '账号已移除', 'plugin.account_role_denied': '角色授权已变更',
  'plugin.upstream_auth_invalid': '账号登录已失效', 'plugin.upstream_device_required': '请先在官方应用完成设备验证', 'plugin.upstream_challenge_required': '请先在官方应用完成验证',
  'plugin.vault_locked': '账号库已锁定，稍后重试', 'plugin.service_unavailable': '账号插件不可用，稍后重试', 'plugin.game_sync_busy': '同时进行的同步过多，稍后重试', 'plugin.account_delegation_throttled': '分页请求过于频繁，稍后重试',
}
function date(ms: number) { return ms ? new Date(ms).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' }) : '等待下一次调度' }
async function load() {
  const request = ++revision; loading.value = true
  try {
    const result = await props.invoke<{ items: Round[]; tasks: Task[] }>('gacha.task.list')
    if (!disposed && request === revision) { rounds.value = result.items; tasks.value = result.tasks; loadError.value = '' }
  } catch (cause) { if (!disposed && request === revision) loadError.value = cause instanceof Error ? cause.message : '后台任务读取失败。' }
  finally { if (!disposed && request === revision) loading.value = false }
}
async function change(action: string, ref?: string) {
  if (busy.value) return
  const choice = props.choices.find(c => c.key === selected.value)
  if (action === 'create' && !choice) return
  const name = action === 'create' ? (kind.value === 'daily' ? 'gacha.task.create' : 'gacha.task.start') : action
  busy.value = true; revision++; loading.value = false; error.value = ''; notice.value = ''
  try {
    const result = await props.invoke<{ delegation_revoked?: boolean }>(name, ref ? { ref, confirm: true } : {
      account_ref: choice!.account.ref, role_ref: choice!.role.ref, days: Number(days.value), hour: Number(hour.value), full: full.value, notify: notify.value, confirm: true,
    })
    if (disposed) return
    notice.value = {
      'gacha.task.start': '后台同步已开始，关闭页面仍会继续。', 'gacha.task.create': '每日同步已开启。', 'gacha.task.run': '任务已排队，下一次调度时重新运行。', 'gacha.task.cancel': '本轮后台同步已取消。',
    }[name] ?? (result.delegation_revoked ? '每日同步与委托已停止。' : '每日同步已停止；账号服务暂未确认撤销，原委托等待到期。')
    await load()
  } catch (cause) { if (!disposed) error.value = cause instanceof Error ? cause.message : '任务操作未完成。' }
  finally { if (!disposed) busy.value = false }
}
watch(kind, value => { if (value === 'daily') days.value = 30 })
void load()
const timer = setInterval(() => { if (!busy.value && !loading.value) void load() }, 5000)
// Closing this view only stops polling; syncs run in the plugin.
onUnmounted(() => { disposed = true; revision++; clearInterval(timer) })
</script>
<template>
  <section class="gacha-tasks separated">
    <div class="section-heading"><h3>后台与每日同步</h3><button :disabled="busy || loading" @click="load">刷新任务</button></div>
    <p class="hint">关闭页面后仍会继续。每一轮在插件后台一次读完，相邻两页至少间隔一秒，全部卡池读取成功后合并；超过后台时限（默认 15 分钟）或插件重启时本轮作废，已有档案保留。删除档案会停止并暂停对应同步。</p>
    <details><summary>开始后台同步</summary>
      <form @submit.prevent="change('create')"><fieldset :disabled="busy"><legend class="sr-only">后台同步设置</legend>
        <label>后台同步角色<select v-model="selected" required><option value="" disabled>选择已授权角色</option><option v-for="c in choices" :key="c.key" :value="c.key">{{ c.label }}</option></select></label>
        <div class="task-fields"><label>同步安排<select v-model="kind"><option value="once">单次后台同步</option><option value="daily">每日同步</option></select></label><label v-if="kind === 'daily'">授权天数<input v-model.number="days" type="number" min="1" max="90" step="1" required></label><label v-if="kind === 'daily'">北京时间（小时）<input v-model.number="hour" type="number" min="0" max="23" step="1" required></label></div>
        <label class="check"><input v-model="full" type="checkbox">{{ kind === 'daily' ? '每轮' : '' }}全量读取官方仍保留的历史（默认只读新增记录）</label>
        <label class="check"><input v-model="notify" type="checkbox">{{ kind === 'daily' ? '每轮' : '' }}完成后私聊账号所属用户，通知结果</label>
        <button class="primary" type="submit" :disabled="!selected">{{ kind === 'daily' ? '开启每日同步' : '开始单次后台同步' }}</button>
      </fieldset></form>
    </details>
    <p v-if="error || loadError" role="alert" class="feedback danger">{{ error || loadError }}</p><p v-if="notice" role="status" class="feedback">{{ notice }}</p>
    <h4>进行中与最近的同步</h4>
    <p v-if="!rounds.length" class="hint">插件本次运行以来尚无后台同步；聊天中的“更新抽卡记录”与每日同步的每一轮也会列在这里。</p>
    <ul class="task-list"><li v-for="round in rounds" :key="round.ref">
      <div class="task-heading"><strong>{{ round.role.nickname }} · {{ round.role.uid }}</strong><span>{{ roundStates[round.state] || '状态待确认' }}</span></div>
      <p>{{ round.daily ? '每日同步' : '单次后台同步' }} · {{ round.full ? '全量' : '增量' }} · 开始于 {{ date(round.started_ms) }}（北京时间）</p>
      <p>已读取 {{ round.progress.pages || 0 }} 页、{{ round.progress.fetched || 0 }} 条。<span v-if="round.progress.result">新增 {{ round.progress.result.added }} 条，共 {{ round.progress.result.total }} 条。</span></p>
      <p v-if="round.last_code" class="hint">{{ reasons[round.last_code] || '本轮同步未完成，可查看插件诊断后重试。' }}</p>
      <div v-if="round.state === 'running'" class="actions"><button :disabled="busy" @click="change('gacha.task.cancel', round.ref)">取消本轮同步</button></div>
    </li></ul>
    <h4>每日同步任务</h4>
    <p v-if="!tasks.length" class="hint">尚无每日同步任务。</p>
    <ul class="task-list"><li v-for="task in tasks" :key="task.ref">
      <div class="task-heading"><strong>{{ task.role.nickname }} · {{ task.role.uid }}</strong><span>{{ states[task.state] || '状态待确认' }}</span></div>
      <p>每天北京时间 {{ task.hour }}:00 · {{ task.full ? '全量' : '增量' }} · {{ task.notify ? `通知 ${task.owner.actor_id}` : '不发送结果通知' }}</p>
      <p v-if="task.result">最近完成：新增 {{ task.result.added }} 条，共 {{ task.result.total }} 条。</p>
      <p class="hint">{{ reasons[task.last_code] || (task.last_code ? '本次请求未完成，可查看插件诊断后重试。' : '等待首次运行') }}<br>授权到期：{{ date(task.expires_at_ms) }}（北京时间）<template v-if="['waiting', 'running'].includes(task.state)"><br>下次可检查：{{ date(task.next_check_ms) }}</template></p>
      <div class="actions"><button :disabled="busy || ['creating', 'running', 'expired'].includes(task.state) || task.expires_at_ms <= Date.now()" @click="change('gacha.task.run', task.ref)">重新运行此同步</button><button :disabled="busy" @click="change('gacha.task.remove', task.ref)">停止此同步任务</button></div>
    </li></ul>
  </section>
</template>
<style scoped>
.gacha-tasks{display:grid;gap:20px}.gacha-tasks .section-heading{margin-bottom:0}.gacha-tasks h4{margin:0}.gacha-tasks fieldset{display:grid;gap:16px;padding:16px 0 0;border:0;min-width:0}.gacha-tasks label:not(.check){display:grid;gap:8px;min-width:0}.task-fields{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}.task-list{list-style:none;padding:0;margin:0}.task-list li{display:grid;gap:12px;padding:20px 0;border-bottom:1px solid var(--raylea-color-border)}.task-heading{display:flex;justify-content:space-between;flex-wrap:wrap;gap:8px}.task-heading span{font-weight:600}@media(max-width:720px){.task-fields{grid-template-columns:1fr}}
</style>
