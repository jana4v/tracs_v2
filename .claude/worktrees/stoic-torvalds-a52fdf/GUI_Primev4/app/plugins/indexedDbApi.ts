import { defineNuxtPlugin } from '#app';
import Localbase from 'localbase';

declare module '#app' {
  interface NuxtApp {
    $db: Localbase | null; // Add the `$db` property to the NuxtApp interface
    $dbUtils: {
      add: (key: string, item: any,collection?: string) => Promise<void>;
      get: (key: string, collection?: string) => Promise<any>;
      remove: (key: string, collection?: string) => Promise<void>;
    }; // Add utility functions
  }
}

export default defineNuxtPlugin((nuxtApp) => {
  // Check if we're running on the client side using import.meta.client
  if (import.meta.client) {
    const db = new Localbase('db');
    db.config.debug = false;

    // Define utility functions
    const add = async ( key: string, item: any, collection: string = 'scg' ) => {
      await db.collection(collection).add(item,key);
    };

    const get = async ( key: string, collection: string = 'scg') => {
      return await db.collection(collection).doc(key).get();
    };

    const remove = async ( key: string, collection: string = 'scg') => {
      await db.collection(collection).doc(key).delete();
    };

    return {
      provide: {
        db, // Provide the `db` instance
        dbUtils: {
          add,
          get,
          remove
        }, // Provide utility functions
      },
    };
  } else {
    return {
      provide: {
        db: null, // Provide `null` for server-side
        dbUtils: {
          addItem: () => Promise.reject('Not available on server'),
          getItem: () => Promise.reject('Not available on server'),
          deleteItem: () => Promise.reject('Not available on server'),
        }, // Provide fallbacks for server-side
      },
    };
  }
});