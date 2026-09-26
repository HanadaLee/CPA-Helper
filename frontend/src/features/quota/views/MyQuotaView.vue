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
import QuotaProgress from '../components/QuotaProgress.vue'

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
    detail: t('当前可用限额与有效额度卡之和', 'Available base limits plus active card credit'),
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
    <div class="grid gap-4 sm:grid-cols-2">
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
          <span>{{ metric.detail }}</span>
        </CardContent>
      </Card>
    </div>
    <Card data-testid="quota-limits">
      <CardHeader>
        <CardTitle>{{ t('限额管理', 'Limit management') }}</CardTitle>
        <CardDescription>{{ t('用量同时计入日限额和周限额，任一耗尽即暂停使用限额；额度卡不受此限制。', 'Base usage counts toward both limits. Reaching either blocks base usage, but cards remain available.') }}</CardDescription>
      </CardHeader>
      <CardContent class="grid gap-6 md:grid-cols-2">
        <template v-if="quota">
          <QuotaProgress :label="t('日限额', 'Daily limit')" :remaining="quota.daily_remaining_usd" :total="quota.daily_quota_usd" :unlimited="quota.unlimited">
            <span>{{ t('重置', 'Resets') }} {{ formatDateTime(quota.daily_resets_at) }}</span>
          </QuotaProgress>
          <QuotaProgress :label="t('周限额', 'Weekly limit')" :remaining="quota.weekly_remaining_usd" :total="quota.weekly_quota_usd" :unlimited="quota.unlimited">
            <span>{{ t('重置', 'Resets') }} {{ formatDateTime(quota.weekly_resets_at) }}</span>
          </QuotaProgress>
          <p v-if="!quota.unlimited" class="text-sm text-muted-foreground md:col-span-2">
            {{ t('当前可用限额', 'Currently available base limit') }} {{ formatUsd(quota.limits_remaining_usd) }}
            · {{ t('取日、周剩余限额中的较小值', 'The smaller of the two remaining limits') }}
          </p>
        </template>
        <template v-else><Skeleton v-for="i in 2" :key="i" class="h-20 w-full" /></template>
      </CardContent>
    </Card>
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
                    <TableHead>{{ t('请求时间', 'Request time') }}</TableHead><TableHead>{{ t('费用', 'Cost') }}</TableHead><TableHead>{{ t('限额抵扣', 'Base limit deduction') }}</TableHead><TableHead>{{ t('额度卡抵扣', 'Card deductions') }}</TableHead><TableHead>{{ t('未覆盖', 'Uncovered') }}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableEmpty v-if="!charges.length" :colspan="5">
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
                    </TableCell><TableCell>{{ formatUsd(charge.limit_usd) }}</TableCell>
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
                    <TableHead>{{ t('时间', 'Time') }}</TableHead><TableHead>{{ t('来源', 'Source') }}</TableHead><TableHead>{{ t('恢复日限额', 'Daily limit restored') }}</TableHead><TableHead>
                      {{
                        t('恢复周限额', 'Weekly limit restored')
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
