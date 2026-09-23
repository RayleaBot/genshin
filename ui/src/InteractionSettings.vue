<script setup lang="ts">
import{ref}from'vue'
const props=defineProps<{prefix:string;invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
interface Profile{ref:string;owner:{source_protocol:string;source_adapter:string;bot_id:string;actor_id:string};lists:Record<string,string[]>}
const rows=ref<Profile[]>([]),next=ref<number|null>(null),busy=ref(false),error=ref(''),candidate=ref('')
async function load(offset=0){busy.value=true;error.value='';try{const r=await props.invoke<{items:Profile[];next_offset:number|null}>('interaction.list',{offset});rows.value=offset?[...rows.value,...r.items]:r.items;next.value=r.next_offset}catch(e){error.value=e instanceof Error?e.message:'偏好读取失败。'}finally{busy.value=false}}
async function remove(){busy.value=true;error.value='';try{await props.invoke('interaction.remove',{ref:candidate.value,confirm:true});candidate.value='';await load()}catch(e){error.value=e instanceof Error?e.message:'移除未完成。'}finally{busy.value=false}}
void load()
</script>
<template><section><div class="section-heading"><h2>角色互动</h2><button :disabled="busy" @click="load()">刷新偏好</button></div><p>发送“{{prefix}}老婆设置 角色名”（老公、女友、男友、女儿、儿子等同理）指定自己的角色，“{{prefix}}老婆”从中随机展示角色卡片，未指定时从本人角色中按类型随机。戳一戳展示本人任意角色的卡片，可在查询设置中关闭。</p><p v-if="error" role="alert" class="feedback danger">{{error}}</p><p v-if="!rows.length" class="empty">没有保存的互动偏好。</p><ul><li v-for="row in rows" :key="row.ref"><strong>{{row.owner.actor_id}}</strong> · {{row.owner.source_protocol}} / {{row.owner.source_adapter}} / {{row.owner.bot_id}}<p>已设置 {{Object.keys(row.lists).length}} 份角色列表。</p><button :disabled="busy" @click="candidate=row.ref">移除互动偏好</button><div v-if="candidate===row.ref" class="actions"><span>将移除该主体的角色列表。</span><button :disabled="busy" @click="remove">确认移除</button><button :disabled="busy" @click="candidate=''">取消</button></div></li></ul><button v-if="next!==null" :disabled="busy" @click="load(next!)">更多主体</button></section></template>
