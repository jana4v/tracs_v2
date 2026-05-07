import { useIamAuth } from './auth'

function baseApiRoot() {
  if (!import.meta.client)
    return ''
  return `${window.location.protocol}//${window.location.host}/iam/api/v1`
}

function candidateUrls(path: string): string[] {
  const normalized = path.startsWith('/') ? path : `/${path}`
  const root = baseApiRoot()
  if (!root)
    return []

  if (normalized.startsWith('/iam/')) {
    return [`${root}${normalized}`]
  }

  return [`${root}/iam${normalized}`, `${root}${normalized}`]
}

export function useIamApi() {
  const auth = useIamAuth()

  async function withFallback<T>(path: string, options: any): Promise<T> {
    const urls = candidateUrls(path)
    let lastError: any

    for (const url of urls) {
      try {
        return (await $fetch(url, options)) as T
      }
      catch (err: any) {
        lastError = err
        const status = Number(err?.statusCode ?? err?.status ?? 0)
        if (status !== 404 && status !== 405) {
          throw err
        }
      }
    }

    throw lastError ?? new Error('IAM request failed')
  }

  async function request<T>(path: string, options: any = {}): Promise<T> {
    await auth.refreshIfNeeded()

    const token = auth.accessToken.value
    if (!token) {
      throw new Error('Authentication required')
    }

    const headers = {
      ...(options.headers || {}),
      Authorization: `Bearer ${token}`,
    }

    try {
      return await withFallback<T>(path, { ...options, headers })
    }
    catch (err: any) {
      const status = Number(err?.statusCode ?? err?.status ?? 0)
      if (status !== 401)
        throw err

      // Token may be expired despite local timer; force refresh and retry once.
      await auth.refreshIfNeeded(true)
      const retriedToken = auth.accessToken.value
      if (!retriedToken)
        throw err

      return await withFallback<T>(path, {
        ...options,
        headers: {
          ...(options.headers || {}),
          Authorization: `Bearer ${retriedToken}`,
        },
      })
    }
  }

  return { request }
}
