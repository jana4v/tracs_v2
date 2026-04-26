
<script >
import { defineComponent, toRefs, reactive, ref } from 'vue';
export default defineComponent({
  name: 'CheckBoxCellEditor',
  components: {

  },
  props: ['params'],
  setup(props) {

    // eslint-disable-next-line vue/no-setup-props-destructure
    let { params } = props;

    const checked = ref(params.node.isSelected());

    const onCheckboxChange = () => {
      const currentNode = params.node;
      if (currentNode.group) {
        // If it's a group node, select/deselect all child nodes
        currentNode.setSelected(checked.value, true);
      } else {
        // Update individual node selection
        currentNode.setSelected(checked.value);
        // Propagate the change up to update the group node selection state
        updateGroupNodeSelection(currentNode);
      }
    };

    const updateGroupNodeSelection = (node) => {
      // Implement logic to update the group node selection based on child nodes
      // This could involve checking if all/some/none of the children are selected
      // and updating the group node's checkbox accordingly
    };

    // If the node is a group node and its selection changes, update the checkbox state
    watch(
      () => params.node.isSelected(),
      (newVal) => {
        if (params.node.group) {
          checked.value = newVal;
        }
      }
    );
   

    const getValue = () => {
        return checked.value;
    };

    const getGui = () => {

    };
    const afterGuiAttached = () => {
    };

    const isPopup = () => {
      // and we could leave this method out also, false is the default
      return false;
    };


    const close_editor = () => {
      params.stopEditing();
    }

    function onKeyDown(event) {
      const key = event.key;
      if (key == 'Enter') {
        event.preventDefault();
        event.stopPropagation();
      }
    }

    return {
      getValue,
      checked,
      props,
      onCheckboxChange,      
      afterGuiAttached,
      isPopup,
      close_editor,
      getGui,
      onKeyDown,
    };
  },
  mounted() {
    // nextTick(() => {
    //   this.$refs.container.focus();
    // });
    //console.log(this.$refs.dropdownRef);
    //this.$refs.dropdownRef.showPopup('');
  },
});
</script>

<template>
  <Checkbox v-model="checked" @change="onCheckboxChange" />
</template>
<style>
.center-cell {
  text-align: center;
  vertical-align: middle;
}
</style>
