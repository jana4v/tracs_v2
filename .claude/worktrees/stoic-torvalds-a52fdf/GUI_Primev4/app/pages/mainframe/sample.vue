

<script setup>
import { ref, onMounted } from "vue";
import MonacoEditor from "@guolao/vue-monaco-editor";
import * as monaco from "monaco-editor";

const editorContent = ref(`myFunction();`);

onMounted(() => {
  // Check if language is already registered
  if (!monaco.languages.getLanguages().some(lang => lang.id === "myCustomLang")) {
    monaco.languages.register({ id: "myCustomLang" });

    monaco.languages.setMonarchTokensProvider("myCustomLang", {
      tokenizer: {
        root: [
          [/\b(myFunction)\b/, "keyword"],
          [/\b(myVariable)\b/, "variable"],
        ],
      },
    });

    // Prevent duplicate autocomplete providers
    monaco.languages.registerCompletionItemProvider("myCustomLang", {
      provideCompletionItems: () => {
        return {
          suggestions: [
            {
              label: "myFunction",
              kind: monaco.languages.CompletionItemKind.Function,
              insertText: "myFunction();",
              documentation: "This is a custom function",
            },
            {
              label: "myVariable",
              kind: monaco.languages.CompletionItemKind.Variable,
              insertText: "myVariable",
              documentation: "This is a custom variable",
            },
          ],
        };
      },
    });
  }
});
</script>

<template>
     <div class="mt-50">
  <MonacoEditor
    v-model="editorContent"
    language="myCustomLang"
    theme="vs-dark"
    height="400px"
    width="400px"
    :options="{
      quickSuggestions: true, // Disables inline autocomplete popups
      parameterHints: { enabled: false }, // Disables function parameter hints
      minimap: { enabled: false }, // Hides minimap
      scrollbar: { vertical: 'hidden', horizontal: 'hidden' }, // Removes scrollbars
      overviewRulerLanes: 0 // Hides overview ruler
    }"
    class="custom-editor"
  />
</div>
</template>

<style>
/* Change Monaco Editor background */
.custom-editor .monaco-editor {
  background-color: #b4a7a7 !important; /* Dark grey */
}
</style>