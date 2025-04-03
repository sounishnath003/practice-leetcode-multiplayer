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
        this.user_id = undefined;

        this.wss.addEventListener('open', (e) => {
            console.log('websocket.connection.open', e);
        });

        this.wss.addEventListener('message', (e) => {
            const message = JSON.parse(e.data);
            this.user_id = message.user_id;

            if (message.type === 'code') {
                // Update editor content without triggering change event
                const currentCursor = this.editor.getCursor();
                this.editor.setValue(message.content);
                this.editor.setCursor(currentCursor);
            }

            // Update the problemTitle and problemDescription Div
            this.#updateProblemTitle(message.problem_title);
            this.#updateProblemDescription(message.problem_description);

        });

        // Handle editor changes
        this.editor.on('change', (cm, change) => {
            if (change.origin !== 'setValue') {
                const content = cm.getValue();
                this.#sendCode(content);
            }
        });
    }

    #sendCode(content) {
        if (this.wss.readyState === WebSocket.OPEN) {
            const message = {
                type: 'code',
                room_id: this.roomId,
                content: content,
                user_id: this.user_id,
                problem_title: this.#getProblemTitle(),
                problem_description: this.#getProblemDescription(),
            };
            this.wss.send(JSON.stringify(message));
        }
    }

    #getProblemTitle() {
        const questionTitle = document.querySelector("h2#questionTitle")?.textContent.trim();
        return questionTitle;
    }
    #updateProblemTitle(updatedProblemTitle) {
        const questionTitle = document.querySelector("h2#questionTitle");
        if (questionTitle) {
            questionTitle.textContent = updatedProblemTitle;
        }
    }

    #getProblemDescription() {
        const problemDescription = document.querySelector("div#problemDescription")?.innerHTML.trim();
        return problemDescription || `<div class="tracking-normal flex flex-col items-center h-screen text-center text-red-700 text-medium text-sm">
        <div class="p-2 rounded-lg bg-red-50">Search for questions from
            the search box.</div>
    </div>`;
    }

    #updateProblemDescription(updatedProblemDescriptionContent) {
        const problemDescription = document.querySelector("div#problemDescription");
        if (problemDescription) {
            problemDescription.innerHTML = updatedProblemDescriptionContent;
        }
    }

}

function runWebsocketProcess() {
    const roomId = document.querySelector("span#roomId").textContent.trim();
    const codeEditor = codeboxInit();

    const wss = new WebSocketClient(roomId, codeEditor);
}

runWebsocketProcess();