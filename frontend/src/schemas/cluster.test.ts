import { describe, expect, it } from 'vitest'
import { clusterImportSchema } from './cluster'

const validInput = {
  name: 'devops7.2',
  kubeconfig: 'apiVersion: v1',
  description: '',
}

describe('clusterImportSchema', () => {
  it('accepts dotted cluster names', () => {
    expect(clusterImportSchema.safeParse(validInput).success).toBe(true)
  })

  it('rejects unsupported cluster name characters', () => {
    expect(clusterImportSchema.safeParse({ ...validInput, name: 'devops 7.2' }).success).toBe(false)
  })
})
