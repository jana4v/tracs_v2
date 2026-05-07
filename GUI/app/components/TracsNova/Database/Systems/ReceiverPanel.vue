<template>
  <div class="rx-panel">
    <Toast />

    <div class="rx-panel-toolbar">
      <span class="panel-title">Receivers</span>
      <Button
        icon="pi pi-plus"
        label="Add Receiver"
        size="small"
        @click="addDraft"
      />
    </div>

    <div v-if="store.loading" class="rx-panel-loading">
      <ProgressSpinner style="width: 40px; height: 40px;" />
    </div>

    <ScrollPanel v-else style="height: calc(100vh - 10rem);">
      <div class="rx-panel-list">
        <div v-for="(_, idx) in drafts" :key="`draft-${idx}`" class="rx-panel-item">
          <TracsNovaDatabaseSystemsReceiverCard
            @saved="onDraftSaved(idx)"
            @deleted="removeDraft(idx)"
          />
        </div>

        <div v-for="rx in store.list" :key="rx.code" class="rx-panel-item">
          <TracsNovaDatabaseSystemsReceiverCard
            :receiver="rx"
            @saved="onSaved"
            @deleted="onDeleted"
          />
        </div>

        <div v-if="store.list.length === 0 && drafts.length === 0" class="rx-panel-empty">
          <i class="pi pi-inbox" style="font-size: 2rem; color: #334155;" />
          <p>No receivers found. Click <strong>Add Receiver</strong> to create one.</p>
        </div>
      </div>
    </ScrollPanel>
  </div>
</template>

<script lang="ts" setup>
import { useToast } from 'primevue/usetoast';
import { useReceiverStore } from '@/stores/tracsNova/receiver';

const store = useReceiverStore();
const toast = useToast();

const drafts = ref<number[]>([]);

onMounted(() => store.fetchAll());

function addDraft() {
  drafts.value.push(Date.now());
}

function removeDraft(idx: number) {
  drafts.value.splice(idx, 1);
}

function onDraftSaved(idx: number) {
  drafts.value.splice(idx, 1);
  toast.add({ severity: 'success', summary: 'Saved', detail: 'Receiver added successfully.', life: 3000 });
}

function onSaved() {
  toast.add({ severity: 'success', summary: 'Saved', detail: 'Receiver updated successfully.', life: 3000 });
}

function onDeleted() {
  toast.add({ severity: 'info', summary: 'Deleted', detail: 'Receiver removed from database.', life: 3000 });
}
</script>

<style scoped>
.rx-panel {
  padding: 1rem;
}

.rx-panel-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.panel-title {
  font-size: 1.4rem;
  font-weight: 700;
  color: #22d3ee;
}

.rx-panel-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 0.25rem 0.1rem;
}

.rx-panel-item {
  width: 100%;
}

.rx-panel-loading,
.rx-panel-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding: 3rem;
  color: #64748b;
  text-align: center;
}
</style>
