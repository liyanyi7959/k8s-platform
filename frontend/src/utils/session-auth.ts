/** Business code reserved for an expired AIOPS login session. */
export const PLATFORM_SESSION_EXPIRED_CODE = 1002

/**
 * Kubernetes authentication failures are request-local: they must never clear
 * the AIOPS session or navigate away from the current platform page.
 */
export const CLUSTER_CREDENTIAL_INVALID_CODE = 2001

export function shouldInvalidatePlatformSession(status: number, businessCode?: unknown): boolean {
  return status === 401 || businessCode === 401 || businessCode === PLATFORM_SESSION_EXPIRED_CODE
}
