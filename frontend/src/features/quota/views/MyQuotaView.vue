<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RefreshCw } from '@lucide/vue'
import { toast } from 'vue-sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  Table,
  TableHeader,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  TableEmpty,
} from '@/components/ui/table'
import TablePaginationFooter from '@/shared/ui/TablePaginationFooter.vue'
import { useI18n } from '@/shared/i18n'
import { formatDateTime, formatUsd } from '@/shared/utils/format'
import type { QuotaCharge, QuotaReset, UserQuotaStatus } from '@/shared/types/api'
import { getCurrentUserQuota } from '@/features/users/api/usersApi'
import { listQuotaCharges, listQuotaResets } from '../api/quotaApi'
import QuotaCardsPanel from '../components/QuotaCardsPanel.vue'

const { t, errorText } = useI18n()
const quota = ref<UserQuotaStatus | null>(null)
const loading = ref(true)
const cardsPanel = ref<InstanceType<typeof QuotaCardsPanel> | null>(null)
const charges = ref<QuotaCharge[]>([])
const resets = ref<QuotaReset[]>([])
const chargePage = ref(1)
const resetPage = ref(1)
const chargeTotal = ref(0)
const resetTotal = ref(0)
const metrics = computed(() => [
  {
    label: t('可用余额', 'Available balance'),
    value: quota.value?.available_usd,
    detail: t('包含每日、每周和未过期额度卡', 'Daily, weekly and unexpired card credit'),
  },
  {
    label: t('每日额度', 'Daily quota'),
    value: quota.value?.daily_remaining_usd,
    detail: quota.value
      ? t(
          `已用 ${formatUsd(quota.value.daily_used_usd)} / ${formatUsd(quota.value.daily_quota_usd)}`,
          `Used ${formatUsd(quota.value.daily_used_usd)} / ${formatUsd(quota.value.daily_quota_usd)}`,
        )
      : '',
    reset: quota.value?.daily_resets_at,
  },
  {
    label: t('每周额度', 'Weekly quota'),
    value: quota.value?.weekly_remaining_usd,
    detail: quota.value
      ? t(
          `已用 ${formatUsd(quota.value.weekly_used_usd)} / ${formatUsd(quota.value.weekly_quota_usd)}`,
          `Used ${formatUsd(quota.value.weekly_used_usd)} / ${formatUsd(quota.value.weekly_quota_usd)}`,
        )
      : '',
    reset: quota.value?.weekly_resets_at,
  },
  {
    label: t('额度卡余额', 'Card balance'),
    value: quota.value?.cards_remaining_usd,
    detail: t('按到期时间依次抵扣', 'Deducted by earliest expiration'),
    card: true,
  },
])

async function loadCharges() {
  const result = await listQuotaCharges(chargePage.value)
  charges.value = result.items
  chargeTotal.value = result.total
}
async function loadResets() {
  const result = await listQuotaResets(resetPage.value)
  resets.value = result.items
  resetTotal.value = result.total
}
async function load() {
  loading.value = true
  try {
    await Promise.all([
      getCurrentUserQuota().then((value) => {
        quota.value = value
      }),
      loadCharges(),
      loadResets(),
    ])
  } catch (error) {
    toast.error(errorText(error, '加载配额失败', 'Failed to load quota'))
  } finally {
    loading.value = false
  }
}
async function refresh() {
  await Promise.all([load(), cardsPanel.value?.refresh()])
}
watch(chargePage, () => {
  void loadCharges().catch((error) =>
    toast.error(errorText(error, '加载扣款记录失败', 'Failed to load deductions')),
  )
})
watch(resetPage, () => {
  void loadResets().catch((error) =>
    toast.error(errorText(error, '加载重置记录失败', 'Failed to load resets')),
  )
})
onMounted(load)
</script>

<template>
  <section class="page">
    <div class="page-toolbar">
      <h1 data-page-title class="page-title">{{ t('我的配额', 'My Quota') }}</h1>
      <Button variant="outline" :disabled="loading" @click="refresh">
        <RefreshCw data-icon="inline-start" />{{ t('刷新', 'Refresh') }}
      </Button>
    </div>
    <Alert v-if="quota?.paused" variant="destructive">
      <AlertDescription>
        {{
          t(
            '可用额度已用尽，API 密钥已暂停。收到额度卡、使用重置卡或进入新的额度周期后会自动恢复。',
            'API keys are paused because credit is exhausted. They resume after receiving credit, using a reset card, or entering a new quota period.',
          )
        }}
      </AlertDescription>
    </Alert>
    <Alert v-if="quota?.sync_error" variant="destructive">
      <AlertDescription>
        {{ t('密钥状态同步失败，系统将自动重试：', 'Key synchronization failed and will retry: ')
        }}{{ quota.sync_error }}
      </AlertDescription>
    </Alert>
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <Card v-for="metric in metrics" :key="metric.label">
        <CardHeader>
          <CardDescription>{{ metric.label }}</CardDescription><CardTitle v-if="quota">
            {{
              quota.unlimited && !metric.card
                ? t('无限制', 'Unlimited')
                : formatUsd(metric.value ?? 0)
            }}
          </CardTitle><Skeleton v-else class="h-7 w-28" />
        </CardHeader>
        <CardContent class="flex flex-col gap-2 text-sm text-muted-foreground">
          <span>{{ metric.detail }}</span><span v-if="metric.reset">{{ t('下次重置', 'Next reset') }} {{ formatDateTime(metric.reset) }}</span>
        </CardContent>
      </Card>
    </div>
    <QuotaCardsPanel ref="cardsPanel" @changed="load" />
    <Card>
      <CardHeader>
        <CardTitle>{{ t('配额记录', 'Quota history') }}</CardTitle><CardDescription>
          {{
            t(
              '每日 0 点、每周一 0 点按北京时间重置。使用重置卡不会延后这些时间。',
              'Resets occur at midnight daily and on Mondays, Beijing time. Reset cards do not postpone these times.',
            )
          }}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs default-value="charges" class="flex flex-col gap-4">
          <TabsList class="self-start">
            <TabsTrigger value="charges">{{ t('扣款记录', 'Deductions') }}</TabsTrigger><TabsTrigger value="resets">{{ t('重置记录', 'Resets') }}</TabsTrigger>
          </TabsList>
          <TabsContent value="charges">
            <div class="overflow-hidden rounded-lg border border-border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{{ t('请求时间', 'Request time') }}</TableHead><TableHead>{{ t('费用', 'Cost') }}</TableHead><TableHead>{{ t('每日', 'Daily') }}</TableHead><TableHead>{{ t('每周', 'Weekly') }}</TableHead><TableHead>{{ t('额度卡抵扣', 'Card deductions') }}</TableHead><TableHead>{{ t('未覆盖', 'Uncovered') }}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableEmpty v-if="!charges.length" :colspan="6">
                    {{
                      loading ? t('加载中', 'Loading') : t('暂无扣款记录', 'No deductions')
                    }}
                  </TableEmpty><TableRow v-for="charge in charges" :key="charge.id">
                    <TableCell>{{ formatDateTime(charge.timestamp) }}</TableCell><TableCell>
                      <div class="flex flex-col gap-1">
                        <span>{{
                          charge.unpriced ? t('未定价', 'Unpriced') : formatUsd(charge.amount_usd)
                        }}</span><span v-if="charge.legacy_usd > 0" class="text-xs text-muted-foreground">{{ t('原额度扣款', 'Legacy quota deduction') }} {{ formatUsd(charge.legacy_usd) }}</span>
                      </div>
                    </TableCell><TableCell>{{ formatUsd(charge.daily_usd) }}</TableCell><TableCell>{{ formatUsd(charge.weekly_usd) }}</TableCell>
                    <TableCell>
                      <div class="flex flex-col gap-1">
                        <span>{{ formatUsd(charge.cards_usd) }}</span><span
                          v-for="card in charge.cards"
                          :key="card.card_id"
                          class="text-xs text-muted-foreground"
                        >{{ card.name || t('额度卡', 'Credit card') }} #{{ card.card_id }} ·
                          {{ formatUsd(card.amount_usd) }}</span>
                      </div>
                    </TableCell><TableCell>{{ formatUsd(charge.uncovered_usd) }}</TableCell>
                  </TableRow>
                </TableBody>
              </Table><TablePaginationFooter
                v-model:page="chargePage"
                :page-size="20"
                :page-size-options="[20]"
                :total="chargeTotal"
              />
            </div>
          </TabsContent>
          <TabsContent value="resets">
            <div class="overflow-hidden rounded-lg border border-border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{{ t('时间', 'Time') }}</TableHead><TableHead>{{ t('来源', 'Source') }}</TableHead><TableHead>{{ t('恢复每日额度', 'Daily credit restored') }}</TableHead><TableHead>
                      {{
                        t('恢复每周额度', 'Weekly credit restored')
                      }}
                    </TableHead>
                  </TableRow>
                </TableHeader><TableBody>
                  <TableEmpty v-if="!resets.length" :colspan="4">
                    {{
                      t('暂无重置记录', 'No resets')
                    }}
                  </TableEmpty><TableRow v-for="reset in resets" :key="reset.id">
                    <TableCell>{{ formatDateTime(reset.created_at) }}</TableCell><TableCell>
                      {{
                        reset.card_id
                          ? t(`重置卡 #${reset.card_id}`, `Reset card #${reset.card_id}`)
                          : t('管理员重置', 'Administrator reset')
                      }}
                    </TableCell><TableCell>{{ formatUsd(reset.daily_used_usd) }}</TableCell><TableCell>{{ formatUsd(reset.weekly_used_usd) }}</TableCell>
                  </TableRow>
                </TableBody>
              </Table><TablePaginationFooter
                v-model:page="resetPage"
                :page-size="20"
                :page-size-options="[20]"
                :total="resetTotal"
              />
            </div>
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  </section>
</template>
