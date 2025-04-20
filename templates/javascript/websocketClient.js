"use strict";

// Language-specific boilerplate code
const languageBoilerplate = {
    python: "# Write your Python code here...\n\nif __name__ == '__main__':\n    print('Hello, Python!')",
    javascript: "// Write your JavaScript code here...\n\nconsole.log('Hello, JavaScript!');",
    java: "// Write your Java code here...\n\npublic class Main {\n    public static void main(String[] args) {\n        System.out.println(\"Hello, Java!\");\n    }\n}",
    c: "// Write your C code here...\n\n#include <stdio.h>\n\nint main() {\n    printf(\"Hello, C!\\n\");\n    return 0;\n}",
    cpp: "// Write your C++ code here...\n\n#include <iostream>\n\nint main() {\n    std::cout << \"Hello, C++!\" << std::endl;\n    return 0;\n}",
    go: "// Write your Go code here...\n\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, Go!\")\n}",
    ruby: "# Write your Ruby code here...\n\nputs 'Hello, Ruby!'",
    php: "// Write your PHP code here...\n\n<?php\n    echo 'Hello, PHP!';\n?>",
    rust: "// Write your Rust code here...\n\nfn main() {\n    println!(\"Hello, Rust!\");\n}",
    default: "// Write your code here...\n// You can select language to get the starter snippet from leetcode...\n// Start typing the 'QuestionSlug: two-sum' from leetcode, to load the question information",
};

let currentEditor = null; // Keep track of the current CodeMirror editor instance

function codeboxInit(language) {
    const codeboxElement = document.querySelector('#codebox');
    // Destroy the existing editor instance if it exists
    if (currentEditor) {
        currentEditor.toTextArea(); // Restore the original <textarea>
        currentEditor = null;
    }

    // Set the boilerplate code for the selected language
    let boilerplate = languageBoilerplate[language?.toLowerCase()] || languageBoilerplate.default;

    // Check the Selected Language boiler plate
    let codeEditorSnippet = undefined;
    if (language === 'python') {
        codeEditorSnippet = document.querySelector("#codeSnippetCode #pythonSnippet");
        boilerplate = codeEditorSnippet.textContent;
    }
    else if (language === 'java') {
        codeEditorSnippet = document.querySelector("#codeSnippetCode #javaSnippet");
        boilerplate = codeEditorSnippet.textContent;
    } else if (language === 'javascript') {
        codeEditorSnippet = document.querySelector("#codeSnippetCode #javascriptSnippet");
        boilerplate = codeEditorSnippet.textContent;
    }

    codeboxElement.value = boilerplate; // Use `.value` to set the content of the <textarea>

    // If the language is Java, use the "clike" mode
    language = language === 'java' ? 'clike' : language;

    // Initialize the CodeMirror editor
    currentEditor = CodeMirror.fromTextArea(codeboxElement, {
        lineNumbers: true,
        mode: { name: language?.toLowerCase() ?? "python" },
        theme: "eclipse",
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
        hintOptions: {
            completeSingle: false, // Prevent auto-selecting the first suggestion
        },
    });

    return currentEditor;
}

class WebSocketClient {
    constructor(roomId, editor) {
        this.roomId = roomId;
        this.editor = editor;
        const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
        this.wss = new WebSocket(`${protocol}://${window.location.host}/ws?room_id=${roomId}`);
        this.user_id = undefined;
        this.role = undefined;
        this.notificationContainer = this.createNotificationContainer();
        this.joinedUserElement = document.querySelector('#joinedUser');

        this.wss.addEventListener('open', (e) => {
            console.log('WebSocket connection opened:', e);
        });

        this.wss.addEventListener('message', (e) => {
            const message = JSON.parse(e.data);

            this.user_id = message.user_id;
            this.role = message.role;

            if (message.type === 'join') {
                this.showNotification(`${message.role} joined the room`, 'success');
                this.updateJoinedUser(message.role);
            } else if (message.type === 'leave') {
                this.showNotification(`${message.role} left the room`, 'warning');
                this.updateJoinedUser(null);
            } else if (message.type === 'code') {
                // Update editor content without triggering change event
                const currentCursor = this.editor.getCursor();
                this.editor.setValue(message.content);
                this.editor.setCursor(currentCursor);
            }

            // Update the problem title and description
            if (message.problem_title) {
                this.#updateProblemTitle(message.problem_title);
            }
            if (message.problem_description) {
                this.#updateProblemDescription(message.problem_description);
            }
        });

        this.wss.addEventListener('close', () => {
            this.showNotification('WebSocket connection closed.', 'error');
        });

        // Handle editor changes
        this.editor.on('change', (cm, change) => {
            if (change.origin !== 'setValue') {
                const content = cm.getValue();
                this.#sendCode(content);
            }
        });
    }

    createNotificationContainer() {
        const container = document.createElement('div');
        container.className = 'fixed top-4 right-4 z-50 flex flex-col gap-2';
        document.body.appendChild(container);
        return container;
    }

    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        const baseClasses = 'px-4 py-2 rounded-lg shadow-lg text-white transform transition-all duration-300 bottom-0 text-xs';
        const typeClasses = {
            success: 'bg-green-500',
            warning: 'bg-yellow-500',
            error: 'bg-red-500',
            info: 'bg-blue-500'
        };

        notification.className = `${baseClasses} ${typeClasses[type] || typeClasses.info}`;
        notification.textContent = message;

        this.notificationContainer.appendChild(notification);

        // Animate in
        setTimeout(() => {
            notification.style.opacity = '1';
            notification.style.transform = 'translateX(0)';
        }, 10);

        // Remove after 3 seconds
        setTimeout(() => {
            notification.style.opacity = '0';
            notification.style.transform = 'translateX(100%)';
            setTimeout(() => {
                this.notificationContainer.removeChild(notification);
            }, 300);
        }, 3000);
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

    updateJoinedUser(oppositeRole) {
        if (this.joinedUserElement) {
            if (oppositeRole) {
                // If I am Author, show Collaborator and vice versa
                const myRole = this.role;
                const displayRole = myRole === 'Author' ? 'Collaborator' : 'Author';
                this.joinedUserElement.textContent = `@${displayRole}`;
            } else {
                this.joinedUserElement.textContent = 'None';
            }
        }
    }
}

function runWebsocketProcess() {
    const roomId = document.querySelector("span#roomId").textContent.trim();
    let codeEditor = codeboxInit(); // Initialize with the default language

    // Initialize WebSocket connection
    let wss = new WebSocketClient(roomId, codeEditor);

    // Reload the code editor and WebSocket connection when the programming language changes
    const languageSelector = document.querySelector('#programmingLanguages');
    languageSelector.addEventListener('change', (event) => {
        const selectedLanguage = event.target.value.toLowerCase();

        // Destroy the existing WebSocket connection
        if (wss && wss.wss.readyState === WebSocket.OPEN) {
            wss.wss.close(); // Close the WebSocket connection
        }

        // Reinitialize the editor with the new language
        codeEditor = codeboxInit(selectedLanguage);

        // Reinitialize the WebSocket connection with the new editor
        wss = new WebSocketClient(roomId, codeEditor);
    });
}

runWebsocketProcess();