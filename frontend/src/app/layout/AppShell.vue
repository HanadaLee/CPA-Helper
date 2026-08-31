<script setup lang="ts">
import type { Component } from 'vue'
import { useMediaQuery } from '@vueuse/core'
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { isNavigationFailure, NavigationFailureType, useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarRail,
  SidebarTrigger,
} from '@/components/ui/sidebar'
import {
  BarChart3,
  Cpu,
  DollarSign,
  EllipsisVertical,
  KeyRound,
  Languages,
  List,
  ListChecks,
  LogOut,
  Monitor,
  Moon,
  Network,
  Settings,
  Sun,
  UserRound,
  Users,
} from '@lucide/vue'

import { getMe, isAuthUser, logout } from '@/features/auth/api/authApi'
import { useCurrentUser } from '@/features/auth/state/currentUser'
import { getBranding } from '@/features/settings/api/settingsApi'
import { useThemePreference } from '@/shared/composables/useThemePreference'
import { useI18n } from '@/shared/i18n'
import type { BrandingResponse } from '@/shared/types/api'
import { logoUrl } from '@/shared/utils/assets'

const route = useRoute()
const router = useRouter()
const navigationTarget = ref<string | null>(null)
const isRouteTransitioning = ref(false)
const { currentUser, setCurrentUser } = useCurrentUser()
const hasLoadedUser = ref(currentUser.value !== null)
const { isDark, preference, setThemePreference } = useThemePreference()
const { language, t, toggleLanguage } = useI18n()
const isMobileViewport = useMediaQuery('(max-width: 768px)')
const branding = ref<BrandingResponse>({
  brand_name_zh: 'CPA-Helper',
  brand_name_en: 'CPA-Helper',
  brand_subtitle_zh: '边缘网关管理平台',
  brand_subtitle_en: 'Edge Gateway Management Platform',
})
let navigationFeedbackTimer: number | undefined
let routeTransitionReleaseTimer: number | undefined

onBeforeUnmount(() => {
  if (navigationFeedbackTimer !== undefined) {
    window.clearTimeout(navigationFeedbackTimer)
  }
  if (routeTransitionReleaseTimer !== undefined) {
    window.clearTimeout(routeTransitionReleaseTimer)
  }
})

async function refreshCurrentUser() {
  try {
    setCurrentUser(await getMe())
  } catch {
    setCurrentUser(null)
  } finally {
    hasLoadedUser.value = true
  }
}

function applyBranding(value: Partial<BrandingResponse> | null | undefined) {
  if (!value) return
  for (const key of ['brand_name_zh', 'brand_name_en', 'brand_subtitle_zh', 'brand_subtitle_en'] as const) {
    const nextValue = value[key]
    if (typeof nextValue === 'string' && nextValue.trim()) {
      branding.value[key] = nextValue.trim()
    }
  }
}

async function refreshBranding() {
  try {
    applyBranding(await getBranding())
  } catch {
    // Keep the built-in branding when the public setting cannot be loaded.
  }
}

onMounted(() => {
  void refreshCurrentUser()
  void refreshBranding()
})

function handleAccountUpdated(event: Event) {
  const nextUser = (event as CustomEvent<unknown>).detail
  if (isAuthUser(nextUser)) {
    setCurrentUser(nextUser)
    hasLoadedUser.value = true
    return
  }
  void refreshCurrentUser()
}

function handleBrandingUpdated(event: Event) {
  applyBranding((event as CustomEvent<Partial<BrandingResponse>>).detail)
}

window.addEventListener('cpa:account-updated', handleAccountUpdated)
window.addEventListener('cpa:branding-updated', handleBrandingUpdated)
onBeforeUnmount(() => {
  window.removeEventListener('cpa:account-updated', handleAccountUpdated)
  window.removeEventListener('cpa:branding-updated', handleBrandingUpdated)
})

interface NavigationItem {
  label: string
  key: string
  icon: Component
}

const adminMenuItems = computed<NavigationItem[]>(() => [
  { label: t('用量分析', 'Usage Analytics'), key: '/admin/usage', icon: BarChart3 },
  { label: t('请求明细', 'Request Records'), key: '/admin/records', icon: List },
  { label: t('用户管理', 'Users'), key: '/admin/users', icon: Users },
  { label: t('模型价格', 'Model Prices'), key: '/admin/pricing', icon: DollarSign },
  { label: t('上游管理', 'Upstreams'), key: '/admin/upstreams', icon: Network },
  { label: t('账号管理', 'Account Management'), key: '/admin/account-mgmt', icon: ListChecks },
  { label: 'CPAMC', key: '/admin/cpamc', icon: Monitor },
  { label: t('系统设置', 'System Settings'), key: '/admin/settings', icon: Settings },
])

const showAccountStatusForUser = computed(
  () => currentUser.value?.is_admin === false && currentUser.value.can_view_account_status,
)
const showUsageHistoryForUser = computed(
  () => currentUser.value?.is_admin === false && currentUser.value.can_view_usage_history,
)

const accountMenuItems = computed<NavigationItem[]>(() => [
  ...(showUsageHistoryForUser.value
    ? [{ label: t('用量分析', 'Usage Analytics'), key: '/account/history', icon: BarChart3 }]
    : []),
  { label: t('我的用量', 'My Usage'), key: '/account/usage', icon: BarChart3 },
  { label: t('我的明细', 'My Records'), key: '/account/records', icon: List },
  { label: t('API 密钥', 'API Keys'), key: '/account/keys', icon: KeyRound },
  { label: t('可用模型', 'Available Models'), key: '/account/models', icon: Cpu },
  ...(showAccountStatusForUser.value
    ? [{ label: t('账号状态', 'Account Status'), key: '/account/status', icon: ListChecks }]
    : []),
])

const isAdmin = computed(() => {
  if (currentUser.value) {
    return currentUser.value.is_admin
  }
  if (!hasLoadedUser.value) {
    return route.path.startsWith('/admin')
  }
  return false
})
const roleText = computed(() => (isAdmin.value ? t('管理员', 'Admin') : t('普通用户', 'User')))
const accountText = computed(() => currentUser.value?.username || t('当前账号', 'Current account'))
const brandName = computed(() => language.value === 'zh' ? branding.value.brand_name_zh : branding.value.brand_name_en)
const brandSubtitle = computed(() => language.value === 'zh' ? branding.value.brand_subtitle_zh : branding.value.brand_subtitle_en)

function formatAppVersion(value: string | undefined): string {
  const version = value?.trim()
  if (!version) {
    return 'dev'
  }
  if (version === 'latest' || version === 'dev') {
    return version
  }
  return version.startsWith('v') ? version : `v${version}`
}

const appVersion = formatAppVersion(import.meta.env.VITE_APP_VERSION)

const leafMenuOptions = computed(() =>
  isAdmin.value
    ? [...adminMenuItems.value, ...accountMenuItems.value]
    : accountMenuItems.value,
)

const selectedKey = computed(() => {
  const matched = leafMenuOptions.value.find((item) => route.path.startsWith(String(item.key)))
  return matched ? String(matched.key) : isAdmin.value ? '/admin/usage' : '/account/usage'
})
const currentNavigationLabel = computed(
  () => leafMenuOptions.value.find((item) => item.key === selectedKey.value)?.label ?? brandName.value,
)
const isMenuNavigationPending = computed(() => navigationTarget.value !== null)
const recordsRoutePaths = ['/admin/records', '/account/records'] as const
const isRecordsScrollMode = computed(
  () =>
    recordsRoutePaths.some((path) => route.path === path) ||
    (navigationTarget.value !== null &&
      recordsRoutePaths.some((path) => navigationTarget.value === path)),
)

function finishNavigationFeedback(target: string) {
  if (navigationFeedbackTimer !== undefined) {
    window.clearTimeout(navigationFeedbackTimer)
  }
  navigationFeedbackTimer = window.setTimeout(() => {
    if (navigationTarget.value === target) {
      navigationTarget.value = null
    }
  }, 180)
}

function beginRouteTransition() {
  if (routeTransitionReleaseTimer !== undefined) {
    window.clearTimeout(routeTransitionReleaseTimer)
    routeTransitionReleaseTimer = undefined
  }
  isRouteTransitioning.value = true
}

function finishRouteTransition() {
  if (routeTransitionReleaseTimer !== undefined) {
    window.clearTimeout(routeTransitionReleaseTimer)
  }
  routeTransitionReleaseTimer = window.setTimeout(() => {
    isRouteTransitioning.value = false
    routeTransitionReleaseTimer = undefined
  }, 60)
}

async function handleMenuUpdate(key: string) {
  if (key === route.path) {
    return
  }
  navigationTarget.value = key
  await nextTick()
  try {
    const result = await router.push(key)
    if (
      isNavigationFailure(result, NavigationFailureType.cancelled) &&
      route.path !== key
    ) {
      await router.push(key)
    }
  } finally {
    finishNavigationFeedback(key)
  }
}

async function handleLogout() {
  await logout()
  setCurrentUser(null)
  hasLoadedUser.value = true
  toast.success(t('已退出登录', 'Signed out'))
  await router.push('/login')
}

function cycleTheme() {
  if (preference.value === 'system') {
    setThemePreference('light')
    return
  }
  if (preference.value === 'light') {
    setThemePreference('dark')
    return
  }
  setThemePreference('system')
}

const themeIcon = computed(() => {
  if (preference.value === 'system') {
    return Monitor
  }
  return isDark.value ? Moon : Sun
})

const languageLabel = computed(() => (language.value === 'zh' ? 'EN' : 'CN'))
const languageAriaLabel = computed(() => t('切换语言', 'Switch language'))
const themeAriaLabel = computed(() => t('切换主题', 'Switch theme'))
</script>

<template>
  <SidebarProvider class="app-shell" :default-open="true">
    <Sidebar class="app-sidebar" collapsible="icon" variant="inset">
      <SidebarHeader class="sidebar-brand-header">
        <div class="sidebar-brand-button">
          <span class="brand-mark">
            <img :src="logoUrl" alt="">
          </span>
          <span class="brand-copy group-data-[collapsible=icon]:hidden">
            <strong>{{ brandName }}</strong>
            <span>{{ brandSubtitle }}</span>
          </span>
        </div>
      </SidebarHeader>

      <SidebarContent class="sider-menu">
        <SidebarGroup v-if="isAdmin">
          <SidebarGroupLabel>{{ t('管理中心', 'Admin Center') }}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in adminMenuItems" :key="item.key">
                <SidebarMenuButton
                  data-navigation="true"
                  :is-active="selectedKey === item.key"
                  :tooltip="item.label"
                  @click="handleMenuUpdate(item.key)"
                >
                  <component :is="item.icon" />
                  <span>{{ item.label }}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>{{ t('我的账户', 'My Account') }}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in accountMenuItems" :key="item.key">
                <SidebarMenuButton
                  data-navigation="true"
                  :is-active="selectedKey === item.key"
                  :tooltip="item.label"
                  @click="handleMenuUpdate(item.key)"
                >
                  <component :is="item.icon" />
                  <span>{{ item.label }}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter class="sidebar-user-footer">
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <SidebarMenuButton size="lg" class="user-menu-button">
                  <span class="user-avatar"><UserRound /></span>
                  <span class="user-copy group-data-[collapsible=icon]:hidden">
                    <strong>{{ accountText }}</strong>
                    <span>{{ roleText }}</span>
                  </span>
                  <EllipsisVertical class="user-menu-chevron group-data-[collapsible=icon]:hidden" />
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                :side="isMobileViewport ? 'top' : 'right'"
                align="end"
                :side-offset="8"
                class="user-dropdown"
              >
                <DropdownMenuGroup>
                  <DropdownMenuItem variant="destructive" @select="handleLogout">
                    <LogOut />
                    <span>{{ t('退出登录', 'Sign out') }}</span>
                  </DropdownMenuItem>
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>

    <SidebarInset class="app-main md:!m-0 md:!rounded-none md:!shadow-none">
      <header class="app-header">
        <SidebarTrigger class="navigation-trigger" :aria-label="t('打开导航', 'Open navigation')" />
        <Separator orientation="vertical" class="header-divider data-[orientation=vertical]:h-4" />
        <div class="desktop-location">
          {{ currentNavigationLabel }}
        </div>
        <div class="mobile-brand" :aria-label="`${brandName} · ${accountText}`">
          <img class="mobile-brand-logo" :src="logoUrl" alt="" aria-hidden="true">
          <div class="mobile-brand-copy">
            <div class="mobile-title-row">
              <strong>{{ brandName }}</strong>
              <span class="mobile-version-badge">{{ appVersion }}</span>
            </div>
            <span>{{ accountText }} · {{ roleText }}</span>
          </div>
        </div>
        <div class="header-actions">
          <Button variant="ghost" size="icon" :aria-label="languageAriaLabel" @click="toggleLanguage">
            <Languages />
            <span class="sr-only">{{ languageLabel }}</span>
          </Button>
          <Button variant="ghost" size="icon" :aria-label="themeAriaLabel" @click="cycleTheme">
            <component :is="themeIcon" />
          </Button>
        </div>
      </header>

      <main
        class="content"
        :class="{
          'is-route-pending': isMenuNavigationPending,
          'is-route-transitioning': isRouteTransitioning,
          'is-records-scroll-mode': isRecordsScrollMode,
        }"
      >
        <div v-if="isMenuNavigationPending" class="route-progress" aria-hidden="true" />
        <div class="content-scroll">
          <RouterView v-slot="{ Component: RouteComponent, route: activeRoute }">
            <Transition
              name="route-fade"
              mode="out-in"
              @before-enter="beginRouteTransition"
              @after-enter="finishRouteTransition"
              @enter-cancelled="finishRouteTransition"
              @before-leave="beginRouteTransition"
              @after-leave="finishRouteTransition"
              @leave-cancelled="finishRouteTransition"
            >
              <component :is="RouteComponent" :key="activeRoute.name ?? activeRoute.path" />
            </Transition>
          </RouterView>
        </div>
      </main>
    </SidebarInset>
  </SidebarProvider>
</template>

<style scoped>
.app-shell {
  height: 100vh;
  height: 100dvh;
  min-height: 0;
  overflow: hidden;
  background: var(--sidebar);
}

.brand-mark {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  place-items: center;
  border: 1px solid var(--cpa-border);
  border-radius: 10px;
  overflow: hidden;
  background: var(--cpa-surface-solid);
  box-shadow: 0 1px 2px rgb(15 23 42 / 6%);
}

.brand-mark img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.brand-copy {
  display: grid;
  flex: 1 1 auto;
  min-width: 0;
  gap: 1px;
  text-align: left;
}

.mobile-title-row {
  display: flex;
  min-width: 0;
  align-items: center;
}

.brand-copy strong {
  color: var(--cpa-text-strong);
  font-size: 14px;
  font-weight: 600;
  line-height: 1.2;
}

.mobile-version-badge {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  max-width: 72px;
  height: 18px;
  padding: 0 6px;
  border: 1px solid var(--cpa-border);
  border-radius: 999px;
  overflow: hidden;
  color: var(--cpa-text-muted);
  background: var(--cpa-surface-raised);
  font-size: 11px;
  font-weight: 750;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.brand-copy > span {
  color: var(--cpa-text-muted);
  font-size: 12px;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  border: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
}

.app-main {
  display: flex;
  flex-direction: column;
  height: 100vh;
  height: 100dvh;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  border: 0;
  background: var(--cpa-canvas);
  box-shadow: var(--cpa-shadow-shell);
}

.content {
  position: relative;
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  padding: 0;
  scrollbar-width: thin;
  scrollbar-color: var(--content-scrollbar-thumb) transparent;
  --content-scrollbar-thumb: color-mix(in srgb, var(--cpa-text-muted) 44%, transparent);
  --content-scrollbar-thumb-hover: color-mix(
    in srgb,
    var(--cpa-primary) 58%,
    var(--cpa-text-muted)
  );
  background: var(--cpa-bg);
}

.content.is-route-pending {
  cursor: progress;
}

.route-progress {
  position: absolute;
  z-index: 3;
  top: 0;
  right: 0;
  left: 0;
  height: 2px;
  overflow: hidden;
  background: color-mix(in srgb, var(--cpa-primary) 12%, transparent);
  pointer-events: none;
}

.route-progress::before {
  display: block;
  width: 38%;
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--cpa-primary), var(--cpa-accent-blue));
  animation: route-progress-slide 900ms ease-in-out infinite;
  content: "";
}

.route-fade-enter-active,
.route-fade-leave-active {
  transition:
    opacity 180ms ease,
    transform 180ms ease;
}

.route-fade-enter-from,
.route-fade-leave-to {
  opacity: 0;
  transform: translateY(6px);
}

@keyframes route-progress-slide {
  0% {
    transform: translateX(-120%);
    opacity: 0.35;
  }

  50% {
    opacity: 0.9;
  }

  100% {
    transform: translateX(260%);
    opacity: 0.35;
  }
}

.mobile-brand {
  display: inline-grid;
  grid-template-columns: 28px minmax(0, auto);
  gap: 7px;
  align-items: center;
  justify-self: center;
  min-width: 0;
  max-width: calc(100vw - 116px);
}

.mobile-brand-logo {
  display: block;
  width: 28px;
  height: 28px;
  border-radius: 9px;
  object-fit: cover;
}

.mobile-brand-copy {
  display: grid;
  min-width: 0;
  gap: 1px;
  line-height: 1.1;
}

.mobile-brand-copy strong,
.mobile-brand-copy span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mobile-title-row {
  gap: 5px;
}

.mobile-brand-copy strong {
  color: var(--cpa-text-strong);
  font-size: 14px;
  font-weight: 760;
}

.mobile-version-badge {
  max-width: 58px;
  height: 16px;
  padding: 0 5px;
  font-size: 10px;
}

.mobile-brand-copy > span {
  color: var(--cpa-text-muted);
  font-size: 11px;
  font-weight: 600;
}

.sidebar-brand-header {
  padding: 12px;
  overflow: hidden;
}

:global([data-collapsible="icon"] [data-sidebar="header"].sidebar-brand-header) {
  padding: 8px;
}

.sidebar-brand-button {
  display: flex;
  align-items: center;
  height: 40px;
  gap: 10px;
  padding-inline: 6px;
  overflow: hidden;
  border-radius: var(--radius);
}

:global([data-collapsible="icon"] .sidebar-brand-button) {
  justify-content: center;
  padding-inline: 0;
  transform: translateX(5px);
}

.sidebar-user-footer {
  padding: 12px;
}

.user-avatar {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  place-items: center;
  border-radius: var(--radius);
  color: var(--sidebar-accent-foreground);
  background: var(--sidebar-accent);
}

.user-copy {
  display: grid;
  min-width: 0;
  gap: 2px;
  text-align: left;
}

.user-copy {
  flex: 1 1 auto;
}

.user-copy strong,
.user-copy span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-copy strong {
  color: var(--cpa-text-strong);
  font-size: 12px;
  font-weight: 500;
}

.user-copy span {
  color: var(--cpa-text-muted);
  font-size: 12px;
}

.user-menu-chevron {
  margin-left: auto;
  color: var(--cpa-text-muted);
}

.user-dropdown {
  width: var(--reka-dropdown-menu-trigger-width);
  min-width: 176px;
}

.app-header {
  display: grid;
  grid-template-columns: auto 1px minmax(0, 1fr) auto;
  flex: 0 0 56px;
  align-items: center;
  gap: 8px;
  height: 56px;
  padding: 0 20px;
  border-bottom: 1px solid var(--cpa-border);
  background: color-mix(in srgb, var(--cpa-canvas) 94%, transparent);
  backdrop-filter: blur(12px);
}

.navigation-trigger {
  width: 32px;
  height: 32px;
  color: var(--cpa-text-muted);
}

.header-divider {
  align-self: center;
  width: 1px;
  height: 16px;
  background: var(--cpa-border);
}

.header-divider[data-orientation="vertical"] {
  height: 16px;
}

.desktop-location {
  min-width: 0;
  overflow: hidden;
  color: var(--cpa-text-strong);
  font-size: 14px;
  font-weight: 720;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-actions {
  display: inline-flex;
  gap: 3px;
  align-items: center;
}

.header-actions button {
  color: var(--cpa-text-muted);
}

.content {
  display: flex;
  flex-direction: column;
}

.content-scroll {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 28px 32px 36px;
  scrollbar-gutter: auto;
  scrollbar-width: thin;
  scrollbar-color: var(--content-scrollbar-thumb) transparent;
  background: var(--cpa-canvas);
}

.content-scroll :deep(.page) {
  width: 100%;
  max-width: 1920px;
  margin-inline: auto;
}

.content-scroll::-webkit-scrollbar {
  width: 18px;
  height: 18px;
}

.content-scroll::-webkit-scrollbar-track,
.content-scroll::-webkit-scrollbar-corner {
  background: transparent;
}

.content-scroll::-webkit-scrollbar-thumb {
  min-height: 56px;
  border: 6px solid transparent;
  border-radius: 999px;
  background: var(--content-scrollbar-thumb);
  background-clip: content-box;
}

.content-scroll::-webkit-scrollbar-thumb:hover {
  background: var(--content-scrollbar-thumb-hover);
  background-clip: content-box;
}

.content.is-route-pending .content-scroll,
.content.is-route-transitioning .content-scroll {
  overflow: hidden;
}

.content.is-records-scroll-mode .content-scroll {
  scrollbar-gutter: auto;
  scrollbar-width: none;
}

.content.is-records-scroll-mode .content-scroll::-webkit-scrollbar {
  display: none;
}

@media (max-width: 768px) {
  .app-header {
    grid-template-columns: 36px minmax(0, 1fr) auto;
    gap: 6px;
    height: 58px;
    padding: 0 10px;
  }

  .header-divider,
  .desktop-location {
    display: none;
  }

  .mobile-brand {
    display: inline-grid;
  }

  .content-scroll {
    padding: 14px 16px 18px;
  }
}

@media (min-width: 769px) {
  .mobile-brand {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .route-progress::before {
    animation: none;
    opacity: 0.8;
    transform: none;
  }

  .route-fade-enter-active,
  .route-fade-leave-active {
    transition: opacity 80ms ease;
  }

  .route-fade-enter-from,
  .route-fade-leave-to {
    opacity: 0;
    transform: none;
  }
}
</style>
