import { createApp } from 'vue'

import App from './app/App.vue'
import { router } from './app/router'
import { markAppReady } from './app/state/appStartup'
import './styles/tokens.css'

const app = createApp(App)

app.use(router)
app.mount('#app')

void router.isReady().then(markAppReady, markAppReady)

