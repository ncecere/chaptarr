import type { DownloadClient, DownloadClientInput, Indexer, IndexerInput } from './api'

// Saved services come back without their secrets. An empty string in the input
// tells the server to keep the stored value, so these never carry a secret.

export function toIndexerInput(i: Indexer): IndexerInput {
  return {
    name: i.name,
    type: i.type,
    url: i.url,
    apiKey: '',
    categories: i.categories,
    enabled: i.enabled,
    priority: i.priority,
  }
}

export function toClientInput(c: DownloadClient): DownloadClientInput {
  return {
    name: c.name,
    type: c.type,
    url: c.url,
    username: c.username,
    password: '',
    apiKey: '',
    category: c.category,
    enabled: c.enabled,
  }
}
