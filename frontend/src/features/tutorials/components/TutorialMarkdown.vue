<script setup lang="ts">
import { computed, defineComponent, h, type VNodeChild } from 'vue'
import MarkdownIt from 'markdown-it'
import { Separator } from '@/components/ui/separator'
import { useI18n } from '@/shared/i18n'
import TutorialCopy from './TutorialCopy.vue'
import { emptyTutorialContext, variablePattern, type TutorialContext } from '../tutorialVariables'

const props = withDefaults(defineProps<{ content: string; context?: TutorialContext; preview?: boolean }>(), {
  context: () => emptyTutorialContext, preview: false,
})
const { t } = useI18n()
// No raw HTML, executable links, or remote tracking images. Render tokens as Vue nodes,
// not v-html, so variables can use the same accessible Popover as the rest of the app.
const markdown = new MarkdownIt({ html: false, linkify: true }).disable('image')
type Token = ReturnType<typeof markdown.parse>[number]
const tokens = computed(() => markdown.parse(props.content, {}))

function variableText(text: string): VNodeChild[] {
  const output: VNodeChild[] = []
  let cursor = 0
  for (const match of text.matchAll(new RegExp(variablePattern))) {
    output.push(text.slice(cursor, match.index))
    output.push(h(TutorialCopy, {
      text: match[0], context: props.context, preview: props.preview,
      label: match[1] === 'api_key' ? t('API 密钥', 'API key') : match[1] === 'responses_url' ? 'Responses URL'
        : match[1] === 'chat_completions_url' ? t('聊天 URL', 'Chat URL')
          : match[1] === 'claude_messages_url' ? 'Claude URL' : t('基础 URL', 'Base URL'),
    }))
    cursor = match.index! + match[0].length
  }
  output.push(text.slice(cursor))
  return output
}

function renderTokens(items: Token[]): VNodeChild[] {
  let index = 0
  function walk(): VNodeChild[] {
    const nodes: VNodeChild[] = []
    while (index < items.length) {
      const token = items[index++]!
      if (token.nesting === -1) break
      if (token.type === 'inline') nodes.push(...renderTokens(token.children ?? []))
      else if (token.type === 'text') nodes.push(...variableText(token.content))
      else if (token.type === 'softbreak') nodes.push('\n')
      else if (token.type === 'code_inline') nodes.push(h('code', variableText(token.content)))
      else if (token.type === 'fence' || token.type === 'code_block') {
        nodes.push(h('div', { class: 'tutorial-code' }, [
          h('div', { class: 'tutorial-code-header' }, [h('span', token.info.trim()), h(TutorialCopy, { text: token.content, context: props.context, preview: props.preview })]),
          h('pre', [h('code', variableText(token.content))]),
        ]))
      } else if (token.type === 'hr') nodes.push(h(Separator))
      else if (token.tag) {
        const attrs = Object.fromEntries(token.attrs ?? [])
        if (token.tag === 'a') Object.assign(attrs, { target: '_blank', rel: 'noopener noreferrer' })
        nodes.push(h(token.tag, attrs, token.nesting === 1 ? walk() : token.content))
      } else nodes.push(token.content)
    }
    return nodes
  }
  return walk()
}

const Content = defineComponent({ setup: () => () => h('div', { class: 'tutorial-prose' }, renderTokens(tokens.value)) })
</script>

<template><Content /></template>

<style>
.tutorial-prose { min-width: 0; overflow-wrap: anywhere; line-height: 1.75; }
.tutorial-prose > :first-child { margin-top: 0; }
.tutorial-prose h1, .tutorial-prose h2, .tutorial-prose h3 { margin: 1.5rem 0 .75rem; font-weight: 600; line-height: 1.4; }
.tutorial-prose h1 { font-size: 1.35rem; }
.tutorial-prose h2 { font-size: 1.125rem; }
.tutorial-prose p, .tutorial-prose ul, .tutorial-prose ol, .tutorial-prose blockquote { margin: .75rem 0; }
.tutorial-prose ul { list-style: disc; padding-left: 1.5rem; }
.tutorial-prose ol { list-style: decimal; padding-left: 1.5rem; }
.tutorial-prose li + li { margin-top: .35rem; }
.tutorial-prose a { color: var(--primary); text-decoration: underline; text-underline-offset: 3px; }
.tutorial-prose blockquote { border-left: 3px solid var(--border); padding-left: 1rem; color: var(--muted-foreground); }
.tutorial-prose code { font-family: var(--font-mono); font-size: .875em; }
.tutorial-prose :not(pre) > code { background: var(--muted); border-radius: .3rem; padding: .1rem .3rem; }
.tutorial-prose .tutorial-code { margin: 1rem 0; border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
.tutorial-prose .tutorial-code-header { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: .5rem .75rem; background: var(--muted); font-size: .75rem; color: var(--muted-foreground); }
.tutorial-prose pre { margin: 0; padding: 1rem; overflow-x: auto; white-space: pre; }
.tutorial-prose pre code { background: transparent; }
.tutorial-prose table { display: block; max-width: 100%; overflow-x: auto; border-collapse: collapse; }
.tutorial-prose th, .tutorial-prose td { padding: .5rem .75rem; border: 1px solid var(--border); }
.tutorial-prose button { vertical-align: middle; }
</style>
