import { useNuxtApp } from '#app';

export const rpc = async (rpc_name: string, args: any) => {
  try {
    const { $wamp } = useNuxtApp();

    if (!$wamp) {
      throw new Error("WAMP connection is not established yet");
    }

    console.log("RPC CALLED", rpc_name, args);

    return $wamp.call(rpc_name, args)
      .then((res: any) => {
        console.log("RPC Response:", res);
        return res;
      })
      .catch((err: any) => {
        let _err = err.error;
        err.args.forEach(e => {
          _err += ":"+e
        });
         
        console.error("RPC Error:", err);
        return { data: null, error: _err };
      });

  } catch (err) {
    console.error("RPC Execution Error:", err);
    return { data: null, error: err };
  }
};


export const wamp_publish = (topic: string, args: any[] = [], kwargs: Record<string, any> = {}) => {
  const { $wamp2 } = useNuxtApp();

  if (!$wamp2) {
    console.error("WAMP connection is not established yet.");
    return { publish: () => console.warn("WAMP not connected") };
  }
    try {
      if (!$wamp2.isOpen) {
        console.error("WAMP session is not open. Cannot publish.");
        return false;
      }
      $wamp2.publish(topic, args, kwargs);
      return true;
    } catch (error) {
      console.error("Error publishing to WAMP:", error);
      return false;
    }
  };

export const wamp_url_db_get_document = "scg.nosqlDb.read_document";
export const wamp_url_db_get_documentsWithQuery = "scg.nosqlDb.read_documents";
export const wamp_url_db_create_update_document = "scg.nosqlDb.create_or_update_document";
export const wamp_url_db_update_documentWithQuery = "scg.nosqlDb.update_documents";
export const wamp_url_db_delete_document = "scg.nosqlDb.delete_document";
export const wamp_url_db_delete_documentsWithQuery = "scg.nosqlDb.delete_documents";

export interface DbRequestArgs {
  db_name: string; // Required
  collection_name?: string;
  _key?: string;
  document?: Record<string, any>; // Allows any key-value pairs
  query?: string;
  bindvars?: Record<string, any>; // Allows key-value pairs
}
