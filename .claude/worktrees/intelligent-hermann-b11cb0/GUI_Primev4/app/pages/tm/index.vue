<script setup>
import { ref, onMounted, defineAsyncComponent, markRaw, computed } from 'vue';
import { GridLayout, GridItem } from 'vue3-grid-layout-next';
import { table_config_func} from '@/components/tm/tableConfigurations';
const mode = ref(false);
const draggable = ref(true);
const resizable = ref(true);
const responsive = ref(true)
const layout = ref([
    { x: 0, y: 0, w: 6, h: 10, i: '0', static: false },
    // { x: 4, y:0, w: 4, h: 10, i: '1', static: false },
    // { x: 8, y:0, w: 4, h: 10, i: '2', static: false },

]);


const resizeEvent = (i, newH, newW, newHPx, newWPx) => {
    table_height.value = (newHPx-100) + "px";
}

const table_height=ref("77vh");

</script>

<template>
        
    <div class="card-layout mt-25">
        <GridLayout :layout="layout" :responsive="responsive" :col-num="12" :row-height="85" :is-draggable="draggable" :is-resizable="resizable" :vertical-compact="true" :use-css-transforms="true">
            <GridItem v-for="item in layout" :key="item.i" :static="item.static" :x="item.x" :y="item.y" :w="item.w" :h="item.h" :i="item.i"  @resized="resizeEvent">
                <tmLiveData data_source="TM1" :columnDefs="table_config_func('telemetry#live').ColDefs" :table_height="table_height"></tmLiveData>
            </GridItem>
        </GridLayout>
    </div>
</template>

<style lang="scss" scoped>

.card-layout {
  overflow-x: scroll;
  height: 90vh;
  width: 100%
}

.vue-grid-layout {
  //  background: #eee;
}

.vue-grid-item:not(.vue-grid-placeholder) {
  //  background: #ccc;
    border: 1px solid black;
    touch-action: none; /* Add this line */
}

.vue-grid-item .resizing {
    opacity: 0.9;
}

.vue-grid-item .static {
    background: #cce;
}

.vue-grid-item .text {
    font-size: 24px;
    text-align: center;
    position: absolute;
    top: 0;
    bottom: 0;
    left: 0;
    right: 0;
    margin: auto;
    height: 100%;
    width: 100%;
}

.vue-grid-item .no-drag {
    height: 100%;
    width: 100%;
}

.vue-grid-item .minMax {
    font-size: 12px;
}

.vue-grid-item .add {
    cursor: pointer;
}

.vue-draggable-handle {
    position: absolute;
    width: 20px;
    height: 20px;
    top: 0;
    left: 0;
    background: url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='10' height='10'><circle cx='5' cy='5' r='5' fill='#999999'/></svg>") no-repeat;
    background-position: bottom right;
    padding: 0 8px 8px 0;
    background-repeat: no-repeat;
    background-origin: content-box;
    box-sizing: border-box;
    cursor: pointer;
}
</style>
