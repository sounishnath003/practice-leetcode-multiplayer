"use strict";

function codeboxInit() {
    // Initialize the codebox with CodeMirror
    const editor = CodeMirror.fromTextArea(document.querySelector('#codebox'), {
        lineNumbers: true,
        mode: { name: "python" },
        theme: "hopscotch",
        font: "Fira Code, monospace",
        indent: 4,
        indentUnit: 4,
        smartIndent: true,
        moveOnDrag: true,
        autoCloseTags: true,
        autoCloseBrackets: true,
        lineWrapping: true,
        matchBrackets: true,
        foldGutter: true,
        gutters: ["CodeMirror-linenumbers", "CodeMirror-foldgutter"],
        extraKeys: { "Alt-F": "findPersistent", "Cmd-/": 'toggleComment' },
    });

    return editor;
}

const editor = codeboxInit();