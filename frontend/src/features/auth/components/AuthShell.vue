<script setup lang="ts">
import { logoUrl } from '@/shared/utils/assets'

defineProps<{
  sectionLabel: string
}>()
</script>

<template>
  <main class="auth-screen">
    <section class="auth-brand-panel" aria-hidden="true">
      <div class="brand-stage">
        <span class="brand-word brand-word-cpa">CPA</span>
        <span class="brand-word brand-word-helper">HELPER</span>
      </div>
    </section>

    <section class="auth-content" :aria-label="sectionLabel">
      <div class="brand-mark">
        <img :src="logoUrl" alt="">
      </div>
      <slot />
    </section>
  </main>
</template>

<style scoped>
.auth-screen {
  display: grid;
  grid-template-columns: minmax(420px, 1fr) minmax(420px, 1fr);
  height: 100vh;
  height: 100dvh;
  min-height: 0;
  overflow: auto;
  background: var(--background);
}

.auth-brand-panel {
  position: relative;
  display: grid;
  min-height: 100%;
  align-items: center;
  overflow: hidden;
  padding: 72px 48px;
  background:
    radial-gradient(circle at 18% 18%, color-mix(in oklch, var(--primary-foreground) 24%, transparent), transparent 32%),
    radial-gradient(circle at 82% 78%, color-mix(in oklch, var(--primary-foreground) 18%, transparent), transparent 38%),
    linear-gradient(145deg, color-mix(in oklch, var(--primary) 74%, var(--background)) 0%, var(--primary) 56%, color-mix(in oklch, var(--primary) 86%, black) 100%);
}

.auth-brand-panel::before {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(color-mix(in oklch, var(--primary-foreground) 5%, transparent) 1px, transparent 1px),
    linear-gradient(90deg, color-mix(in oklch, var(--primary-foreground) 5%, transparent) 1px, transparent 1px);
  background-size: 48px 48px;
  mask-image: linear-gradient(135deg, black, transparent 72%);
  content: "";
}

.brand-stage {
  position: relative;
  z-index: 1;
  display: grid;
  width: min(620px, 100%);
  gap: 8px;
  align-content: center;
  justify-self: center;
  font-weight: 820;
  line-height: 0.9;
  text-transform: uppercase;
}

.brand-word {
  display: block;
  max-width: 100%;
  overflow-wrap: anywhere;
  letter-spacing: 0;
}

.brand-word-cpa {
  justify-self: start;
  color: var(--primary-foreground);
  font-size: 148px;
}

.brand-word-helper {
  justify-self: end;
  color: color-mix(in oklch, var(--primary-foreground) 78%, var(--primary));
  font-size: 116px;
}

.auth-content {
  display: grid;
  min-width: 0;
  align-content: center;
  justify-items: center;
  gap: 24px;
  padding: 48px;
  background:
    radial-gradient(circle at 88% 10%, color-mix(in oklch, var(--primary) 10%, transparent), transparent 28%),
    var(--background);
}

.brand-mark {
  display: grid;
  width: 68px;
  height: 68px;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: calc(var(--radius) * 1.6);
  background: var(--card);
  box-shadow: var(--cpa-shadow-soft);
}

.brand-mark img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

@media (max-width: 1320px) {
  .brand-word-cpa {
    font-size: 128px;
  }

  .brand-word-helper {
    font-size: 102px;
  }
}

@media (max-width: 1180px) {
  .auth-screen {
    grid-template-columns: minmax(360px, 0.9fr) minmax(390px, 1.1fr);
  }

  .brand-word-cpa {
    font-size: 118px;
  }

  .brand-word-helper {
    font-size: 94px;
  }
}

@media (max-width: 900px) {
  .auth-screen {
    grid-template-columns: 1fr;
  }

  .auth-brand-panel {
    display: none;
  }

  .auth-content {
    min-height: 100%;
    align-content: start;
    gap: 16px;
    padding: max(28px, env(safe-area-inset-top)) 14px 20px;
  }

  .brand-mark {
    width: 56px;
    height: 56px;
    border-radius: calc(var(--radius) * 1.4);
  }
}

@media (max-width: 520px) {
  .auth-content {
    gap: 14px;
  }
}
</style>
