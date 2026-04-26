

function getServerIpFromUrl(): string {
    const hostname = window.location.hostname; // Extracts the hostname (e.g., "172.10.22.33")
    return hostname;
  }

  // Use relative path to go through Nuxt proxy and avoid CORS issues
  const url = `/publish_to_wamp`;

  type WampMessage = {
      summary: string;
      status: string;
      progress: string; // Converted to string as per your Python code
  };
  
  export async function publishToWampTopic(WampMessage: WampMessage, topic: string): Promise<void> {
        // Publish the message to the topic
        // const { $wamp } = useNuxtApp();
        // console.log($wamp)
        // let x = await $wamp.publish(topic, [WampMessage]) // Arguments are passed as an array
        //   .then(() => {
        //     console.log(`Message published to topic: ${topic}`);
           
        //   })
        //   .catch((error: any) => {
        //     console.error("Error publishing to topic:", error);
    
        //   });

        let data = {
            topic:topic,
            msg:WampMessage
        };

        try {
            // Send the POST request using fetch
            const response = await fetch(url, {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
              },
              body: JSON.stringify(data), // Serialize the payload to JSON
            });
        
            // Check if the response is OK (status code 200-299)
            if (!response.ok) {
              throw new Error(`HTTP error! Status: ${response.status}`);
            }
                } catch (error) {
            console.error("Error publishing to WAMP:", error);
            throw error; // Re-throw the error for further handling
          }

      };
  