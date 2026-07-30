/** Business code reserved for an expired AIOPS login session. */
export const PLATFORM_SESSION_EXPIRED_CODE = 1002

/** RFC 9457 type for an invalid platform login session. */
export const PLATFORM_SESSION_UNAUTHORIZED_PROBLEM_TYPE = 'https://aiops.local/problems/unauthorized'

/**
 * Kubernetes authentication failures are request-local: they must never clear
 * the AIOPS session or navigate away from the current platform page.
 */
export const CLUSTER_CREDENTIAL_INVALID_CODE = 2001

/**
 * A valid AIOPS user session can still be unable to authenticate to a
 * configured Kubernetes cluster. This type must never trigger a platform
 * logout.
 */
export const CLUSTER_CREDENTIAL_INVALID_PROBLEM_TYPE = 'https://aiops.local/problems/cluster-credential-invalid'

export function shouldInvalidatePlatformSession(status: number, businessCode?: unknown, problemType?: unknown): boolean {
  if (businessCode === 401 || businessCode === PLATFORM_SESSION_EXPIRED_CODE) {
    return true
  }

  // Typed Problem Details responses carry the precise authentication scope.
  // Do not infer a platform-session failure from their HTTP status: an
  // upstream Kubernetes API may also legitimately return 401.
  if (typeof problemType === 'string' && problemType.length > 0) {
    return problemType === PLATFORM_SESSION_UNAUTHORIZED_PROBLEM_TYPE
  }

  // Preserve compatibility for existing, untyped HTTP 401 responses while
  // v1 endpoints are migrated to Problem Details.
  return status === 401
}
