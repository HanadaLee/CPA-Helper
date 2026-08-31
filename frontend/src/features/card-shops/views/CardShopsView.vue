<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { InputGroup, InputGroupAddon, InputGroupInput } from '@/components/ui/input-group'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import {
  AlertTriangle,
  Clock3,
  ExternalLink,
  GripVertical,
  MessageCircle,
  Package,
  Plus as PlusIcon,
  RefreshCw,
  RotateCcw,
  Search,
  Settings2,
  ShoppingBag,
  Star,
  Store,
  Trash2,
  X,
} from '@lucide/vue'

import {
  getCardShopFavorites,
  getCardShopTags,
  getCardShops,
  updateCardShopTags,
  updateCardShopFavorite,
} from '@/features/card-shops/api/cardShopsApi'
import { useCurrentUser } from '@/features/auth/state/currentUser'
import type { CardShop, CardShopProductItem } from '@/shared/types/api'
import { useI18n } from '@/shared/i18n'
import { formatDateTime, formatInteger } from '@/shared/utils/format'

type CardShopSortKey = 'relevance' | 'priceAsc' | 'stockDesc' | 'recent' | 'salesDesc'

interface CardShopRow {
  shop: CardShop
  shopKey: string
  isFavorite: boolean
  visibleProducts: CardShopProductItem[]
  productCount: number
  matchedProductCount: number
  minPrice: number | null
  totalStock: number
  sales: number
  latestMs: number
  score: number
}

interface CardShopViewPreferences {
  version: 1
  searchDraft: string
  searchTerms: string[]
  sortKey: CardShopSortKey
  favoriteOnly: boolean
}

type CachedCardShopViewPreferences = Partial<Omit<CardShopViewPreferences, 'version'>>

const DEFAULT_QUICK_TAGS = [
  'Codex',
  'GPT',
  'Claude',
  'Gemini',
  'Kiro',
  'Plus',
  'Team',
  'PayPal',
  'Pix',
  'GoPay',
  '接码',
  '实体卡',
] as const
const MAX_QUICK_TAGS = 30
const MAX_QUICK_TAG_LENGTH = 32
const CARD_SHOP_VIEW_PREFERENCES_VERSION = 1
const CARD_SHOP_VIEW_PREFERENCES_STORAGE_PREFIX = 'cpa-helper-card-shops-view-preferences'
const DEFAULT_SORT_KEY: CardShopSortKey = 'salesDesc'
const DEFAULT_EMPTY_TEXT = '-'

const { currentLanguage, errorText, t } = useI18n()
const { currentUser } = useCurrentUser()
const shops = ref<CardShop[]>([])
const fetchedAt = ref<string | null>(null)
const isLoading = ref(false)
const loadError = ref<string | null>(null)
const favoriteError = ref<string | null>(null)
const favoriteOnly = ref(false)
const favoriteKeys = ref<Set<string>>(new Set())
const favoriteUpdatingKeys = ref<Set<string>>(new Set())
const searchDraft = ref('')
const searchTerms = ref<string[]>([])
const sortKey = ref<CardShopSortKey>(DEFAULT_SORT_KEY)
const quickTags = ref<string[]>(defaultQuickTags())
const isTagModalOpen = ref(false)
const editableTags = ref<string[]>([])
const newTagDraft = ref('')
const tagEditorError = ref<string | null>(null)
const isSavingTags = ref(false)
const draggingTagIndex = ref<number | null>(null)
const activePreferenceUserID = ref<number | null>(null)
const isRestoringViewPreferences = ref(false)

const sortOptions = computed<Array<{ label: string; value: CardShopSortKey }>>(() => [
  { label: t('综合排序', 'Best match'), value: 'relevance' },
  { label: t('价格低到高', 'Price: low to high'), value: 'priceAsc' },
  { label: t('库存多到少', 'Stock: high to low'), value: 'stockDesc' },
  { label: t('最近同步', 'Recently synced'), value: 'recent' },
  { label: t('销量高到低', 'Sales: high to low'), value: 'salesDesc' },
])

const searchQueries = computed(() => {
  const queries = searchTerms.value.map((term) => normalizeSearchText(term))
  const draft = normalizeSearchText(searchDraft.value)
  if (draft) {
    queries.push(draft)
  }
  return Array.from(new Set(queries))
})

const rows = computed<CardShopRow[]>(() => {
  const nextRows = shops.value
    .map((shop, index) => buildShopRow(shop, index, searchQueries.value))
    .filter((row): row is CardShopRow => row !== null)
    .filter((row) => !favoriteOnly.value || row.isFavorite)

  return sortRows(nextRows, sortKey.value)
})

const totalProductCount = computed(() =>
  shops.value.reduce((total, shop) => {
    const products = shopProducts(shop)
    return total + shopProductCount(shop, products)
  }, 0),
)
const favoriteShopCount = computed(() =>
  shops.value.reduce((total, shop, index) => total + (favoriteKeys.value.has(shopKeyForShop(shop, index)) ? 1 : 0), 0),
)
const visibleProductCount = computed(() =>
  rows.value.reduce((total, row) => total + row.visibleProducts.length, 0),
)
const fetchedAtText = computed(() =>
  fetchedAt.value ? formatDateTime(fetchedAt.value, { includeSecond: false }) : DEFAULT_EMPTY_TEXT,
)
const resultSummary = computed(() =>
  t(
    `共 ${formatInteger(rows.value.length)} / ${formatInteger(shops.value.length)} 个店铺，展示 ${formatInteger(visibleProductCount.value)} / ${formatInteger(totalProductCount.value)} 个商品`,
    `${formatInteger(rows.value.length)} / ${formatInteger(shops.value.length)} shops, ${formatInteger(visibleProductCount.value)} / ${formatInteger(totalProductCount.value)} products shown`,
  ),
)
const metricItems = computed(() => [
  {
    key: 'shops',
    label: t('收录店铺', 'Shops'),
    value: formatInteger(shops.value.length),
    footnote: t('公开店铺快照', 'Public shop snapshot'),
    icon: Store,
    tone: 'is-primary',
  },
  {
    key: 'products',
    label: t('在售商品', 'Products'),
    value: formatInteger(totalProductCount.value),
    footnote: t('按接口当前返回统计', 'Counted from the latest response'),
    icon: Package,
    tone: 'is-blue',
  },
  {
    key: 'filtered',
    label: t('筛选结果', 'Filtered'),
    value: formatInteger(rows.value.length),
    footnote: resultSummary.value,
    icon: Search,
    tone: 'is-purple',
  },
  {
    key: 'fetched',
    label: t('本次拉取', 'Fetched'),
    value: fetchedAtText.value,
    footnote: t('进入页面或点击刷新时更新', 'Updated on page load or refresh'),
    icon: Clock3,
    tone: 'is-orange',
  },
])

onMounted(() => {
  void refresh()
})

watch(
  () => currentUser.value?.id ?? null,
  (userID) => {
    if (userID === null) {
      activePreferenceUserID.value = null
      return
    }
    restoreCardShopViewPreferences(userID)
  },
  { immediate: true },
)

watch(
  () => ({
    searchDraft: searchDraft.value,
    searchTerms: [...searchTerms.value],
    sortKey: sortKey.value,
    favoriteOnly: favoriteOnly.value,
  }),
  () => saveCardShopViewPreferences(),
)

async function refresh() {
  isLoading.value = true
  loadError.value = null
  favoriteError.value = null
  try {
    const [shopsResponse, favoritesResponse, tagsResponse] = await Promise.all([
      getCardShops(),
      getCardShopFavorites(),
      getCardShopTags(),
    ])
    shops.value = shopsResponse.shops ?? []
    fetchedAt.value = shopsResponse.fetched_at
    favoriteKeys.value = new Set(favoritesResponse.shop_keys ?? [])
    if (favoriteOnly.value && favoriteKeys.value.size === 0) {
      favoriteOnly.value = false
    }
    quickTags.value = tagsOrDefault(tagsResponse.tags)
  } catch (error) {
    loadError.value = errorText(error, '加载卡网收录失败', 'Failed to load card shops')
  } finally {
    isLoading.value = false
  }
}

function restoreCardShopViewPreferences(userID: number) {
  isRestoringViewPreferences.value = true
  activePreferenceUserID.value = null
  resetCardShopViewPreferences()

  const preferences = readCardShopViewPreferences(userID)
  if (preferences) {
    if (typeof preferences.searchDraft === 'string') {
      searchDraft.value = preferences.searchDraft
    }
    if (Array.isArray(preferences.searchTerms)) {
      searchTerms.value = preferences.searchTerms
    }
    if (preferences.sortKey) {
      sortKey.value = preferences.sortKey
    }
    if (typeof preferences.favoriteOnly === 'boolean') {
      favoriteOnly.value = preferences.favoriteOnly
    }
  }

  activePreferenceUserID.value = userID
  isRestoringViewPreferences.value = false
}

function resetCardShopViewPreferences() {
  searchDraft.value = ''
  searchTerms.value = []
  sortKey.value = DEFAULT_SORT_KEY
  favoriteOnly.value = false
}

function readCardShopViewPreferences(userID: number): CachedCardShopViewPreferences | null {
  if (typeof localStorage === 'undefined') {
    return null
  }
  const raw = localStorage.getItem(cardShopViewPreferencesStorageKey(userID))
  if (!raw) {
    return null
  }

  try {
    const value: unknown = JSON.parse(raw)
    if (!value || typeof value !== 'object') {
      return null
    }
    const source = value as Record<string, unknown>
    if (source.version !== CARD_SHOP_VIEW_PREFERENCES_VERSION) {
      return null
    }

    const preferences: CachedCardShopViewPreferences = {}
    if (typeof source.searchDraft === 'string') {
      preferences.searchDraft = source.searchDraft
    }
    const cachedSearchTerms = normalizeCachedSearchTerms(source.searchTerms)
    if (cachedSearchTerms) {
      preferences.searchTerms = cachedSearchTerms
    }
    if (isCardShopSortKey(source.sortKey)) {
      preferences.sortKey = source.sortKey
    }
    if (typeof source.favoriteOnly === 'boolean') {
      preferences.favoriteOnly = source.favoriteOnly
    }
    return preferences
  } catch {
    return null
  }
}

function saveCardShopViewPreferences() {
  const userID = activePreferenceUserID.value
  if (userID === null || isRestoringViewPreferences.value || typeof localStorage === 'undefined') {
    return
  }

  const preferences: CardShopViewPreferences = {
    version: CARD_SHOP_VIEW_PREFERENCES_VERSION,
    searchDraft: searchDraft.value,
    searchTerms: searchTerms.value,
    sortKey: sortKey.value,
    favoriteOnly: favoriteOnly.value,
  }
  try {
    localStorage.setItem(cardShopViewPreferencesStorageKey(userID), JSON.stringify(preferences))
  } catch {
    // Keep the page usable when browser storage is unavailable.
  }
}

function cardShopViewPreferencesStorageKey(userID: number): string {
  return `${CARD_SHOP_VIEW_PREFERENCES_STORAGE_PREFIX}:${userID}`
}

function normalizeCachedSearchTerms(value: unknown): string[] | null {
  if (!Array.isArray(value)) {
    return null
  }

  const terms: string[] = []
  const seen = new Set<string>()
  for (const item of value) {
    if (typeof item !== 'string') {
      return null
    }
    const term = item.trim()
    if (!term) {
      continue
    }
    const key = normalizeSearchText(term)
    if (seen.has(key)) {
      continue
    }
    seen.add(key)
    terms.push(term)
  }
  return terms
}

function isCardShopSortKey(value: unknown): value is CardShopSortKey {
  return (
    value === 'relevance' ||
    value === 'priceAsc' ||
    value === 'stockDesc' ||
    value === 'recent' ||
    value === 'salesDesc'
  )
}

function setSortKey(value: unknown) {
  if (isCardShopSortKey(value)) {
    sortKey.value = value
  }
}

function defaultQuickTags(): string[] {
  return [...DEFAULT_QUICK_TAGS]
}

function tagsOrDefault(tags: string[] | null | undefined): string[] {
  return Array.isArray(tags) && tags.length > 0 ? [...tags] : defaultQuickTags()
}

function openTagManager() {
  editableTags.value = [...quickTags.value]
  newTagDraft.value = ''
  tagEditorError.value = null
  isTagModalOpen.value = true
}

function closeTagManager() {
  if (isSavingTags.value) {
    return
  }
  isTagModalOpen.value = false
}

function addEditableTag() {
  const result = normalizeQuickTagsForSave([...editableTags.value, newTagDraft.value])
  if (result.error) {
    tagEditorError.value = result.error
    return
  }
  editableTags.value = result.tags
  newTagDraft.value = ''
  tagEditorError.value = null
}

function updateEditableTag(index: number, value: string) {
  const nextTags = [...editableTags.value]
  nextTags[index] = value
  editableTags.value = nextTags
}

function removeEditableTag(index: number) {
  editableTags.value = editableTags.value.filter((_, tagIndex) => tagIndex !== index)
  tagEditorError.value = null
}

function startTagDrag(index: number, event: DragEvent) {
  draggingTagIndex.value = index
  tagEditorError.value = null
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.dropEffect = 'move'
    event.dataTransfer.setData('text/plain', String(index))
  }
}

function dragOverTag(index: number, event: DragEvent) {
  event.preventDefault()
  const fromIndex = draggingTagIndex.value
  if (fromIndex === null || fromIndex === index) {
    return
  }
  const nextTags = [...editableTags.value]
  const tag = nextTags[fromIndex]
  if (tag === undefined) {
    return
  }
  nextTags.splice(fromIndex, 1)
  nextTags.splice(index, 0, tag)
  editableTags.value = nextTags
  draggingTagIndex.value = index
}

function dropTag(event: DragEvent) {
  event.preventDefault()
  draggingTagIndex.value = null
}

function endTagDrag() {
  draggingTagIndex.value = null
}

function restoreDefaultTags() {
  editableTags.value = defaultQuickTags()
  newTagDraft.value = ''
  tagEditorError.value = null
}

async function saveQuickTags() {
  const values = newTagDraft.value.trim() ? [...editableTags.value, newTagDraft.value] : editableTags.value
  const result = normalizeQuickTagsForSave(values)
  if (result.error) {
    tagEditorError.value = result.error
    return
  }

  isSavingTags.value = true
  tagEditorError.value = null
  try {
    const tags = tagsEqual(result.tags, defaultQuickTags()) ? [] : result.tags
    const response = await updateCardShopTags({ tags })
    quickTags.value = tagsOrDefault(response.tags)
    editableTags.value = [...quickTags.value]
    newTagDraft.value = ''
    isTagModalOpen.value = false
  } catch (error) {
    tagEditorError.value = errorText(error, '保存快速搜索标签失败', 'Failed to save quick search tags')
  } finally {
    isSavingTags.value = false
  }
}

function normalizeQuickTagsForSave(values: string[]): { tags: string[]; error: string | null } {
  if (values.length > MAX_QUICK_TAGS) {
    return {
      tags: [],
      error: t(`快速搜索标签不能超过 ${MAX_QUICK_TAGS} 个`, `Quick search tags cannot exceed ${MAX_QUICK_TAGS}`),
    }
  }

  const tags: string[] = []
  const seen = new Set<string>()
  for (const value of values) {
    const tag = value.trim()
    if (!tag) {
      return { tags: [], error: t('快速搜索标签不能为空', 'Quick search tags cannot be empty') }
    }
    if ([...tag].length > MAX_QUICK_TAG_LENGTH) {
      return {
        tags: [],
        error: t(
          `单个快速搜索标签不能超过 ${MAX_QUICK_TAG_LENGTH} 个字符`,
          `A quick search tag cannot exceed ${MAX_QUICK_TAG_LENGTH} characters`,
        ),
      }
    }
    const key = normalizeSearchText(tag)
    if (seen.has(key)) {
      return { tags: [], error: t('快速搜索标签不能重复', 'Quick search tags cannot be duplicated') }
    }
    seen.add(key)
    tags.push(tag)
  }
  return { tags, error: null }
}

function tagsEqual(left: string[], right: string[]): boolean {
  return left.length === right.length && left.every((tag, index) => tag === right[index])
}

function applyQuickTag(tag: string) {
  const normalizedTag = normalizeSearchText(tag)
  const existingIndex = searchTerms.value.findIndex((term) => normalizeSearchText(term) === normalizedTag)
  if (existingIndex >= 0) {
    searchTerms.value.splice(existingIndex, 1)
    return
  }
  addSearchTerm(tag)
  if (normalizeSearchText(searchDraft.value) === normalizedTag) {
    searchDraft.value = ''
  }
}

function addSearchDraft() {
  addSearchTerm(searchDraft.value)
  searchDraft.value = ''
}

function addSearchTerm(value: string) {
  const term = value.trim()
  if (!term) {
    return
  }
  const normalizedTerm = normalizeSearchText(term)
  if (searchTerms.value.some((item) => normalizeSearchText(item) === normalizedTerm)) {
    return
  }
  searchTerms.value.push(term)
}

function removeSearchTerm(term: string) {
  const normalizedTerm = normalizeSearchText(term)
  searchTerms.value = searchTerms.value.filter((item) => normalizeSearchText(item) !== normalizedTerm)
}

function clearSearchFilters() {
  searchTerms.value = []
  searchDraft.value = ''
}

async function toggleFavorite(row: CardShopRow) {
  const nextFavorite = !row.isFavorite
  favoriteError.value = null
  setFavoriteUpdating(row.shopKey, true)
  try {
    const response = await updateCardShopFavorite({
      shop_key: row.shopKey,
      favorite: nextFavorite,
    })
    favoriteKeys.value = new Set(response.shop_keys ?? [])
    if (favoriteOnly.value && favoriteKeys.value.size === 0) {
      favoriteOnly.value = false
    }
  } catch (error) {
    favoriteError.value = errorText(error, '更新收藏失败', 'Failed to update favorite')
  } finally {
    setFavoriteUpdating(row.shopKey, false)
  }
}

function setFavoriteUpdating(shopKey: string, updating: boolean) {
  const nextKeys = new Set(favoriteUpdatingKeys.value)
  if (updating) {
    nextKeys.add(shopKey)
  } else {
    nextKeys.delete(shopKey)
  }
  favoriteUpdatingKeys.value = nextKeys
}

function isFavoriteUpdating(shopKey: string): boolean {
  return favoriteUpdatingKeys.value.has(shopKey)
}

function isSearchTermActive(value: string): boolean {
  const normalizedValue = normalizeSearchText(value)
  return searchQueries.value.includes(normalizedValue)
}

function normalizeSearchText(value: string): string {
  return value.trim().toLowerCase()
}

function textValue(value: string | null | undefined): string {
  return value?.trim() || ''
}

function displayText(value: string | null | undefined): string {
  return textValue(value) || DEFAULT_EMPTY_TEXT
}

function numberValue(value: number | null | undefined): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

function shopProducts(shop: CardShop): CardShopProductItem[] {
  const productItems = Array.isArray(shop.productItems) ? shop.productItems : []
  if (productItems.length > 0) {
    return productItems
  }
  const names = Array.isArray(shop.productsInStock) ? shop.productsInStock : []
  return names.map((name) => ({ name }))
}

function shopProductCount(shop: CardShop, products: CardShopProductItem[]): number {
  const summaryCount = numberValue(shop.productSummary?.totalCount)
  if (summaryCount === null || !Number.isInteger(summaryCount) || summaryCount < products.length) {
    return products.length
  }
  return summaryCount
}

function searchableProductTitleText(product: CardShopProductItem): string {
  return textValue(product.name).toLowerCase()
}

function shopKeyForShop(shop: CardShop, index: number): string {
  return textValue(shop.id) || textValue(shop.shopUrl) || textValue(shop.shopName) || `shop-${index}`
}

function buildShopRow(shop: CardShop, index: number, queries: string[]): CardShopRow | null {
  const products = shopProducts(shop)
  const productTitleTexts = products.map((product) => searchableProductTitleText(product))
  const matchedProducts = queries.length > 0
    ? products.filter((_, productIndex) =>
        queries.every((query) => productTitleTexts[productIndex]?.includes(query) ?? false),
      )
    : products
  if (queries.length > 0 && matchedProducts.length === 0) {
    return null
  }

  const visibleProducts = queries.length > 0 ? matchedProducts : products
  const shopKey = shopKeyForShop(shop, index)
  return {
    shop,
    shopKey,
    isFavorite: favoriteKeys.value.has(shopKey),
    visibleProducts,
    productCount: shopProductCount(shop, products),
    matchedProductCount: visibleProducts.length,
    minPrice: minProductPrice(visibleProducts),
    totalStock: visibleProducts.reduce((total, product) => total + (numberValue(product.stockCount) ?? 0), 0),
    sales: numberValue(shop.shopSellCount) ?? 0,
    latestMs: updatedAtMs(shop),
    score: queries.length > 0 ? relevanceScore(visibleProducts, queries) : 0,
  }
}

function relevanceScore(products: CardShopProductItem[], queries: string[]): number {
  return queries.reduce((total, query) => total + relevanceScoreForQuery(products, query), 0)
}

function relevanceScoreForQuery(products: CardShopProductItem[], query: string): number {
  let score = 0
  products.forEach((product) => {
    const name = searchableProductTitleText(product)
    if (name === query) {
      score += 36
    } else if (name.includes(query)) {
      score += 18
    }
  })
  return score
}

function minProductPrice(products: CardShopProductItem[]): number | null {
  const prices = products
    .map((product) => numberValue(product.price))
    .filter((price): price is number => price !== null)
  return prices.length > 0 ? Math.min(...prices) : null
}

function updatedAtMs(shop: CardShop): number {
  const updatedAt = textValue(shop.updatedAt)
  if (!updatedAt) {
    return 0
  }
  const parsed = Date.parse(updatedAt)
  return Number.isFinite(parsed) ? parsed : 0
}

function sortRows(sourceRows: CardShopRow[], key: CardShopSortKey): CardShopRow[] {
  const nextRows = [...sourceRows]
  nextRows.sort((left, right) => {
    const favoriteComparison = compareFavoriteRows(left, right)
    if (favoriteComparison !== 0) {
      return favoriteComparison
    }

    switch (key) {
      case 'priceAsc':
        return compareNullableNumberAsc(left.minPrice, right.minPrice) || compareShopNames(left, right)
      case 'stockDesc':
        return right.totalStock - left.totalStock || compareShopNames(left, right)
      case 'recent':
        return right.latestMs - left.latestMs || compareShopNames(left, right)
      case 'salesDesc':
        return right.sales - left.sales || compareShopNames(left, right)
      case 'relevance':
        return (
          right.score - left.score ||
          right.latestMs - left.latestMs ||
          right.sales - left.sales ||
          right.totalStock - left.totalStock ||
          compareShopNames(left, right)
        )
    }
  })
  return nextRows
}

function compareFavoriteRows(left: CardShopRow, right: CardShopRow): number {
  if (left.isFavorite === right.isFavorite) {
    return 0
  }
  return left.isFavorite ? -1 : 1
}

function compareNullableNumberAsc(left: number | null, right: number | null): number {
  if (left === null && right === null) {
    return 0
  }
  if (left === null) {
    return 1
  }
  if (right === null) {
    return -1
  }
  return left - right
}

function compareShopNames(left: CardShopRow, right: CardShopRow): number {
  return displayText(left.shop.shopName).localeCompare(displayText(right.shop.shopName), currentLanguage.value)
}

function formatShopPrice(value: number | null | undefined): string {
  const price = numberValue(value)
  if (price === null) {
    return DEFAULT_EMPTY_TEXT
  }
  return new Intl.NumberFormat(currentLanguage.value === 'zh' ? 'zh-CN' : 'en-US', {
    style: 'currency',
    currency: 'CNY',
    maximumFractionDigits: price < 10 ? 2 : 1,
  }).format(price)
}

function formatCount(value: number | null | undefined): string {
  const count = numberValue(value)
  return count === null ? DEFAULT_EMPTY_TEXT : formatInteger(count)
}

function telegramHref(value: string | null | undefined): string | null {
  const telegram = textValue(value)
  if (!telegram) {
    return null
  }
  if (telegram.startsWith('http://') || telegram.startsWith('https://')) {
    return telegram
  }
  if (telegram.startsWith('@') && telegram.length > 1) {
    return `https://t.me/${encodeURIComponent(telegram.slice(1))}`
  }
  return null
}
</script>

<template>
  <section class="page card-shops-page">
    <div class="page-toolbar card-shops-header">
      <h1 data-page-title class="page-title">{{ t('卡网收录', 'Card shops') }}</h1>
      <Button :disabled="isLoading" @click="refresh">
        <Spinner v-if="isLoading" data-icon="inline-start" />
        <RefreshCw v-else data-icon="inline-start" />
        {{ t('刷新', 'Refresh') }}
      </Button>
    </div>

    <Alert class="risk-alert">
      <AlertTriangle />
      <AlertTitle>{{ t('风险提示', 'Risk notice') }}</AlertTitle>
      <AlertDescription>
        {{ t('仅做公开店铺信息收录，不参与交易，也不对店铺商品、售后和风险负责。使用前请自行甄别。', 'This only indexes public shop information. CPA-Helper does not participate in transactions and is not responsible for products, after-sales service, or risk. Assess independently before use.') }}
      </AlertDescription>
    </Alert>

    <Card>
      <CardHeader>
        <CardTitle>{{ t('筛选店铺', 'Filter shops') }}</CardTitle>
        <CardDescription>{{ t('可组合商品关键字、排序与收藏条件。', 'Combine product keywords, sorting, and favorites.') }}</CardDescription>
      </CardHeader>
      <CardContent class="card-shop-toolbar">
        <div class="search-row">
          <InputGroup>
            <InputGroupAddon><Search /></InputGroupAddon>
            <InputGroupInput
              v-model="searchDraft"
              :placeholder="t('商品标题名', 'Product title')"
              @keydown.enter.prevent="addSearchDraft"
            />
          </InputGroup>
          <div class="sort-control">
            <span>{{ t('排序', 'Sort') }}</span>
            <Select :model-value="sortKey" @update:model-value="setSortKey">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem v-for="option in sortOptions" :key="option.value" :value="option.value">
                    {{ option.label }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
          <div class="favorite-filter">
            <span>{{ t('收藏筛选', 'Favorites') }}</span>
            <label class="favorite-switch-line">
              <Switch v-model="favoriteOnly" />
              <span class="truncate">
                {{ t(`仅看收藏 ${formatInteger(favoriteShopCount)}`, `Favorites only ${formatInteger(favoriteShopCount)}`) }}
              </span>
            </label>
          </div>
        </div>
        <div class="tag-row">
          <span>{{ t('快速搜索标签:', 'Quick search tags:') }}</span>
          <Button
            v-for="tag in quickTags"
            :key="tag"
            size="sm"
            :variant="isSearchTermActive(tag) ? 'default' : 'outline'"
            @click="applyQuickTag(tag)"
          >
            {{ tag }}
          </Button>
          <Button class="tag-manage-button" size="sm" variant="secondary" @click="openTagManager">
            <Settings2 data-icon="inline-start" />
            {{ t('管理标签', 'Manage tags') }}
          </Button>
        </div>
        <div v-if="searchTerms.length > 0" class="selected-term-row">
          <span>{{ t('已选条件:', 'Selected filters:') }}</span>
          <Badge v-for="term in searchTerms" :key="term" variant="secondary" class="selected-term-badge">
            <span class="truncate">{{ term }}</span>
            <button
              type="button"
              :aria-label="t(`移除 ${term}`, `Remove ${term}`)"
              @click="removeSearchTerm(term)"
            >
              <X />
            </button>
          </Badge>
          <Button size="sm" variant="ghost" @click="clearSearchFilters">
            {{ t('清空', 'Clear') }}
          </Button>
        </div>
      </CardContent>
    </Card>

    <Alert v-if="favoriteError" class="favorite-error" variant="destructive">
      <AlertDescription>{{ favoriteError }}</AlertDescription>
    </Alert>

    <div class="card-shop-metrics">
      <Card v-for="metric in metricItems" :key="metric.key" size="sm" class="card-shop-metric" :class="metric.tone">
        <CardHeader class="flex flex-row items-start justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <CardDescription>{{ metric.label }}</CardDescription>
            <CardTitle class="text-xl tabular-nums">{{ metric.value }}</CardTitle>
          </div>
          <div class="card-shop-metric__icon">
            <component :is="metric.icon" class="size-5" />
          </div>
        </CardHeader>
        <CardContent class="truncate text-xs text-muted-foreground" :title="metric.footnote">
          {{ metric.footnote }}
        </CardContent>
      </Card>
    </div>

    <Alert v-if="loadError" variant="destructive">
      <AlertDescription class="error-state">
        <strong>{{ loadError }}</strong>
        <Button variant="outline" :disabled="isLoading" @click="refresh">
          <Spinner v-if="isLoading" data-icon="inline-start" />
          <RefreshCw v-else data-icon="inline-start" />
          {{ t('重试', 'Retry') }}
        </Button>
      </AlertDescription>
    </Alert>

    <section v-if="isLoading && shops.length === 0" class="shop-list" aria-busy="true">
      <Card v-for="index in 3" :key="index" class="shop-card">
        <div class="flex items-start justify-between gap-4">
          <div class="grid flex-1 gap-2">
            <Skeleton class="h-5 w-40" />
            <Skeleton class="h-4 w-64 max-w-full" />
          </div>
          <Skeleton class="h-8 w-20" />
        </div>
        <Skeleton class="h-20 w-full" />
      </Card>
    </section>

    <section v-else-if="!loadError && rows.length > 0" class="shop-list">
      <Card v-for="row in rows" :key="row.shopKey" class="shop-card" :class="{ 'is-favorite': row.isFavorite }">
        <div class="shop-card-head">
          <div class="shop-title-block">
            <div class="shop-title-line">
              <h2>{{ displayText(row.shop.shopName) }}</h2>
              <a
                v-if="textValue(row.shop.shopUrl)"
                class="external-link"
                :href="textValue(row.shop.shopUrl)"
                target="_blank"
                rel="noreferrer"
                :aria-label="t('打开店铺链接', 'Open shop link')"
              >
                <ExternalLink class="size-4" />
              </a>
            </div>
            <a
              v-if="textValue(row.shop.shopUrl)"
              class="shop-url"
              :href="textValue(row.shop.shopUrl)"
              target="_blank"
              rel="noreferrer"
            >
              {{ textValue(row.shop.shopUrl) }}
            </a>
          </div>
          <div class="shop-actions">
            <Button
              size="sm"
              :variant="row.isFavorite ? 'secondary' : 'outline'"
              :disabled="isFavoriteUpdating(row.shopKey)"
              @click="toggleFavorite(row)"
            >
              <Spinner v-if="isFavoriteUpdating(row.shopKey)" data-icon="inline-start" />
              <Star v-else data-icon="inline-start" :class="{ 'fill-current': row.isFavorite }" />
              {{ row.isFavorite ? t('已收藏', 'Favorited') : t('收藏', 'Favorite') }}
            </Button>
            <div class="shop-tags">
              <Badge variant="secondary">
                {{ t(`销量 ${formatCount(row.shop.shopSellCount)}`, `Sales ${formatCount(row.shop.shopSellCount)}`) }}
              </Badge>
              <Badge variant="outline">
                {{ t(`商品 ${formatInteger(row.productCount)}`, `${formatInteger(row.productCount)} products`) }}
              </Badge>
              <Badge variant="outline">
                {{ t(`库存 ${formatInteger(row.totalStock)}`, `Stock ${formatInteger(row.totalStock)}`) }}
              </Badge>
            </div>
          </div>
        </div>

        <div class="shop-meta-row">
          <span>
            <MessageCircle class="size-4" />
            <a
              v-if="telegramHref(row.shop.telegram)"
              :href="telegramHref(row.shop.telegram) ?? undefined"
              target="_blank"
              rel="noreferrer"
            >
              {{ displayText(row.shop.telegram) }}
            </a>
            <template v-else>{{ displayText(row.shop.telegram) }}</template>
          </span>
          <span>
            <Clock3 class="size-4" />
            {{ formatDateTime(row.shop.updatedAt ?? null, { includeSecond: false }) }}
          </span>
          <span>
            <ShoppingBag class="size-4" />
            {{ t(`展示 ${formatInteger(row.matchedProductCount)} 个商品`, `${formatInteger(row.matchedProductCount)} products shown`) }}
          </span>
        </div>

        <p v-if="textValue(row.shop.notes)" class="shop-notes">{{ textValue(row.shop.notes) }}</p>

        <div v-if="row.visibleProducts.length > 0" class="product-grid">
          <div
            v-for="(product, index) in row.visibleProducts"
            :key="`${row.shopKey}-${textValue(product.itemUrl) || textValue(product.name) || index}`"
            class="product-item"
          >
            <a
              v-if="textValue(product.itemUrl)"
              class="product-name"
              :href="textValue(product.itemUrl)"
              target="_blank"
              rel="noreferrer"
            >
              {{ displayText(product.name) }}
            </a>
            <strong v-else class="product-name">{{ displayText(product.name) }}</strong>
            <div class="product-meta">
              <span>{{ formatShopPrice(product.price) }}</span>
              <span>{{ t(`库存 ${formatCount(product.stockCount)}`, `Stock ${formatCount(product.stockCount)}`) }}</span>
              <span>{{ displayText(product.group) }}</span>
              <span>{{ displayText(product.category) }}</span>
            </div>
          </div>
        </div>
        <div v-else class="product-empty">
          {{ t('暂无匹配商品明细', 'No matching product details') }}
        </div>
      </Card>
    </section>

    <Card v-else-if="!loadError && !isLoading" class="empty-panel">
      <CardContent class="empty-state">
        {{ t('当前筛选下暂无店铺', 'No shops match the current filters') }}
      </CardContent>
    </Card>

    <Dialog v-model:open="isTagModalOpen">
      <DialogContent class="max-h-[calc(100svh-2rem)] overflow-hidden sm:max-w-[620px]">
        <DialogHeader>
          <DialogTitle>{{ t('管理快速搜索标签', 'Manage quick search tags') }}</DialogTitle>
          <DialogDescription>
            {{ t(`最多 ${MAX_QUICK_TAGS} 个标签，可拖动调整顺序。`, `Up to ${MAX_QUICK_TAGS} tags; drag to reorder.`) }}
          </DialogDescription>
        </DialogHeader>
        <div class="quick-tag-editor">
          <div class="quick-tag-add-row">
            <div class="grid gap-1">
              <Input
                v-model="newTagDraft"
                :maxlength="MAX_QUICK_TAG_LENGTH"
                :placeholder="t('新增标签', 'New tag')"
                @keydown.enter.prevent="addEditableTag"
              />
              <span class="text-right text-xs text-muted-foreground">{{ [...newTagDraft].length }} / {{ MAX_QUICK_TAG_LENGTH }}</span>
            </div>
            <Button variant="outline" :disabled="editableTags.length >= MAX_QUICK_TAGS" @click="addEditableTag">
              <PlusIcon data-icon="inline-start" />
              {{ t('新增', 'Add') }}
            </Button>
          </div>

          <Alert v-if="tagEditorError" variant="destructive">
            <AlertDescription>{{ tagEditorError }}</AlertDescription>
          </Alert>

          <div v-if="editableTags.length > 0" class="quick-tag-list">
            <div
              v-for="(tag, index) in editableTags"
              :key="`${tag}-${index}`"
              class="quick-tag-editor-item"
              :class="{ 'is-dragging': draggingTagIndex === index }"
              @dragover.prevent="dragOverTag(index, $event)"
              @drop.prevent="dropTag"
            >
              <button
                type="button"
                class="quick-tag-drag-handle"
                draggable="true"
                :aria-label="t('拖动排序', 'Drag to reorder')"
                @dragstart="startTagDrag(index, $event)"
                @dragend="endTagDrag"
              >
                <GripVertical class="size-4" />
              </button>
              <Input
                :model-value="tag"
                :maxlength="MAX_QUICK_TAG_LENGTH"
                @update:model-value="updateEditableTag(index, String($event))"
              />
              <Button size="icon-sm" variant="ghost" :aria-label="t('删除', 'Delete')" @click="removeEditableTag(index)">
                <Trash2 />
              </Button>
            </div>
          </div>
          <div v-else class="quick-tag-empty">
            {{ t('保存后将显示默认快速搜索标签', 'Default quick search tags will be shown after saving') }}
          </div>

          <DialogFooter class="quick-tag-footer sm:justify-between">
            <Button variant="outline" @click="restoreDefaultTags">
              <RotateCcw data-icon="inline-start" />
              {{ t('恢复默认', 'Restore default') }}
            </Button>
            <div class="quick-tag-footer-actions">
              <Button variant="ghost" :disabled="isSavingTags" @click="closeTagManager">
                {{ t('取消', 'Cancel') }}
              </Button>
              <Button :disabled="isSavingTags" @click="saveQuickTags">
                <Spinner v-if="isSavingTags" data-icon="inline-start" />
                {{ t('保存', 'Save') }}
              </Button>
            </div>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  </section>
</template>

<style scoped>
.card-shops-page {
  gap: 12px;
}

.risk-alert {
  border-radius: var(--radius-lg);
}

.card-shop-toolbar {
  display: grid;
  gap: 12px;
  padding-block: 14px;
}

.search-row {
  display: grid;
  grid-template-columns: minmax(240px, 1fr) minmax(180px, 280px) minmax(150px, 190px);
  gap: 12px;
  align-items: end;
  min-width: 0;
}

.sort-control,
.favorite-filter {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.sort-control span,
.favorite-filter > span,
.tag-row > span,
.selected-term-row > span {
  color: var(--cpa-text-muted);
  font-size: 12px;
  font-weight: 700;
}

.favorite-switch-line {
  display: flex;
  align-items: center;
  min-height: 34px;
  min-width: 0;
  gap: 8px;
  color: var(--cpa-text-strong);
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
}

.favorite-switch-line span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tag-row,
.selected-term-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.tag-row > button {
  min-width: 58px;
}

.tag-manage-button {
  margin-left: 2px;
  color: var(--primary);
  font-weight: 600;
}

.selected-term-row {
  padding-top: 2px;
}

.selected-term-badge {
  display: inline-flex;
  max-width: min(260px, 100%);
  gap: .25rem;
}

.selected-term-badge button {
  display: inline-flex;
  width: 1rem;
  height: 1rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
}

.quick-tag-editor {
  display: grid;
  gap: 12px;
}

.quick-tag-add-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  align-items: start;
}

.quick-tag-list {
  display: grid;
  max-height: min(420px, 52vh);
  gap: 8px;
  overflow: auto;
  padding-right: 2px;
}

.quick-tag-editor-item {
  display: grid;
  grid-template-columns: 32px minmax(0, 1fr) 32px;
  gap: 6px;
  align-items: center;
  min-width: 0;
  border-radius: var(--cpa-radius-sm);
  transition:
    background 0.15s ease,
    opacity 0.15s ease;
}

.quick-tag-editor-item > button:last-child {
  width: 32px;
  min-width: 32px;
  padding-inline: 0;
}

.quick-tag-editor-item.is-dragging {
  background: color-mix(in srgb, var(--cpa-primary) 8%, transparent);
  opacity: 0.72;
}

.quick-tag-drag-handle {
  display: inline-grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border: 0;
  border-radius: var(--cpa-radius-sm);
  color: var(--cpa-text-muted);
  background: transparent;
  cursor: grab;
}

.quick-tag-drag-handle:hover,
.quick-tag-drag-handle:focus-visible {
  color: var(--cpa-primary);
  background: var(--cpa-primary-wash);
  outline: none;
}

.quick-tag-drag-handle:active {
  cursor: grabbing;
}

.quick-tag-empty {
  display: grid;
  min-height: 64px;
  place-items: center;
  border: 1px dashed var(--cpa-border);
  border-radius: var(--cpa-radius-sm);
  color: var(--cpa-text-muted);
  background: var(--cpa-surface-muted);
  font-size: 13px;
  text-align: center;
}

.quick-tag-footer {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  align-items: center;
}

.quick-tag-footer-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.card-shop-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.card-shop-metric {
  min-width: 0;
  min-height: 96px;
}

.card-shop-metric__icon {
  display: flex;
  width: 2.25rem;
  height: 2.25rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-lg);
  background: color-mix(in srgb, var(--primary) 10%, transparent);
  color: var(--primary);
}

.card-shop-metric.is-blue .card-shop-metric__icon {
  background: color-mix(in srgb, var(--chart-2) 12%, transparent);
  color: var(--chart-2);
}

.card-shop-metric.is-purple .card-shop-metric__icon {
  background: color-mix(in srgb, var(--chart-4) 12%, transparent);
  color: var(--chart-4);
}

.card-shop-metric.is-orange .card-shop-metric__icon {
  background: color-mix(in srgb, var(--cpa-warning) 12%, transparent);
  color: var(--cpa-warning);
}

.error-state {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.error-state strong {
  color: var(--cpa-danger);
  font-size: 13px;
  font-weight: 700;
}

.shop-list {
  display: grid;
  gap: 10px;
  min-width: 0;
}

.shop-card {
  display: grid;
  gap: 10px;
  padding: 14px;
}

.shop-card.is-favorite {
  --favorite-ink: var(--cpa-text-strong);
  --favorite-text: var(--cpa-text-muted);
  --favorite-muted: var(--cpa-text-muted);
  --favorite-accent: var(--cpa-text-muted);
  --favorite-border: color-mix(in srgb, #d97706 28%, var(--cpa-border));
  --favorite-bg: color-mix(in srgb, #fffbeb 68%, var(--cpa-surface));
  --favorite-panel: color-mix(in srgb, #fffbeb 78%, var(--cpa-surface-muted));
  --favorite-product-bg: color-mix(in srgb, #fef3c7 46%, var(--cpa-surface-raised));
  --favorite-link-bg: color-mix(in srgb, #fffbeb 62%, transparent);
  --favorite-link-bg-hover: color-mix(in srgb, #fef3c7 56%, transparent);
  --favorite-chip-bg: color-mix(in srgb, #fffbeb 74%, var(--cpa-surface-muted));
  --favorite-chip-strong-bg: color-mix(in srgb, #fef3c7 62%, var(--cpa-surface-muted));
  border-color: var(--favorite-border);
  background: var(--favorite-bg);
}

:root.dark .shop-card.is-favorite {
  --favorite-ink: #f7f2e8;
  --favorite-text: #cdbf9f;
  --favorite-muted: #bfae8e;
  --favorite-accent: #f4c56a;
  --favorite-border: rgb(201 145 54 / 45%);
  --favorite-bg: linear-gradient(180deg, rgb(126 87 28 / 28%), rgb(56 43 25 / 52%)),
    var(--cpa-surface);
  --favorite-panel: rgb(48 42 30 / 78%);
  --favorite-product-bg: rgb(58 48 30 / 82%);
  --favorite-link-bg: rgb(245 190 90 / 10%);
  --favorite-link-bg-hover: rgb(245 190 90 / 16%);
  --favorite-chip-bg: rgb(44 39 31 / 86%);
  --favorite-chip-strong-bg: rgb(70 53 27 / 88%);
  border-color: var(--favorite-border);
  background: var(--favorite-bg);
}

.shop-card-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  align-items: start;
  min-width: 0;
}

.shop-title-block {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.shop-title-line {
  display: flex;
  gap: 7px;
  align-items: center;
  min-width: 0;
}

.shop-title-line h2 {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--cpa-text-strong);
  font-size: 17px;
  font-weight: 760;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.external-link {
  display: inline-grid;
  flex: 0 0 auto;
  width: 24px;
  height: 24px;
  place-items: center;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius-sm);
  color: var(--cpa-primary);
  text-decoration: none;
}

.external-link:hover {
  border-color: color-mix(in srgb, var(--cpa-primary) 28%, var(--cpa-border));
  background: var(--cpa-primary-wash);
}

.shop-card.is-favorite .external-link {
  border-color: color-mix(in srgb, #d97706 24%, var(--cpa-border));
  color: var(--favorite-accent);
  background: var(--favorite-link-bg);
}

.shop-card.is-favorite .external-link:hover {
  border-color: color-mix(in srgb, #d97706 38%, var(--cpa-border));
  background: var(--favorite-link-bg-hover);
}

.shop-url {
  min-width: 0;
  overflow: hidden;
  color: var(--cpa-primary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-decoration: none;
}

.shop-url:hover,
.shop-meta-row a:hover,
.product-name:hover {
  text-decoration: underline;
}

.shop-actions {
  display: grid;
  justify-items: end;
  gap: 8px;
}

.shop-tags {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
}

.shop-meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  min-width: 0;
  color: var(--cpa-text-muted);
  font-size: 12px;
}

.shop-meta-row span {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  gap: 5px;
}

.shop-meta-row a {
  color: var(--cpa-primary);
  text-decoration: none;
}

.shop-card.is-favorite .shop-title-line h2 {
  color: var(--favorite-ink);
}

.shop-card.is-favorite .shop-meta-row {
  color: var(--favorite-muted);
}

.shop-card.is-favorite .shop-url,
.shop-card.is-favorite .shop-meta-row a {
  color: var(--favorite-accent);
}

.shop-notes {
  margin: 0;
  padding: 8px 10px;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius-sm);
  color: var(--cpa-text-muted);
  background: var(--cpa-surface-muted);
  font-size: 12px;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.shop-card.is-favorite .shop-notes {
  border-color: color-mix(in srgb, #d97706 18%, var(--cpa-border));
  color: var(--favorite-muted);
  background: var(--favorite-panel);
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 8px;
  min-width: 0;
}

.product-empty {
  min-height: 52px;
  padding: 14px 12px;
  border: 1px dashed var(--cpa-border);
  border-radius: var(--cpa-radius-sm);
  color: var(--cpa-text-muted);
  background: var(--cpa-surface-muted);
  font-size: 13px;
  text-align: center;
}

.product-item {
  display: grid;
  gap: 6px;
  min-width: 0;
  min-height: 72px;
  padding: 9px 10px;
  border: 1px solid color-mix(in srgb, #15803d 24%, var(--cpa-border));
  border-radius: var(--cpa-radius-sm);
  background: color-mix(in srgb, #22c55e 9%, var(--cpa-surface-raised));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.24);
}

.shop-card.is-favorite .product-item {
  border-color: color-mix(in srgb, #d97706 24%, var(--cpa-border));
  background: var(--favorite-product-bg);
}

.product-name {
  display: -webkit-box;
  min-width: 0;
  overflow: hidden;
  color: var(--cpa-text-strong);
  font-size: 13px;
  font-weight: 720;
  line-height: 1.38;
  overflow-wrap: anywhere;
  text-decoration: none;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.shop-card.is-favorite .product-name {
  color: var(--favorite-ink);
}

.product-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  min-width: 0;
}

.product-meta span {
  display: inline-flex;
  max-width: 100%;
  min-width: 0;
  align-items: center;
  height: 22px;
  padding: 0 7px;
  overflow: hidden;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius-sm);
  color: var(--cpa-text-muted);
  background: var(--cpa-surface-muted);
  font-size: 11px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.product-meta span:first-child {
  color: var(--cpa-primary);
  background: var(--cpa-primary-wash);
  border-color: color-mix(in srgb, var(--cpa-primary) 20%, var(--cpa-border));
  font-variant-numeric: tabular-nums;
}

.shop-card.is-favorite .product-meta span {
  border-color: color-mix(in srgb, #d97706 16%, var(--cpa-border));
  color: var(--favorite-muted);
  background: var(--favorite-chip-bg);
}

.shop-card.is-favorite .product-meta span:first-child {
  border-color: color-mix(in srgb, #d97706 28%, var(--cpa-border));
  color: var(--favorite-accent);
  background: var(--favorite-chip-strong-bg);
}

.empty-state {
  display: grid;
  min-height: 120px;
  place-items: center;
  color: var(--cpa-text-muted);
  font-size: 13px;
}

@media (max-width: 1180px) {
  .card-shop-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .search-row,
  .shop-card-head {
    grid-template-columns: 1fr;
  }

  .card-shop-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .shop-tags {
    justify-content: flex-start;
  }

  .shop-actions {
    justify-items: start;
  }

  .product-grid {
    grid-template-columns: 1fr;
  }

  .error-state {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 460px) {
  .card-shop-metrics {
    grid-template-columns: 1fr;
  }

  .quick-tag-add-row,
  .quick-tag-footer {
    grid-template-columns: 1fr;
  }

  .quick-tag-add-row,
  .quick-tag-footer,
  .quick-tag-footer-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .quick-tag-footer-actions button,
  .quick-tag-footer > button {
    width: 100%;
  }
}
</style>
