import { describe, expect, it } from 'vitest'
import {
  CLUSTER_CREDENTIAL_INVALID_CODE,
  CLUSTER_CREDENTIAL_INVALID_PROBLEM_TYPE,
  PLATFORM_SESSION_EXPIRED_CODE,
  PLATFORM_SESSION_UNAUTHORIZED_PROBLEM_TYPE,
  shouldInvalidatePlatformSession,
} from './session-auth'

describe('shouldInvalidatePlatformSession', () => {
  it('only invalidates the AIOPS session for platform authentication failures', () => {
    expect(shouldInvalidatePlatformSession(401)).toBe(true)
    expect(shouldInvalidatePlatformSession(200, 401)).toBe(true)
    expect(shouldInvalidatePlatformSession(200, PLATFORM_SESSION_EXPIRED_CODE)).toBe(true)
    expect(shouldInvalidatePlatformSession(401, undefined, PLATFORM_SESSION_UNAUTHORIZED_PROBLEM_TYPE)).toBe(true)
  })

  it('keeps the AIOPS session for a Kubernetes credential failure', () => {
    expect(shouldInvalidatePlatformSession(200, CLUSTER_CREDENTIAL_INVALID_CODE)).toBe(false)
    expect(shouldInvalidatePlatformSession(424, undefined, CLUSTER_CREDENTIAL_INVALID_PROBLEM_TYPE)).toBe(false)
    expect(shouldInvalidatePlatformSession(401, undefined, CLUSTER_CREDENTIAL_INVALID_PROBLEM_TYPE)).toBe(false)
  })
})
