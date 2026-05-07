class IndexedDBHelper {
  private dbName: string
  private storeName: string

  constructor(dbName: string, storeName: string) {
    this.dbName = dbName
    this.storeName = storeName
  }

  /** Open IndexedDB database */
  private async openDB(): Promise<IDBDatabase> {
    return new Promise((resolve, reject) => {
      if (typeof window === 'undefined') {
        return reject(new Error('IndexedDB is not available in a server environment.'))
      }

      const request: IDBOpenDBRequest = indexedDB.open(this.dbName, 1)

      request.onupgradeneeded = (event: IDBVersionChangeEvent) => {
        const db = (event.target as IDBOpenDBRequest).result
        if (!db.objectStoreNames.contains(this.storeName)) {
          db.createObjectStore(this.storeName, { keyPath: 'id', autoIncrement: true })
        }
      }

      request.onsuccess = () => resolve(request.result)
      request.onerror = () => reject(new Error('Failed to open IndexedDB.'))
    })
  }

  /** Add or update a record */
  async addData<T extends { id?: number }>(data: T): Promise<IDBValidKey> {
    const db = await this.openDB()
    return new Promise((resolve, reject) => {
      const tx = db.transaction(this.storeName, 'readwrite')
      const store = tx.objectStore(this.storeName)
      const request = store.put(data) // put() handles both insert & update

      request.onsuccess = () => resolve(request.result)
      request.onerror = () => reject(new Error('Failed to add/update data.'))
    })
  }

  /** Get a record by key */
  async getDataByKey<T>(id: IDBValidKey): Promise<T | null> {
    const db = await this.openDB()
    return new Promise((resolve, reject) => {
      const tx = db.transaction(this.storeName, 'readonly')
      const store = tx.objectStore(this.storeName)
      const request = store.get(id)

      request.onsuccess = () => resolve(request.result || null)
      request.onerror = () => reject(new Error('Failed to retrieve data.'))
    })
  }

  /** Get all records */
  async getAllData<T>(): Promise<T[]> {
    const db = await this.openDB()
    return new Promise((resolve, reject) => {
      const tx = db.transaction(this.storeName, 'readonly')
      const store = tx.objectStore(this.storeName)
      const request = store.getAll()

      request.onsuccess = () => resolve(request.result)
      request.onerror = () => reject(new Error('Failed to retrieve all data.'))
    })
  }

  /** Delete a record by key */
  async deleteData(id: IDBValidKey): Promise<string> {
    const db = await this.openDB()
    return new Promise((resolve, reject) => {
      const tx = db.transaction(this.storeName, 'readwrite')
      const store = tx.objectStore(this.storeName)
      const request = store.delete(id)

      request.onsuccess = () => resolve('Deleted successfully.')
      request.onerror = () => reject(new Error('Failed to delete data.'))
    })
  }
}

export default defineNuxtPlugin(() => {
  return {
    provide: {
      indexedDB: new IndexedDBHelper('scg', 'rf'),
    },
  }
})
