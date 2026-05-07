// Composable helper for REST API access using Nuxt runtime config + useFetch
/**
 * Simplified API fetch with automatic error handling and WAMP notifications
 * @param {string} path - API endpoint path
 * @param {object} options - Fetch options (method, body, etc.)
 * @param {string} errorSummary - Error message summary for WAMP notification
 * @param {string} wampTopic - WAMP topic to publish errors to
 * @returns {Promise<any>} - Unwrapped data value or null on error
 */
import { wamp_publish } from './wampApi.js'

export async function useAPIFetch(path, options = {}) {
  const config = useRuntimeConfig()

  const baseURL
    = config?.public?.apiBase
      || (process.client ? `http://${window.location.hostname}/restApi` : '/restApi')

  const { data, error } = await useFetch(path, {
    ...options,
    baseURL,
    key: options.key ?? `${path}-${Date.now()}-${Math.random()}`,
  })

  return { data, error }
}
export async function useSimpleAPIFetch(path, options = {}, errorSummary = 'API Request Failed', wampTopic = null) {
  const { data, error } = await useAPIFetch(path, options)

  if (error?.value) {
    const errorDetail = error.value?.data?.detail || error.value?.message || 'Backend service is not available'
    console.error(`${errorSummary}:`, errorDetail)

    // Publish to WAMP if topic provided and publishToWampTopic is available
    if (wampTopic && process.client) {
      try {
        // await publishToWampTopic({
        //   summary: errorSummary,
        //   status: `${errorSummary} Error: ${errorDetail}`,
        //   progress: "0",
        // }, wampTopic);

        await wamp_publish(wampTopic, '', {
          summary: errorSummary,
          status: `${errorSummary} Error: ${errorDetail}`,
          progress: '0',
        })
      }
      catch (e) {
        console.error('WAMP publish failed:', e)
      }
    }

    return null
  }

  return data?.value
}
