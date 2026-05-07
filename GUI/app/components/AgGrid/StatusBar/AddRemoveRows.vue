<script lang="ts">
// import { defineComponent, ref } from 'vue';
// import { agTableStore } from 'stores/ag_table';

// export default defineComponent({
//   name: 'AddRemoveRows',
//   setup(props, ctx) {
//     const agTable_store = agTableStore();

//     const no_of_rows = ref(1);
//     function addRows() {
//       const { params } = props;
//       // console.log(params.api);
//       let rows = [];
//       for (let i = 0; i < no_of_rows.value; i++) rows.push({});
//       params.api.applyTransaction({ add: rows });
//     }
//     function deleteSelectedRows() {
//       const { params } = props;
//       let selectedCellRange = params.api.getCellRanges();
//       //console.log(selectedCellRange);
//       if (selectedCellRange.length > 0) {
//         let selection_ranger = [
//           selectedCellRange[0].startRow.rowIndex,
//           selectedCellRange[0].endRow.rowIndex,
//         ].sort();
//         if (selection_ranger[0] != selection_ranger[1]) {
//           for (let i = selection_ranger[0]; i <= selection_ranger[1]; i++) {
//             try {
//               params.api.getRowNode(i).setSelected(true);
//               //console.log(i);
//             } catch {}
//           }
//         }
//       }
//       params.api.applyTransaction({ remove: params.api.getSelectedRows() });
//     }
//     function saveData() {
//       console.log('Save Clicked');
//       const { params } = props;
//       console.log(params);
//       const changedData = [];

//       params.api.forEachNode((rowNode) => {
//         if (rowNode.data && (rowNode.dirty || rowNode.updated)) {
//           changedData.push(rowNode.data);
//         }
//       });

//       console.log('Changed data:', changedData);
//       ctx.emit('save', 'Save Clicked');
//     }
//     return { addRows, deleteSelectedRows, saveData, agTable_store, no_of_rows };
//   },
// });
//
</script>

// <template>
  //   <q-btn-group>
    //     <div class="q-mr-md">
      //       <q-input
//         v-if="agTable_store.is_addRowsAllowed"
//         v-model="no_of_rows"
//         type="number"
//         outlined
//         dense
//         bg-color="yellow"
//         @click.stop=""
//       >
        //         <template #before>
          //           <q-btn
//             color="yellow"
//             glossy
//             icon-right="plus"
//             label="Add Rows"
//             @click="addRows"
//           />
          //
        </template>
        //
      </q-input>
      //
    </div>
    //     <q-btn
//       v-if="agTable_store.is_removeRowsAllowed"
//       color="orange"
//       glossy
//       icon-right="plus"
//       label="Delete Selected Rows"
//       @click="deleteSelectedRows"
//     />
    //     <q-btn
//       v-if="agTable_store.is_saveAllowed"
//       color="green"
//       rounded
//       glossy
//       icon-right="save"
//       label="Save Changes"
//       @click="saveData"
//     />
    //
  </q-btn-group>
  //
</template>

// <style scoped lang="scss">
// .input-field {
//   position: absolute;
//   top: 0;
//   left: 0;
//   right: 0;
//   bottom: 0;
//   opacity: 0;
//   z-index: 1;
// }
//
</style>
