<script setup lang="ts">
import { useStorage } from '@vueuse/core'
import { useColorModeStore } from '~/stores/colorMode'

// Initialize the store
const colorModeStore = useColorModeStore()

const mode = useColorMode({
  emitAuto: true,
  modes: {
    contrast: 'dark contrast',
    cafe: 'cafe',
  },
})

const modes = ['dark', 'light', 'auto']
// const allModes = [...modes, 'cafe', 'contrast']

const { state, next } = useCycleList(modes, { initialValue: 'dark' })
const storedMode = useStorage('color-mode', 'auto')
watchEffect(() => {
  mode.value = storedMode.value = state.value
  colorModeStore.setCurrentMode(state.value) // Update the store
})
</script>

<template>
  <div>
    <client-only>
      <Button @click="next()">
        <i v-if="mode === 'dark'" i-carbon-moon inline-block align-middle class="align-middle" />
        <i v-if="mode === 'light'" i-carbon-sun inline-block align-middle class="align-middle" />
        <i v-if="mode === 'cafe'" i-carbon-cafe inline-block align-middle class="align-middle" />
        <i v-if="mode === 'contrast'" i-carbon-contrast inline-block align-middle class="align-middle" />
        <i v-if="mode === 'auto'" i-carbon-laptop inline-block align-middle class="align-middle" />
        <span class="ml-2 capitalize">{{ mode }}</span>
      </Button>
    </client-only>
  </div>
</template>

<style>
html.cafe {
    filter: sepia(0.9) hue-rotate(315deg) brightness(0.9);
}
html.contrast {
    filter: contrast(2);
}
</style>
