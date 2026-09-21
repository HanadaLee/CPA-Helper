<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import { CopyIcon } from '@lucide/vue'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { Popover, PopoverAnchor, PopoverContent, PopoverTitle } from '@/components/ui/popover'
import { useI18n } from '@/shared/i18n'
import { copyToClipboard } from '@/shared/utils/clipboard'
import { resolveTutorialVariables, variableRequirements, type TutorialContext } from '../tutorialVariables'

const props = defineProps<{ text: string; context: TutorialContext; label?: string; preview?: boolean }>()
const { t, errorText } = useI18n()
const open = ref(false)
const trigger = ref<InstanceType<typeof Button> | null>(null)
const contentID = useId()
const titleID = useId()
const busy = ref(false)
const search = ref('')
const keyChoice = ref('')
const endpointChoice = ref('')
const modelChoice = ref('')
const needs = computed(() => variableRequirements(props.text))
const keys = computed(() => props.context.apiKeys.filter((key) => key.api_key && !key.disabled))
const endpoints = computed(() => [...new Map(props.context.endpoints.filter((endpoint) => endpoint.baseURL)
  .map((endpoint) => [endpoint.baseURL, endpoint])).values()])
const choosingKey = computed(() => needs.value.key && !keyChoice.value)
const choosingEndpoint = computed(() => needs.value.endpoint && !endpointChoice.value)
const models = computed(() => [...new Map(props.context.models
  .filter((model) => model.id.trim() && (!needs.value.key || !keyChoice.value || model.sources.some((source) => source.api_key_hash === keyChoice.value)))
  .map((model) => [model.id, model])).values()])
const choices = computed(() => choosingKey.value
  ? keys.value.map((key) => ({ value: key.api_key_hash, label: key.description || t('未命名密钥', 'Unnamed key'), hint: `••••${key.api_key?.slice(-4)}` }))
  : choosingEndpoint.value ? endpoints.value.map((endpoint) => ({ value: endpoint.baseURL, label: endpoint.label, hint: endpoint.baseURL }))
    : models.value.map((model) => ({ value: model.id, label: model.id, hint: model.name !== model.id ? model.name : '' })))
const filtered = computed(() => choices.value.filter((choice) => `${choice.label} ${choice.hint}`.toLowerCase().includes(search.value.toLowerCase())))

async function finish() {
  if (needs.value.model && !choosingKey.value) {
    if (!models.value.length) {
      toast.error(t('暂无可用模型，请检查「可用模型」或刷新后重试', 'No available models. Check Available Models or refresh and retry.'))
      open.value = false
      return
    }
    if (!modelChoice.value && models.value.length === 1) modelChoice.value = models.value[0]!.id
  }
  if (choosingKey.value || choosingEndpoint.value || (needs.value.model && !modelChoice.value)) {
    search.value = ''
    open.value = true
    return
  }
  busy.value = true
  try {
    // Re-resolve against the current account data, never store raw keys in selection state.
    const key = keys.value.find((item) => item.api_key_hash === keyChoice.value)?.api_key ?? ''
    if ((needs.value.key && !key) || (needs.value.endpoint && !endpoints.value.some((item) => item.baseURL === endpointChoice.value))
      || (needs.value.model && !models.value.some((item) => item.id === modelChoice.value))) {
      toast.error(t('选项已失效，请重新选择', 'The selection is no longer available. Please select again.'))
      open.value = false
      return
    }
    await copyToClipboard(resolveTutorialVariables(props.text, key, endpointChoice.value, modelChoice.value))
    open.value = false
    toast.success(t('已复制', 'Copied'))
  } catch (error) {
    toast.error(errorText(error, '复制失败，请重试', 'Copy failed. Please retry.'))
  } finally {
    busy.value = false
  }
}

async function activate() {
  if (props.preview || busy.value) return
  if ((needs.value.key && !keys.value.length) || (needs.value.endpoint && !endpoints.value.length)) {
    toast.error(t('暂无可用密钥或 Endpoint，请先创建启用的密钥并等待地址加载', 'No available key or endpoint. Create an enabled key and wait for endpoints to load.'))
    return
  }
  if (needs.value.model) {
    busy.value = true
    try {
      await props.context.ensureModels?.()
    } catch (error) {
      toast.error(errorText(error, '加载可用模型失败，请重试', 'Failed to load available models. Please retry.'))
      return
    } finally {
      busy.value = false
    }
  }
  keyChoice.value = keys.value.length === 1 ? keys.value[0]!.api_key_hash : ''
  endpointChoice.value = endpoints.value.length === 1 ? endpoints.value[0]!.baseURL : ''
  modelChoice.value = ''
  await finish()
}

async function choose(value: string) {
  if (choosingKey.value) keyChoice.value = value
  else if (choosingEndpoint.value) endpointChoice.value = value
  else modelChoice.value = value
  await finish()
}
</script>

<template>
  <Popover v-model:open="open">
    <PopoverAnchor as-child>
      <Button ref="trigger" type="button" variant="outline" size="sm" :disabled="busy || preview" :aria-busy="busy" :title="label ? text : undefined" aria-haspopup="dialog" :aria-expanded="open" :aria-controls="open ? contentID : undefined" @click="activate">
        <Spinner v-if="busy" aria-hidden="true" data-icon="inline-start" /><CopyIcon v-else data-icon="inline-start" />{{ label || t('复制代码', 'Copy code') }}
      </Button>
    </PopoverAnchor>
    <PopoverContent :id="contentID" :aria-labelledby="titleID" align="start" class="w-80 max-w-[calc(100vw-2rem)] p-2" @close-auto-focus.prevent="trigger?.$el?.focus()">
      <PopoverTitle :id="titleID" class="px-2 py-1">{{ choosingKey ? t('选择密钥并复制', 'Select an API key') : choosingEndpoint ? t('选择 Endpoint 并复制', 'Select an endpoint') : t('选择模型并复制', 'Select a model') }}</PopoverTitle>
      <p v-if="Number(needs.key) + Number(needs.endpoint) + Number(needs.model) > 1" class="px-2 py-1 text-xs text-muted-foreground">{{ t('完成所需选择后复制整段代码', 'Complete the required selections to copy the code.') }}</p>
      <Input v-if="choices.length > 5" v-model="search" :placeholder="t('搜索', 'Search')" :aria-label="t('搜索选项', 'Search options')" class="my-2" />
      <div class="flex max-h-64 flex-col gap-1 overflow-y-auto" role="group" :aria-label="t('复制选项', 'Copy options')">
        <Button v-for="choice in filtered" :key="choice.value" type="button" variant="ghost" class="h-auto justify-start py-2" :title="`${choice.label} ${choice.hint}`" :disabled="busy" @click="choose(choice.value)">
          <span class="flex min-w-0 flex-col items-start gap-1">
            <span class="max-w-full truncate">{{ choice.label }}</span>
            <span v-if="choice.hint" class="max-w-full truncate text-xs text-muted-foreground">{{ choice.hint }}</span>
          </span>
        </Button>
        <p v-if="!filtered.length" class="p-2 text-sm text-muted-foreground">{{ t('无匹配选项', 'No matching options') }}</p>
      </div>
    </PopoverContent>
  </Popover>
</template>
