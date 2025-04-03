"use strict";

function codeboxInit() {
    document.querySelector('#codebox').textContent = '# Write some code...';
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


class WebSocketClient {
    constructor(roomId, editor) {
        this.roomId = roomId;
        this.editor = editor;
        this.wss = new WebSocket(`ws://localhost:3000/ws?room_id=${roomId}`);

        this.wss.addEventListener('open', (e) => {
            console.log('websocket.connection.open', e);
        });

        this.wss.addEventListener('message', (e) => {
            const message = JSON.parse(e.data);

            if (message.type === 'code') {
                // Update editor content without triggering change event
                const currentCursor = this.editor.getCursor();
                this.editor.setValue(message.content);
                this.editor.setCursor(currentCursor);
            }
        });

        // Handle editor changes
        this.editor.on('change', (cm, change) => {
            if (change.origin !== 'setValue') {
                const content = cm.getValue();
                this.sendCode(content);
            }
        });
    }

    sendCode(content) {
        if (this.wss.readyState === WebSocket.OPEN) {
            const message = {
                type: 'code',
                room_id: this.roomId,
                content: content
            };
            this.wss.send(JSON.stringify(message));
        }
    }
}

function runWebsocketProcess() {
    const roomId = document.querySelector("span#roomId").textContent.trim();
    const codeEditor = codeboxInit();

    const wss = new WebSocketClient(roomId, codeEditor);
}

runWebsocketProcess();