<template>
    <div>
      <InputText
        v-model="value"
        @input="onInput"
        @blur="onBlur"
        ref="input"
      />
    </div>
  </template>
  
  <script>
  import { ref } from 'vue';
  import InputText from 'primevue/inputtext';
  
  export default {
    components: {
      InputText,
    },
    props: {
      initialValue: String,
      onUpdateValue: Function,
      onBlur: Function,
    },
    setup(props, { emit }) {
      const value = ref(props.initialValue);
  
      const onInput = (event) => {
        value.value = event.target.value;
        props.onUpdateValue(value.value);
      };
  
      const onBlur = () => {
        if (props.onBlur) {
          props.onBlur();
        }
      };
  
      return {
        value,
        onInput,
        onBlur,
      };
    },
  };
  </script>