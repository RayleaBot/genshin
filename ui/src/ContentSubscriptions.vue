<script setup lang="ts">
import {ref} from 'vue'
const props=defineProps<{prefix:string;invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
interface Subscription{ref:string;group_id:string;kinds:string[];warned_on?:string;last_code?:string}
const subscriptions=ref<Subscription[]>([]),subError=ref(''),subBusy=ref(false),removeRef=ref('')
async function loadSubscriptions(){subBusy.value=true;subError.value='';try{subscriptions.value=(await props.invoke<{items:Subscription[]}>('content.subscription.list')).items}catch(e){subError.value=e instanceof Error?e.message:'推送列表读取失败。'}finally{subBusy.value=false}}
async function remove(){subBusy.value=true;subError.value='';try{await props.invoke('content.subscription.remove',{ref:removeRef.value,confirm:true});removeRef.value='';await loadSubscriptions()}catch(e){subError.value=e instanceof Error?e.message:'移除未完成。'}finally{subBusy.value=false}}
void loadSubscriptions()
</script>
<template><section><div class="section-heading"><h2>群推送</h2><button :disabled="subBusy" @click="loadSubscriptions">刷新</button></div><p>群管理员在群里发送“{{prefix}}开启公告推送”“{{prefix}}开启资讯推送”或“{{prefix}}原神开启到期活动推送”开启，“开启”换成“关闭”即停止。每五分钟推送两小时内的一条新帖，每天十点后推送一天内结束的活动；超级管理员发送“{{prefix}}推送公告”立即检查。</p><p v-if="subError" role="alert" class="feedback danger">{{subError}}</p><p v-if="!subscriptions.length" class="empty">没有开启推送的群。</p><ul class="post-list"><li v-for="s in subscriptions" :key="s.ref"><strong>群 {{s.group_id}} · {{s.kinds.map(k=>({announce:'公告',info:'资讯',expiry:'到期活动'} as Record<string,string>)[k]).join('、')}}</strong><span>{{s.last_code||'等待首次检查'}}<template v-if="s.warned_on"> · 到期活动已于 {{s.warned_on}} 推送</template></span><button :disabled="subBusy" @click="removeRef=s.ref">移除此群的推送</button><div v-if="removeRef===s.ref" class="actions"><span>确认停止此群的全部推送？</span><button :disabled="subBusy" @click="remove">确认移除</button><button :disabled="subBusy" @click="removeRef=''">取消</button></div></li></ul></section></template>
<style scoped>.post-list{padding:0;list-style:none;display:grid;gap:16px}.post-list li{display:grid;gap:8px;padding-block:12px;border-bottom:1px solid var(--raylea-color-border)}</style>
