import * as wamp from "autobahn-browser/autobahn.min.js";

export default defineNuxtPlugin((nuxtApp) => {
  // Access Pinia inside plugin
  const tcExecutionStatusStore = tCstore();
  const on_tc_file_event = (args, kwargs) => {
    if (args && args.length) {
      kwargs = args[0];
    }
    tcExecutionStatusStore.setStore({
      summary: kwargs.summary || "",
      status: kwargs.status || "",
      progress: kwargs.progress || 0,
    });
  };
  const papertExecutionStatusStore = usePapertExecutionStatusStore();
  const on_papert_event = (args, kwargs) => {
    if (args && args.length) {
      kwargs = args[0];
    }
    console.log("Received papert event:", kwargs);
    papertExecutionStatusStore.setStore({
      summary: kwargs.summary || "",
      status: kwargs.status || "",
      progress: kwargs.progress || 0,
    });
  };

  if (!import.meta.client) {
    return {
      provide: {
        wamp: null, // Return null on the server side
      },
    };
  }

  return new Promise((resolve, reject) => {
    let host_name = window.location.hostname;
    if (host_name === "localhost") {
      host_name = "127.0.0.1";
    }

    const connection = new wamp.Connection({
      url: `ws://${host_name}/ws`,
      realm: "realm1",
    });

    connection.onopen = async (session) => {
      console.log("WAMP Connected...");
      await session.subscribe("com.tc_file.status", on_tc_file_event);
      console.log("Subscribed to com.tc_file.status");

      await session.subscribe("com.papert.status", on_papert_event);
      console.log("Subscribed to com.papert.status");

      // Subscribe to dialog requests with response callback
      const on_dialog_request = (args, kwargs) => {
        if (args && args.length) {
          kwargs = args[0];
        }
        const dialogStore = useDialogRequestStore();
        
        // Extract callback topic from kwargs
        const callbackTopic = kwargs.callback_topic;
        delete kwargs.callback_topic; // Remove it before passing to store
        
        dialogStore.setDialogRequest({
          ...kwargs,
          __resolve: (result) => {
            // Publish response back to backend on callback topic
            if (callbackTopic) {
              session.publish(callbackTopic, [result]);
              console.log(`Published response to ${callbackTopic}:`, result);
            }
          },
        });
      };
      await session.subscribe("com.app.show_form_dialog", on_dialog_request);
      console.log("Subscribed to com.app.show_form_dialog");

      resolve({
        provide: {
          wamp: session, // Ensure we provide the session
        },
      });
    };

    connection.onclose = (reason, details) => {
      console.error(`WAMP connection closed: ${reason}`);
      resolve({
        provide: {
          wamp: null, // Ensure `wamp` is always provided
        },
      });
    };

    connection.open();
  });
});
