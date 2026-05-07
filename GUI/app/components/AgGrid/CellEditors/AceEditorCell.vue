<!-- AceEditorCell.vue -->
<script>
import ace from 'ace-builds'
import { defineComponent, nextTick } from 'vue'

ace.config.set('basePath', '/ace/src-min-noconflict')
export default defineComponent({
  name: 'AceEditorCell',
  data() {
    return {
      value: this.params.value,
      aceEditor: null,
      InitialValue: this.params.value,
      isEscapePressed: false,
    }
  },
  beforeUnmount() {
    // Remove event listener when the component is unmounted
    // window.removeEventListener('keydown', this.handleKeyDown);
  },

  mounted() {
    // focus on the input field once editing starts
    //  window.addEventListener('keydown', this.handleKeyDown);
    nextTick(() => {
      console.log(this.value)

      this.aceEditor = ace.edit(this.$refs.aceEditorContainer)
      this.aceEditor.setTheme('ace/theme/chrome')
      this.aceEditor.session.setMode('ace/mode/javascript')
      this.aceEditor.setValue(this.value)
      console.log(this.$refs.aceEditorContainer)

      // Auto-completion setup
      ace.config.loadModule('ace/ext/language_tools', () => {
        this.aceEditor.setOptions({
          enableBasicAutocompletion: true,
          enableSnippets: true,
          enableLiveAutocompletion: true,
        })

        this.aceEditor.completers = [
          {
            getCompletions: async (editor, session, pos, prefix, callback) => {
              console.log(prefix)
              if (prefix.length === 0) {
                callback(null, [])
                return
              }

              try {
                // const response = await fetch('YOUR_REST_API_URL', {
                //   method: 'POST',
                //   headers: {
                //     'Content-Type': 'application/json',
                //   },
                //   body: JSON.stringify({
                //     prefix: prefix,
                //   }),
                // });

                // const suggestions = await response.json();
                const suggestions = [
                  {
                    caption: 'suggestion1',
                    value: 'suggestion1 Value',
                    meta: 'suggestion1 Meta',
                  },
                  {
                    caption: 'suggestion2',
                    value: 'suggestion2 Value',
                    meta: 'suggestion2 Meta',
                  },
                ]
                const completions = suggestions.map(suggestion => ({
                  caption: suggestion.caption,
                  value: suggestion.value,
                  meta: suggestion.meta,
                }))

                callback(null, completions)
              }
              catch (error) {
                console.error(error)
                callback(null, [])
              }
            },
          },
        ]
      })

      // Next tick End
    })
  },

  methods: {
    getValue() {
      if (this.isEscapePressed) {
        this.isEscapePressed = false
        return this.InitialValue
      }
      return this.aceEditor.getValue()
    },

    handleKeyDown(event) {
      // var annot = this.aceEditor.getSession().getAnnotations();
      // console.log(event);
      // if (annot.length) event.stopPropagation();
      // //console.log(annot);

      if (event.ctrlKey && event.key === 's') {
        event.preventDefault() // Prevent the default browser save behavior
        event.stopPropagation()
        const enterEvent = new KeyboardEvent('keydown', {
          key: 'Enter',
          keyCode: 13,
          which: 13,
          ctrlKey: true,
          bubbles: true,
          cancelable: true,
        })
        this.$refs.aceEditorContainer.dispatchEvent(enterEvent)
      }
      else if (event.key === 'Escape') {
        this.isEscapePressed = true
        event.preventDefault()
        event.stopPropagation()
        const enterEvent = new KeyboardEvent('keydown', {
          key: 'Enter',
          keyCode: 13,
          which: 13,
          ctrlKey: true,
          bubbles: true,
          cancelable: true,
        })
        this.$refs.aceEditorContainer.dispatchEvent(enterEvent)
      }
      if (!event.ctrlKey) {
        event.stopPropagation()
      }
    },
  },
})
</script>

<template>
  <div ref="aceEditorContainer" class="editor" @keydown="handleKeyDown" />
</template>

<style>
.editor {
  position: absolute !important;
  z-index: 1;
  height: 400px;
  width: 800px;
  font-size: 25px;
}
</style>
