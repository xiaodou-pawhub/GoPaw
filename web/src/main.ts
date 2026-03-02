import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
import naive from 'naive-ui'
import App from './App.vue'
import router from './router'
import i18n from './locales'
import './styles/global.scss'

const app = createApp(App)
const pinia = createPinia()

// 添加 Pinia 持久化插件 / Add Pinia persisted state plugin
pinia.use(piniaPluginPersistedstate)

app.use(pinia)
app.use(naive)
app.use(router)
app.use(i18n)

app.mount('#app')
