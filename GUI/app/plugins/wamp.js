const NATS_COMPAT_STATE_KEY = 'nats-compat-client'

function createDisabledCompatClient() {
  return {
    isOpen: false,

    async subscribe(subject, handler) {
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
  if (!import.meta.client) {
    return {
      provide: {
        wamp: createDisabledCompatClient(),
      },
    }
  }

  return new Promise((resolve) => {
    const state = useState(NATS_COMPAT_STATE_KEY, () => null)
    if (state.value) {
      resolve({
        provide: {
          wamp: state.value,
        },
      })
      return
    }

    state.value = createDisabledCompatClient()
    resolve({
      provide: {
        wamp: state.value,
      },
    })
  })
})
