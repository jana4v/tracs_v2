import * as wamp from 'autobahn-browser/autobahn.min.js';

export default defineNuxtPlugin((nuxtApp) => {
  if (!import.meta.client) {
    return {
      provide: {
        wamp2: null, // Return null on the server side
      },
    };
  }

  return new Promise((resolve, reject) => {
    let host_name = window.location.hostname;
    if (host_name === 'localhost') {
      host_name = '127.0.0.1';
    }

    const connection = new wamp.Connection({
      url: `ws://${host_name}/ws`,
      realm: 'realm1',
    });

    connection.onopen = (session) => {
      console.log("WAMP Connected...");
      resolve({
        provide: {
          wamp2: session,  // Ensure we provide the session
        },
      });
    };

    connection.onclose = (reason, details) => {
      console.error(`WAMP connection closed: ${reason}`);
      resolve({
        provide: {
          wamp2: null, // Ensure `wamp` is always provided
        },
      });
    };

    connection.open();
  });
});
