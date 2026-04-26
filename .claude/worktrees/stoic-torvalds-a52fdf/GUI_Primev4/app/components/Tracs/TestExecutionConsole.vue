<template>
  <div>
    <div class="grid">
      <div class="col-12">
        <h1 class="mb-0 mt-3 text-3xl flex text-primary-600 font-bold">
          Status
        </h1>
        <ProgressBar :value="parseInt(progress)" class="mt-3 mb-3" />
        <div readonly class="fixed-size-textarea">
          <div class="mb-4 m-2 text-2xl flex text-primary-800 font-bold">
            {{ summary }}
          </div>
          <div
            v-for="(item, index) in status"
            :key="index"
            class="p-mb-2 m-2 text-2xl flex text-primary-700 font-italic"
          >
            {{ item }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  topic: String,
});
const { $wamp } = useNuxtApp();
const summary = ref("");
const status = ref([]);
const progress = ref(0);
let wamp_subscription = null;

const on_event = (args,kwargs) => {
  if(args.length){
    kwargs = args[0]
  }
  console.log(args,kwargs)
  summary.value = kwargs.summary;
  status.value.unshift(kwargs.status);
  if (status.value.length > 20) {
    status.value.pop();
  }
  progress.value = kwargs.progress;
};

onMounted(async () => {
  try {
    wamp_subscription = await $wamp.subscribe(props.topic, on_event);
    console.log(
      `Subscribed with topic:${props.topic} subscription ID:${wamp_subscription.id}`
    );
  } catch (error) {
    console.error(`Failed to subscribe to topic:${props.topic}`, error);
  }
});

onUnmounted(() => {
  try {
    if (wamp_subscription) $wamp.unsubscribe(wamp_subscription);
  } catch (error) {
    //console.error(`Failed to un subscribe to topic:${props.topic}`, error);
  }
});
</script>

<style lang="scss" scoped>
.fixed-size-textarea {
  width: 100%; /* Fixed width */
  height: 200px; /* Fixed height */
  overflow: auto; /* Adds a scrollbar if needed */
  resize: none; /* Prevents resizing */
  margin-top: 0;
  background-color: var(--surface-ground);
  border: 2px solid rgb(0, 0, 0);

  // Dark theme styles
  .dark & {
    background-color: var(--surface-ground);
    border: 2px solid rgb(111, 107, 107);
  }
}
</style>
