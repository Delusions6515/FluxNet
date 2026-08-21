<script setup>
import { MiuixArrowPreference, MiuixCard, MiuixSmallTitle, MiuixText } from 'miuix-vue'
import { MODULE_VERSION } from '@/api/module'
const links = [{ name: 'FluxNet 仓库', url: 'https://github.com/Delusions6515/FluxNet' }, { name: 'sing-box', url: 'https://github.com/SagerNet/sing-box' }, { name: 'miuix-vue', url: 'https://github.com/YuKongA/miuix-vue' }]
function open(url) { window.open(url, '_blank', 'noopener,noreferrer') }
</script>

<template>
  <div class="page"><MiuixSmallTitle text="FluxNet" /><MiuixCard class="section"><MiuixText type="body1">基于 sing-box 的 Android 透明代理模块。</MiuixText><MiuixText class="muted about-version" type="body2">{{ MODULE_VERSION ? `版本 ${MODULE_VERSION}` : '版本信息不可用' }}</MiuixText></MiuixCard><MiuixSmallTitle text="相关项目" /><MiuixCard class="section section--compact"><MiuixArrowPreference v-for="link in links" :key="link.name" :title="link.name" :summary="link.url" @click="open(link.url)" /></MiuixCard><MiuixSmallTitle text="许可" /><MiuixCard class="section"><MiuixText type="body2">AGPL-3.0</MiuixText></MiuixCard></div>
</template>

<style scoped>.about-version { display: block; margin-top: 8px; }</style>
