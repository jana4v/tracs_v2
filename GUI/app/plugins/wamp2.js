const NATS_COMPAT_STATE_KEY = 'nats-compat-client'

function createDisabledCompatClient() {
  return {
    isOpen: false,
    async subscribe(subject) {
      return {
        id: `disabled:${subject}`,
        subject,
      }
    },
    unsubscribe() {},
    publish() {},
    async call(subject) {
      return {
        data: null,
        error: `WAMP disabled: ${subject}`,
      }
    },
    close() {},
  }
}

export default defineNuxtPlugin(() => {
  const state = useState(NATS_COMPAT_STATE_KEY, () => createDisabledCompatClient())
  return {
    provide: {
      wamp2: state.value,
    },
  }
})
