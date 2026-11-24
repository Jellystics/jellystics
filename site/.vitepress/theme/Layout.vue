<script setup lang="ts">
import DefaultTheme from 'vitepress/theme'
import { useData, useRouter, withBase, inBrowser } from 'vitepress'
import { onMounted, watch } from 'vue'

// Language choice: the first visit of the English home page follows the browser language (fr, es);
// any later visit keeps the language the reader last used.
const KEY = 'jellystics-docs-lang'
const { lang, page } = useData()
const router = useRouter()

function read(): string | null {
  try { return localStorage.getItem(KEY) } catch { return null }
}
function save(value: string) {
  try { localStorage.setItem(KEY, value) } catch { /* storage blocked */ }
}

onMounted(() => {
  if (!inBrowser) return
  const isRootHome = page.value.relativePath === 'index.md'
  const stored = read()
  if (isRootHome && !stored) {
    const nav = (navigator.languages?.[0] || navigator.language || 'en').slice(0, 2).toLowerCase()
    if (nav === 'fr' || nav === 'es') {
      save(nav)
      router.go(withBase(`/${nav}/`))
      return
    }
  }
  save(lang.value.slice(0, 2))
})

watch(lang, (value) => { if (inBrowser) save(value.slice(0, 2)) })
</script>

<template>
  <DefaultTheme.Layout />
</template>
