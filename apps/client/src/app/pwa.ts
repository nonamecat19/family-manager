import { registerSW } from 'virtual:pwa-register'

registerSW({
  onNeedRefresh() {
    // Show a toast or button to refresh
  },
  onOfflineReady() {
    console.log('App is ready to work offline')
  },
})
