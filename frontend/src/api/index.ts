import client from './client'
import type { Repository, Finding, Remediation, AnalysisResult, SecurityOverview } from '../types'

export const api = {
  getRepositories: async (): Promise<Repository[]> => {
    const res = await client.get('/repositories')
    return res.data.repositories
  },
  selectRepository: async (id: string): Promise<void> => {
    await client.post(`/repositories/${id}/select`)
  },
  deselectRepository: async (id: string): Promise<void> => {
    await client.delete(`/repositories/${id}/select`)
  },
  syncFindings: async (id: string): Promise<{ synced: number }> => {
    const res = await client.post(`/repositories/${id}/sync`)
    return res.data
  },
  getRepositoryOverview: async (id: string): Promise<SecurityOverview> => {
    const res = await client.get(`/repositories/${id}/overview`)
    return res.data
  },
  getFindings: async (repoId: string): Promise<Finding[]> => {
    const res = await client.get(`/repositories/${repoId}/findings`)
    return res.data.findings
  },
  analyzeFinding: async (id: string): Promise<AnalysisResult> => {
    const res = await client.post(`/findings/${id}/analyze`)
    return res.data
  },
  approveFinding: async (id: string): Promise<void> => {
    await client.post(`/findings/${id}/approve`)
  },
  declineFinding: async (id: string): Promise<void> => {
    await client.post(`/findings/${id}/decline`)
  },
  createPR: async (id: string): Promise<{ pr_url: string; pr_number: number }> => {
    const res = await client.post(`/findings/${id}/pr`)
    return res.data
  },
  getRemediations: async (): Promise<Remediation[]> => {
    const res = await client.get('/remediations')
    return res.data.remediations
  },
}