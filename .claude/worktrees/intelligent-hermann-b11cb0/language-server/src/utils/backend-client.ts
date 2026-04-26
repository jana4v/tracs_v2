// MongoDB client for fetching TM/TC/SCO mnemonics and procedure names
// Shared MongoDB instance with Julia backend

import { MongoClient, type Db } from 'mongodb'

const MONGO_URI = process.env.MONGO_URI || 'mongodb://localhost:27017'
const DB_NAME = process.env.ASTRA_DB || 'astra'

let client: MongoClient | null = null
let db: Db | null = null

// In-memory cache with TTL
interface CacheEntry<T> {
  data: T
  expires: number
}

const cache = new Map<string, CacheEntry<any>>()
const CACHE_TTL = 30_000 // 30 seconds

async function getDb(): Promise<Db> {
  if (!db) {
    client = new MongoClient(MONGO_URI)
    await client.connect()
    db = client.db(DB_NAME)
    console.log(`[LSP] Connected to MongoDB: ${MONGO_URI}/${DB_NAME}`)
  }
  return db
}

function getCached<T>(key: string): T | null {
  const entry = cache.get(key)
  if (entry && Date.now() < entry.expires) {
    return entry.data
  }
  cache.delete(key)
  return null
}

function setCache<T>(key: string, data: T): void {
  cache.set(key, { data, expires: Date.now() + CACHE_TTL })
}

export interface TMRef {
  bank: number
  mnemonic: string
  full_ref: string
  description: string
  data_type: string
  unit?: string
  subsystem: string
}

export interface TCRef {
  command: string
  full_ref: string
  description: string
  parameters: { name: string; type: string; required: boolean }[]
  subsystem: string
  category?: string
}

export interface SCORef {
  command: string
  full_ref: string
  description: string
  subsystem: string
  category?: string
}

/**
 * Fetch all TM mnemonics, optionally filtered by bank.
 */
export async function getTMMnemonics(bank?: number): Promise<TMRef[]> {
  const cacheKey = `tm_mnemonics_${bank ?? 'all'}`
  const cached = getCached<TMRef[]>(cacheKey)
  if (cached) return cached

  try {
    const database = await getDb()
    const filter = bank !== undefined ? { bank } : {}
    const results = await database.collection('tm_mnemonics').find(filter).toArray()
    const refs = results.map(doc => ({
      bank: doc.bank,
      mnemonic: doc.mnemonic,
      full_ref: doc.full_ref,
      description: doc.description,
      data_type: doc.data_type,
      unit: doc.unit,
      subsystem: doc.subsystem,
    }))
    setCache(cacheKey, refs)
    return refs
  } catch (e) {
    console.error('[LSP] Failed to fetch TM mnemonics:', e)
    return []
  }
}

/**
 * Fetch all TC mnemonics.
 */
export async function getTCMnemonics(): Promise<TCRef[]> {
  const cached = getCached<TCRef[]>('tc_mnemonics')
  if (cached) return cached

  try {
    const database = await getDb()
    const results = await database.collection('tc_mnemonics').find({}).toArray()
    const refs = results.map(doc => ({
      command: doc.command,
      full_ref: doc.full_ref,
      description: doc.description,
      parameters: doc.parameters || [],
      subsystem: doc.subsystem,
      category: doc.category,
    }))
    setCache('tc_mnemonics', refs)
    return refs
  } catch (e) {
    console.error('[LSP] Failed to fetch TC mnemonics:', e)
    return []
  }
}

/**
 * Fetch all SCO commands.
 */
export async function getSCOCommands(): Promise<SCORef[]> {
  const cached = getCached<SCORef[]>('sco_commands')
  if (cached) return cached

  try {
    const database = await getDb()
    const results = await database.collection('sco_commands').find({}).toArray()
    const refs = results.map(doc => ({
      command: doc.command,
      full_ref: doc.full_ref,
      description: doc.description,
      subsystem: doc.subsystem,
      category: doc.category,
    }))
    setCache('sco_commands', refs)
    return refs
  } catch (e) {
    console.error('[LSP] Failed to fetch SCO commands:', e)
    return []
  }
}

/**
 * Fetch all procedure names.
 */
export async function getProcedureNames(): Promise<string[]> {
  const cached = getCached<string[]>('procedure_names')
  if (cached) return cached

  try {
    const database = await getDb()
    const results = await database.collection('procedures').find({}, { projection: { name: 1 } }).toArray()
    const names = results.map(doc => doc.name as string)
    setCache('procedure_names', names)
    return names
  } catch (e) {
    console.error('[LSP] Failed to fetch procedure names:', e)
    return []
  }
}

/**
 * Get all TM full_ref strings as a Set (for validation).
 */
export async function getTMRefSet(): Promise<Set<string>> {
  const mnemonics = await getTMMnemonics()
  return new Set(mnemonics.map(m => m.full_ref))
}

/**
 * Get all TC full_ref strings as a Set.
 */
export async function getTCRefSet(): Promise<Set<string>> {
  const mnemonics = await getTCMnemonics()
  return new Set(mnemonics.map(m => m.full_ref))
}

/**
 * Get all SCO full_ref strings as a Set.
 */
export async function getSCORefSet(): Promise<Set<string>> {
  const commands = await getSCOCommands()
  return new Set(commands.map(c => c.full_ref))
}

/**
 * Clear the cache (e.g., when data changes).
 */
export function clearCache(): void {
  cache.clear()
}

/**
 * Disconnect from MongoDB.
 */
export async function disconnect(): Promise<void> {
  if (client) {
    await client.close()
    client = null
    db = null
  }
}
