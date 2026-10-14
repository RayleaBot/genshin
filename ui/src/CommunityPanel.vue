<script setup lang="ts">
import {computed,ref,watch} from 'vue'
import BusinessView,{type View} from './BusinessView.vue'
import CloudGameTasks from './CloudGameTasks.vue'
interface Account {ref:string;account_label:string;owner:{actor_id:string};cloud_configured?:string[]}
const props=defineProps<{accounts:Account[];invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
const selected=ref(''),busy=ref(false),error=ref(''),view=ref<View|null>(null),cloudConfirmed=ref(false),rewards=ref<{id:string;message:string}[]>([])
const account=computed(()=>props.accounts.find(a=>a.ref===selected.value))
watch(selected,()=>{view.value=null;rewards.value=[];cloudConfirmed.value=false})
async function run(action:string,payload:Record<string,unknown>={}){if(busy.value||!selected.value)return;busy.value=true;error.value='';try{const r=await props.invoke<{view:View;result:{data:{items?:typeof rewards.value}}}>(action,{account_ref:selected.value,...payload});view.value=r.view;if(action==='cloudgame.rewards')rewards.value=r.result.data.items??[]}catch(e){error.value=e instanceof Error?e.message:'操作未完成。'}finally{busy.value=false;cloudConfirmed.value=false}}
</script>
<template>
 <section class="community-panel"><h2>米游社签到与云原神</h2><p class="hint">米游币查询与米游币任务由“米游社账号”插件统一提供。这里可为所选账号在米游社原神版块签到；云游戏的独立授权在“米游社账号”管理页加密配置。</p>
 <label>账号<select v-model="selected" :disabled="busy"><option value="" disabled>选择账号</option><option v-for="a in accounts" :key="a.ref" :value="a.ref">{{a.account_label}} · 用户 {{a.owner.actor_id}}</option></select></label><p v-if="error" role="alert" class="feedback danger">{{error}}</p>
 <template v-if="account"><div class="actions"><button :disabled="busy" @click="run('community.sign',{confirm:true})">为此账号在原神版块签到</button></div>
 <section><h3>云原神</h3><p class="hint">授权{{account.cloud_configured?.includes('genshin')?'已配置':'未配置'}}。钱包查询可能同时领取免费时长，因此需要明确确认；只读状态按钮显示上次保存的快照。</p><button :disabled="busy" @click="run('cloudgame.status')">查看最近云游戏状态</button><form @submit.prevent="cloudConfirmed&&run('cloudgame.sign',{confirm:true})"><fieldset :disabled="busy"><legend>查询并领取免费时长</legend><label class="check"><input v-model="cloudConfirmed" type="checkbox">确认查询此云账号钱包并领取可能发放的每日时长</label><button :disabled="!cloudConfirmed||!account.cloud_configured?.includes('genshin')">查询云钱包并签到</button></fieldset></form>
 <button :disabled="busy||!account.cloud_configured?.includes('genshin')" @click="run('cloudgame.rewards')">读取云游戏通知奖励</button><ul v-if="rewards.length"><li v-for="reward in rewards" :key="reward.id"><span>{{reward.message||reward.id}}</span><button :disabled="busy" @click="run('cloudgame.claim',{reward_id:reward.id,confirm:true})">领取此通知奖励</button></li></ul></section>
 <BusinessView v-if="view" :value="view"/>
 </template><CloudGameTasks :accounts="accounts" :invoke="invoke"/></section>
</template>
<style scoped>.community-panel{display:grid;gap:20px}.community-panel label:not(.check){display:grid;gap:8px;min-width:0}.community-panel select{min-width:0;width:100%}.community-panel fieldset{display:grid;gap:14px;border:1px solid var(--raylea-color-border);padding:16px;border-radius:12px;min-width:0}.community-panel>section{display:grid;gap:14px}.community-panel ul{list-style:none;padding:0;display:grid;gap:10px}.community-panel li{display:flex;flex-wrap:wrap;gap:10px;justify-content:space-between}</style>
