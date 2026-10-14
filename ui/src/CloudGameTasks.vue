<script setup lang="ts">
import {computed,ref,watch} from 'vue'
interface Account {ref:string;account_label:string;owner:{actor_id:string};cloud_configured?:string[]}
interface Task {ref:string;kind:string;account_ref:string;owner:{actor_id:string};role:{nickname:string};enabled:boolean;once:boolean;hour:number;minute:number;last_code:string;next_check_ms:number;expires_at_ms:number}
const props=defineProps<{accounts:Account[];invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
const selected=ref(''),once=ref(true),hour=ref(8),minute=ref(0),days=ref(30),notify=ref(false),confirmed=ref(false),busy=ref(false),error=ref(''),notice=ref(''),items=ref<Task[]>([])
const account=computed(()=>props.accounts.find(a=>a.ref===selected.value))
watch([selected,once,hour,minute,days,notify],()=>{confirmed.value=false})
async function load(){items.value=(await props.invoke<{items:Task[]}>('cloudgame.task.list')).items}
async function run(fn:()=>Promise<void>){if(busy.value)return;busy.value=true;error.value='';notice.value='';try{await fn()}catch(e){error.value=e instanceof Error?e.message:'账号任务操作未完成。'}finally{busy.value=false}}
async function create(){if(!confirmed.value||!account.value)return;await run(async()=>{await props.invoke('cloudgame.task.create',{account_ref:selected.value,once:once.value,hour:Number(hour.value),minute:Number(minute.value),days:once.value?1:Number(days.value),notify:notify.value,confirm:true});confirmed.value=false;await load();notice.value='任务已开启。'})}
async function remove(task:Task){await run(async()=>{const r=await props.invoke<{delegation_revoked:boolean}>('cloudgame.task.remove',{ref:task.ref});await load();notice.value=r.delegation_revoked?'任务与委托已停止。':'本地任务已停止；账号服务暂未确认撤销，原委托将自动到期。'})}
const labels:Record<string,string>={cloud_attempted:'已尝试云签到，结果待核对',cloud_completed:'云钱包查询已完成',expired:'委托已到期','plugin.account_cloud_unavailable':'尚未配置云授权','plugin.upstream_cloud_unavailable':'云授权需处理','plugin.upstream_challenge_required':'需要官方验证','plugin.upstream_device_required':'需要设备验证'}
void run(load)
</script>
<template>
 <section class="account-tasks"><h3>云游戏任务</h3><p class="hint">默认关闭。到点查询一次云钱包并领取可能发放的时长；结果不确定时不重放。</p><p v-if="error" role="alert" class="feedback danger">{{error}}</p><p v-if="notice" role="status">{{notice}}</p>
 <form @submit.prevent="create"><fieldset :disabled="busy"><legend>新账号任务</legend><label>任务账号<select v-model="selected" required><option value="" disabled>选择账号</option><option v-for="a in accounts" :key="a.ref" :value="a.ref">{{a.account_label}} · 用户 {{a.owner.actor_id}}</option></select></label><label>执行方式<select v-model="once"><option :value="true">执行一次</option><option :value="false">每天执行</option></select></label>
 <div v-if="!once" class="fields"><label>北京时间（时）<input v-model="hour" type="number" min="0" max="23" required></label><label>北京时间（分）<input v-model="minute" type="number" min="0" max="59" required></label><label>任务授权天数<input v-model="days" type="number" min="1" max="90" required></label></div>
 <p v-if="account&&!account.cloud_configured?.includes('genshin')" class="hint">请先到账号管理页配置此账号的加密云授权。</p>
 <label class="check"><input v-model="notify" type="checkbox">完成后私聊通知账号所属用户</label><label class="check"><input v-model="confirmed" type="checkbox">确认以所选账号执行上述操作，并创建有期限的定时委托</label><button class="primary" :disabled="!confirmed||!selected||!account?.cloud_configured?.includes('genshin')">开启账号任务</button>
 </fieldset></form>
 <div class="section-heading"><h4>已保存的账号任务</h4><button :disabled="busy" @click="run(load)">刷新账号任务进度</button></div><p v-if="!items.length" class="hint">尚未创建账号任务。</p><ul><li v-for="task in items" :key="task.ref"><strong>{{task.role.nickname}} · 云游戏签到</strong><p>{{task.enabled?'已开启':'已停止'}} · {{task.once?'执行一次':`每日 ${String(task.hour).padStart(2,'0')}:${String(task.minute||0).padStart(2,'0')}（北京时间）`}} · 所属用户 {{task.owner.actor_id}}</p><p>{{labels[task.last_code]||task.last_code||'等待执行'}}</p><p class="hint">到期 {{new Date(task.expires_at_ms).toLocaleString()}}</p><button :disabled="busy" @click="remove(task)">停止并移除此账号任务</button></li></ul>
 </section>
</template>
<style scoped>.account-tasks{display:grid;gap:18px;border-top:1px solid var(--raylea-color-border);padding-top:20px}.account-tasks fieldset{display:grid;gap:14px;border:1px solid var(--raylea-color-border);padding:16px;border-radius:12px;min-width:0}.account-tasks label:not(.check){display:grid;gap:8px;min-width:0}.account-tasks select{min-width:0;width:100%}.fields{display:flex;gap:12px;flex-wrap:wrap}.fields label{flex:1 1 130px}.account-tasks ul{list-style:none;padding:0;display:grid;gap:16px}.account-tasks li{border-top:1px solid var(--raylea-color-border);padding-top:14px;overflow-wrap:anywhere}.account-tasks li p{margin:8px 0}</style>
