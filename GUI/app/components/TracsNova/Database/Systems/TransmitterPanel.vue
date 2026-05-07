<template>
  <div class="tx-panel">
    <Toast />

    <!-- Toolbar -->
    <div class="tx-panel-toolbar">
      <span class="panel-title">Transmitters</span>
      <Button
        icon="pi pi-plus"
        label="Add Transmitter"
        size="small"
        @click="addDraft"
      />
    </div>

    <!-- Loading state -->
    <div v-if="store.loading" class="tx-panel-loading">
      <ProgressSpinner style="width: 40px; height: 40px;" />
    </div>

    <!-- Scrollable list -->
    <ScrollPanel v-else style="height: calc(100vh - 10rem);">
      <div class="tx-panel-list">
        <!-- Draft (new) cards first -->
        <div v-for="(_, idx) in drafts" :key="`draft-${idx}`" class="tx-panel-item">
          <TracsNovaDatabaseSystemsTransmitterCard
            @saved="onDraftSaved(idx)"
            @deleted="removeDraft(idx)"
          />
        </div>

        <!-- Existing transmitters -->
        <div v-for="tx in store.list" :key="tx.code" class="tx-panel-item">
          <TracsNovaDatabaseSystemsTransmitterCard
            :transmitter="tx"
            @saved="onSaved"
            @deleted="onDeleted"
          />
        </div>

        <!-- Empty state -->
        <div v-if="store.list.length === 0 && drafts.length === 0" class="tx-panel-empty">
          <i class="pi pi-inbox" style="font-size: 2rem; color: #334155;" />
          <p>No transmitters found. Click <strong>Add Transmitter</strong> to create one.</p>
        </div>
      </div>
    </ScrollPanel>
  </div>
</template>

<script lang="ts" setup>
import { useToast } from 'primevue/usetoast';
import { useTransmitterStore } from '@/stores/tracsNova/transmitter';

const store = useTransmitterStore();
const toast = useToast();

// Tracks how many "new" blank cards are visible
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
  toast.add({ severity: 'success', summary: 'Saved', detail: 'Transmitter added successfully.', life: 3000 });
}

function onSaved() {
  toast.add({ severity: 'success', summary: 'Saved', detail: 'Transmitter updated successfully.', life: 3000 });
}

function onDeleted() {
  toast.add({ severity: 'info', summary: 'Deleted', detail: 'Transmitter removed from database.', life: 3000 });
}
</script>

<style scoped>
.tx-panel {
  padding: 1rem;
}

.tx-panel-toolbar {
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

.tx-panel-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 0.25rem 0.1rem;
}

.tx-panel-item {
  width: 100%;
}

.tx-panel-loading,
.tx-panel-empty {
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
