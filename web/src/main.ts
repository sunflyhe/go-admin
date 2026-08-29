import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './styles/index.css'
import App from './App.vue'
import { router } from './router'
import { perm } from './directives/perm'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.directive('perm', perm)
app.mount('#app')
