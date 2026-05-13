// Language-specific boilerplate code
const languageBoilerplate = {
    python: "# Write your Python code here...\n\nif __name__ == '__main__':\n    print('Hello, Python!')",
    javascript: "// Write your JavaScript code here...\n\nconsole.log('Hello, JavaScript!');",
    java: "// Write your Java code here...\n\npublic class Solution {\n    public static void main(String[] args) {\n        System.out.println(\"Hello, Java!\");\n    }\n}",
    c: "// Write your C code here...\n\n#include <stdio.h>\n\nint main() {\n    printf(\"Hello, C!\\n\");\n    return 0;\n}",
    cpp: "// Write your C++ code here...\n\n#include <iostream>\n\nint main() {\n    std::cout << \"Hello, C++!\" << std::endl;\n    return 0;\n}",
    go: "// Write your Go code here...\n\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, Go!\")\n}",
    ruby: "# Write your Ruby code here...\n\nputs 'Hello, Ruby!'",
    php: "// Write your PHP code here...\n\n<?php\n    echo 'Hello, PHP!';\n?>",
    rust: "// Write your Rust code here...\n\nfn main() {\n    println!(\"Hello, Rust!\");\n}",
    default: "// Write your code here...\n// You can select language to get the starter snippet from leetcode...\n// Start typing the 'QuestionSlug: two-sum' from leetcode, to load the question information",
};

let currentEditor = null; // Keep track of the current Monaco editor instance
let isRemoteUpdate = false; // Flag to prevent infinite sync loops

function codeboxInit(language, cachedContent, callback) {
    const codeboxElement = document.querySelector('#codebox');
    
    // Normalize language
    language = language?.toLowerCase();
    if (language === 'c++') language = 'cpp';
    const monacoLang = language === 'java' ? 'java' : (language === 'cpp' ? 'cpp' : language);

    // Set the boilerplate code for the selected language
    let boilerplate = languageBoilerplate[language] || languageBoilerplate.default;

    // Check the Selected Language boiler plate from hidden snippets if they exist
    let codeEditorSnippet = undefined;
    if (language === 'python') {
        codeEditorSnippet = document.querySelector("#codeSnippetCode #pythonSnippet");
        if (codeEditorSnippet) boilerplate = codeEditorSnippet.textContent;
    }
    else if (language === 'java') {
        codeEditorSnippet = document.querySelector("#codeSnippetCode #javaSnippet");
        if (codeEditorSnippet) boilerplate = codeEditorSnippet.textContent;
    } else if (language === 'javascript') {
        codeEditorSnippet = document.querySelector("#codeSnippetCode #javascriptSnippet");
        if (codeEditorSnippet) boilerplate = codeEditorSnippet.textContent;
    } else if (language === 'cpp') {
        codeEditorSnippet = document.querySelector("#codeSnippetCode #cppSnippet");
        if (codeEditorSnippet) boilerplate = codeEditorSnippet.textContent;
    }

    const initialContent = cachedContent !== undefined ? cachedContent : (boilerplate.trim().length == 0 ? languageBoilerplate[language] : boilerplate.trim());

    const initMonaco = () => {
        require(['vs/editor/editor.main'], function() {
            if (currentEditor) {
                const model = monaco.editor.createModel(initialContent, monacoLang);
                currentEditor.setModel(model);
            } else {
                currentEditor = monaco.editor.create(codeboxElement, {
                    value: initialContent,
                    language: monacoLang,
                    theme: 'vs-dark',
                    automaticLayout: true,
                    fontSize: 14,
                    lineHeight: 22,
                    minimap: { enabled: false },
                    scrollbar: {
                        vertical: 'visible',
                        horizontal: 'visible'
                    },
                    tabSize: 4,
                    insertSpaces: true,
                    suggestOnTriggerCharacters: true,
                    acceptSuggestionOnEnter: 'on',
                    folding: true,
                });

                // Add Cmd+Enter / Ctrl+Enter shortcut to run code
                currentEditor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, function() {
                    const runBtn = document.getElementById('run-code-btn');
                    if (runBtn) runBtn.click();
                });
            }
            
            if (callback) callback(currentEditor);
        });
    };

    if (typeof require === 'undefined') {
        // Wait for loader if called too early (e.g. during HTMX swap)
        const checkRequire = setInterval(() => {
            if (typeof require !== 'undefined') {
                clearInterval(checkRequire);
                initMonaco();
            }
        }, 50);
        // Timeout after 5 seconds to prevent infinite loop
        setTimeout(() => clearInterval(checkRequire), 5000);
    } else {
        initMonaco();
    }

    return currentEditor;
}

class WebSocketClient {
    constructor(roomId, editor, onLanguageChange) {
        this.roomId = roomId;
        this.editor = editor;
        this.onLanguageChange = onLanguageChange;
        const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
        this.wss = new WebSocket(`${protocol}://${window.location.host}/ws?room_id=${roomId}`);
        this.user_id = undefined;
        this.role = undefined;
        this.roomUsers = new Map(); // Track users in the room
        this.notificationContainer = this.createNotificationContainer();
        this.joinedUserElement = document.querySelector('#joinedUser');
        this.webrtcHandler = null;
        this.remoteDecorations = []; // To track and clear remote change highlights

        // Call readiness state
        this.localCallReady = false;
        this.remoteCallReady = false;
        this.callTimerInterval = null;

        // Question block observer
        this.questionBlock = document.getElementById('questionBlock');
        this.observer = null;
        this.setupQuestionObserver();

        // Listen for HTMX swaps to re-attach observer and sync new content
        document.body.addEventListener('htmx:afterSwap', (event) => {
            this.setupQuestionObserver();
            if (this.editor) {
                this.#sendCode(this.editor.getValue());
            }
        });

        this.wss.addEventListener('open', (e) => {
            console.log('WebSocket connection opened:', e);
        });

        this.wss.addEventListener('message', (e) => {
            const message = JSON.parse(e.data);
            console.log('Received message:', message);

            if (this.webrtcHandler && ['offer', 'answer', 'ice-candidate'].includes(message.type)) {
                this.webrtcHandler.handleMessage(message);
            }

            if (message.type === 'join') {
                this.roomUsers.set(message.user_id, {
                    role: message.role,
                    userId: message.user_id
                });
                this.showNotification(`${message.role} joined the room`, 'success');
                this.updateJoinedUser(message.role);
                this.initializeWebRTC();
            } else if (message.type === 'leave') {
                this.roomUsers.delete(message.user_id);
                this.showNotification(`${message.role} left the room`, 'warning');
                this.updateJoinedUser(null);
                if (this.webrtcHandler) {
                    this.webrtcHandler.disconnect();
                }
            } else if (message.type === 'language_change') {
                if (this.onLanguageChange) {
                    this.onLanguageChange(message.language);
                }
            } else if (message.type === 'call_ready') {
                this.remoteCallReady = true;
                this.showNotification(`${message.role} is ready to call!`, 'success');
                this.checkAutoConnect();
            } else if (message.type === 'call_ended') {
                this.endCall(false);
                this.showNotification(`${message.role} ended the call`, 'info');
            } else if (message.type === 'code') {
                if (this.editor && message.user_id !== this.user_id) {
                    const currentPos = this.editor.getPosition();
                    const oldContent = this.editor.getValue();
                    const newContent = message.content;
                    
                    isRemoteUpdate = true;
                    this.editor.setValue(newContent);
                    if (currentPos) this.editor.setPosition(currentPos);
                    isRemoteUpdate = false;

                    // Highlight remote changes using Monaco decorations
                    const oldLines = oldContent.split('\n');
                    const newLines = newContent.split('\n');
                    const changedLines = [];
                    
                    newLines.forEach((line, idx) => {
                        if (line !== oldLines[idx]) {
                            changedLines.push({
                                range: new monaco.Range(idx + 1, 1, idx + 1, 1),
                                options: {
                                    isWholeLine: true,
                                    className: 'remote-change-flash-anim'
                                }
                            });
                        }
                    });

                    if (changedLines.length > 0) {
                        const newDecorations = this.editor.createDecorationsCollection(changedLines);
                        setTimeout(() => {
                            newDecorations.clear();
                        }, 1500);
                    }
                }
            } else if (message.type === 'sync') {
                this.user_id = message.user_id;
                this.role = message.role;
                this.initializeWebRTC();

                if (message.language && this.onLanguageChange) {
                    this.onLanguageChange(message.language);
                }
                if (message.content && this.editor) {
                    isRemoteUpdate = true;
                    this.editor.setValue(message.content);
                    isRemoteUpdate = false;
                }
                if (message.connected_users) {
                    message.connected_users.forEach(u => {
                        this.roomUsers.set(u.user_id, {
                            role: u.role,
                            userId: u.user_id
                        });
                        if (u.role !== this.role) {
                            this.updateJoinedUser(u.role);
                        }
                    });
                }
                if (message.problem_title) this.updateProblemTitle(message.problem_title);
                if (message.problem_description) this.updateProblemDescription(message.problem_description);
                if (message.question_meta) this.updateQuestionMeta(message.question_meta);
                if (message.question_hints) this.updateQuestionHints(message.question_hints);
                if (message.question_snippets) this.updateQuestionSnippets(message.question_snippets);
            } else if (message.type === 'execution_output') {
                const result = message.content;
                const outputArea = document.getElementById('output');
                if (outputArea && message.user_id !== this.user_id) {
                    const runnerRole = message.role || (message.user_id === this.user_id ? "You" : "Peer");
                    const timestamp = new Date().toLocaleTimeString();
                    const header = `--- Run by ${runnerRole} at ${timestamp} ---\n`;
                    if (result.error) {
                        outputArea.value = `${header}Error:\n${result.stderr || result.message}`;
                    } else {
                        outputArea.value = `${header}${result.stdout}`;
                        if (result.stderr) {
                            outputArea.value += `\n--- Stderr ---\n${result.stderr}`;
                        }
                    }
                    this.showNotification(`Remote execution finished`, 'info');
                }
            }

            if (message.problem_title) this.updateProblemTitle(message.problem_title);
            if (message.problem_description) this.updateProblemDescription(message.problem_description);
            if (message.question_meta) this.updateQuestionMeta(message.question_meta);
            if (message.question_hints) this.updateQuestionHints(message.question_hints);
            if (message.question_snippets) this.updateQuestionSnippets(message.question_snippets);
        });

        this.wss.addEventListener('close', () => {
            this.showNotification('WebSocket connection closed.', 'error');
            if (this.webrtcHandler) {
                this.webrtcHandler.disconnect();
            }
        });

        // Handle editor changes
        this.setupEditorListeners();

        this.createAudioControls();
    }

    setupEditorListeners() {
        if (!this.editor) return;
        this.editor.onDidChangeModelContent((e) => {
            if (!isRemoteUpdate) {
                const content = this.editor.getValue();
                this.#sendCode(content);
            }
        });
    }

    setupQuestionObserver() {
        this.questionBlock = document.getElementById('questionBlock');
        if (this.questionBlock) {
            if (this.observer) this.observer.disconnect();
            this.observer = new MutationObserver((mutations) => {
                if (this.editor) {
                    this.#sendCode(this.editor.getValue());
                }
            });
            this.observer.observe(this.questionBlock, {
                childList: true,
                subtree: true,
                characterData: true
            });
        }
    }

    updateEditor(newEditor) {
        this.editor = newEditor;
        this.setupEditorListeners();
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
        setTimeout(() => {
            notification.style.opacity = '1';
            notification.style.transform = 'translateX(0)';
        }, 10);
        setTimeout(() => {
            notification.style.opacity = '0';
            notification.style.transform = 'translateX(100%)';
            setTimeout(() => {
                if (this.notificationContainer.contains(notification)) {
                    this.notificationContainer.removeChild(notification);
                }
            }, 300);
        }, 3000);
    }

    sendLanguageChange(language) {
        if (this.wss.readyState === WebSocket.OPEN) {
            const message = {
                type: 'language_change',
                room_id: this.roomId,
                user_id: this.user_id,
                language: language
            };
            this.wss.send(JSON.stringify(message));
        }
    }

    #sendCode(content) {
        if (this.wss.readyState === WebSocket.OPEN && !isRemoteUpdate) {
            const message = {
                type: 'code',
                room_id: this.roomId,
                content: content,
                user_id: this.user_id,
                problem_title: this.getProblemTitle(),
                problem_description: this.getProblemDescription(),
                question_meta: this.getQuestionMeta(),
                question_hints: this.getQuestionHints(),
                question_snippets: this.getQuestionSnippets(),
            };
            this.wss.send(JSON.stringify(message));
        }
    }

    getProblemTitle() {
        const el = document.querySelector("#questionTitle");
        return el ? el.innerHTML : "";
    }

    updateProblemTitle(content) {
        const el = document.querySelector("#questionTitle");
        if (el && el.innerHTML !== content) {
            this.withObserverPaused(() => el.innerHTML = content);
        }
    }

    getProblemDescription() {
        const el = document.querySelector("#problemDescription");
        return el ? el.innerHTML : "";
    }

    updateProblemDescription(content) {
        const el = document.querySelector("#problemDescription");
        if (el && el.innerHTML !== content) {
            this.withObserverPaused(() => el.innerHTML = content);
        }
    }

    getQuestionMeta() {
        const el = document.querySelector("#questionMeta");
        return el ? el.innerHTML : "";
    }

    updateQuestionMeta(content) {
        const el = document.querySelector("#questionMeta");
        if (el && el.innerHTML !== content) {
            this.withObserverPaused(() => el.innerHTML = content);
        }
    }

    getQuestionHints() {
        const el = document.querySelector("#questionHintsSection");
        return el ? el.innerHTML : "";
    }

    updateQuestionHints(content) {
        const el = document.querySelector("#questionHintsSection");
        if (el && el.innerHTML !== content) {
            this.withObserverPaused(() => el.innerHTML = content);
        }
    }

    getQuestionSnippets() {
        const el = document.querySelector("#codeSnippetCode");
        return el ? el.innerHTML : "";
    }

    updateQuestionSnippets(content) {
        const el = document.querySelector("#codeSnippetCode");
        if (el && el.innerHTML !== content) {
            this.withObserverPaused(() => el.innerHTML = content);
        }
    }

    withObserverPaused(callback) {
        const wasRemoteUpdate = isRemoteUpdate;
        isRemoteUpdate = true;
        try {
            callback();
        } finally {
            setTimeout(() => {
                isRemoteUpdate = wasRemoteUpdate;
            }, 0);
        }
    }

    updateJoinedUser(oppositeRole) {
        if (this.joinedUserElement) {
            if (oppositeRole) {
                const myRole = this.role;
                const displayRole = myRole === 'Author' ? 'Collaborator' : 'Author';
                this.joinedUserElement.textContent = `@${displayRole}`;
            } else {
                this.joinedUserElement.textContent = 'None';
            }
        }
    }

    createAudioControls() {
        const controlsContainer = document.getElementById('callControls');
        if (!controlsContainer) return;
        controlsContainer.innerHTML = '';
        const statusText = document.createElement('div');
        statusText.id = 'callStatus';
        statusText.className = 'hidden text-xs font-medium text-red-600 dark:text-red-400';
        statusText.textContent = '';
        const callButton = document.createElement('button');
        callButton.id = 'callButton';
        callButton.className = 'px-3 py-1 bg-green-500 text-white rounded-lg hover:bg-green-600 shadow-lg transition-colors text-xs';
        callButton.textContent = 'Start Call';
        callButton.onclick = () => this.handleCallButtonClick();
        controlsContainer.appendChild(statusText);
        controlsContainer.appendChild(callButton);
    }

    handleCallButtonClick() {
        if (!this.localCallReady && !this.remoteCallReady) {
            this.startCall();
        }
    }

    updateCallUI(state) {
        const btn = document.getElementById('callButton');
        const status = document.getElementById('callStatus');
        if (!btn || !status) return;
        switch (state) {
            case 'idle':
                this.stopCallTimer();
                btn.textContent = 'Start Call';
                btn.className = 'px-3 py-1 bg-green-500 text-white rounded-lg hover:bg-green-600 shadow-lg transition-colors text-xs';
                btn.disabled = false;
                btn.classList.remove('hidden');
                status.textContent = '';
                status.classList.add('hidden');
                break;
            case 'waiting':
                this.stopCallTimer();
                btn.disabled = true;
                btn.classList.add('hidden');
                status.className = 'text-xs font-medium text-red-600 dark:text-red-400';
                status.textContent = 'Waiting for peer...';
                status.classList.remove('hidden');
                break;
            case 'connected':
                btn.disabled = true;
                btn.classList.add('hidden');
                status.className = 'text-xs font-bold text-red-600 dark:text-red-400';
                status.classList.remove('hidden');
                this.startCallTimer();
                break;
        }
    }

    startCallTimer() {
        if (this.callTimerInterval) return;
        const startTime = Date.now();
        const status = document.getElementById('callStatus');
        const update = () => {
            const diff = Math.floor((Date.now() - startTime) / 1000);
            const mins = Math.floor(diff / 60).toString().padStart(2, '0');
            const secs = (diff % 60).toString().padStart(2, '0');
            if (status) status.textContent = `● In Call • ${mins}:${secs}`;
        };
        update();
        this.callTimerInterval = setInterval(update, 1000);
    }

    stopCallTimer() {
        if (this.callTimerInterval) {
            clearInterval(this.callTimerInterval);
            this.callTimerInterval = null;
        }
    }

    async initializeWebRTC() {
        if (!this.webrtcHandler) {
            this.webrtcHandler = new WebRTCHandler(this.roomId, this.user_id, this.wss);
            try {
                await this.webrtcHandler.initialize();
                console.log('WebRTC initialized successfully');
            } catch (error) {
                console.error('Failed to initialize WebRTC:', error);
                this.showNotification('Failed to initialize audio call', 'error');
            }
        }
    }

    async startCall() {
        if (!this.user_id) {
            this.showNotification('Waiting for user ID...', 'warning');
            return;
        }
        this.localCallReady = true;
        this.updateCallUI('waiting');
        if (this.wss.readyState === WebSocket.OPEN) {
            this.wss.send(JSON.stringify({
                type: 'call_ready',
                room_id: this.roomId,
                user_id: this.user_id,
                role: this.role
            }));
        }
        if (!this.checkAutoConnect()) {
            this.showNotification('Waiting for peer to join call...', 'info');
        }
    }

    checkAutoConnect() {
        if (this.localCallReady && this.remoteCallReady) {
            this.updateCallUI('connected');
            this.initiateWebRTCCall();
            return true;
        }
        return false;
    }

    async initiateWebRTCCall() {
        if (!this.webrtcHandler) {
            await this.initializeWebRTC();
        }
        const oppositeRole = this.role === 'Author' ? 'Collaborator' : 'Author';
        const targetUserId = this.getOppositeUserId(oppositeRole);
        if (targetUserId) {
            try {
                await this.webrtcHandler.initiateCall(targetUserId);
                this.showNotification('Connecting audio...', 'success');
            } catch (error) {
                console.error('Failed to start call:', error);
                this.showNotification('Failed to start call', 'error');
                this.updateCallUI('idle');
                this.localCallReady = false;
            }
        } else {
            this.showNotification('Peer not found in room yet', 'warning');
        }
    }

    endCall(notifyPeer = true) {
        if (notifyPeer && this.localCallReady && this.wss.readyState === WebSocket.OPEN) {
            this.wss.send(JSON.stringify({
                type: 'call_ended',
                room_id: this.roomId,
                user_id: this.user_id,
                role: this.role
            }));
        }
        this.localCallReady = false;
        this.remoteCallReady = false;
        this.updateCallUI('idle');
        if (this.webrtcHandler) {
            this.webrtcHandler.disconnect();
            if (notifyPeer) this.showNotification('Call ended', 'info');
        }
    }

    getOppositeUserId(oppositeRole) {
        for (const [userId, userData] of this.roomUsers.entries()) {
            if (userData.role === oppositeRole) {
                return userId;
            }
        }
        return null;
    }
}

function runWebsocketProcess() {
    const roomId = document.querySelector("span#roomId").textContent.trim();
    const languageSelector = document.querySelector('#programmingLanguages');

    const codeCache = new Map();
    let lastLanguage = 'default';

    // Initialize Monaco
    codeboxInit(undefined, undefined, (editor) => {
        // Callback for remote language changes
        const onRemoteLanguageChange = (newLanguage) => {
            const normalizedLanguage = newLanguage.toLowerCase();
            if (editor) {
                codeCache.set(lastLanguage, editor.getValue());
            }

            if (languageSelector.value.toLowerCase() !== normalizedLanguage) {
                for (let i = 0; i < languageSelector.options.length; i++) {
                    if (languageSelector.options[i].value.toLowerCase() === normalizedLanguage) {
                        languageSelector.selectedIndex = i;
                        break;
                    }
                }
                lastLanguage = normalizedLanguage;
                codeboxInit(normalizedLanguage, codeCache.get(normalizedLanguage), (newEditor) => {
                    wss.updateEditor(newEditor);
                });
            }
        };

        // Initialize WebSocket connection
        let wss = new WebSocketClient(roomId, editor, onRemoteLanguageChange);
        window.wssClient = wss;

        // Listen for HTMX swaps
        document.body.addEventListener('htmx:afterSwap', (event) => {
            if (event.target.id === 'questionBlock' || event.detail.target.id === 'questionBlock') {
                codeCache.clear();
                const currentLang = languageSelector.value.toLowerCase();
                codeboxInit(currentLang, undefined, (newEditor) => {
                    lastLanguage = currentLang;
                    wss.updateEditor(newEditor);
                });
            }
        });

        // Language change listener
        languageSelector.addEventListener('change', (event) => {
            const selectedLanguage = event.target.value.toLowerCase();
            if (editor) {
                codeCache.set(lastLanguage, editor.getValue());
            }
            wss.sendLanguageChange(selectedLanguage);
            lastLanguage = selectedLanguage;
            codeboxInit(selectedLanguage, codeCache.get(selectedLanguage), (newEditor) => {
                wss.updateEditor(newEditor);
                if (wss.wss.readyState === WebSocket.OPEN) {
                    const message = {
                        type: 'code',
                        room_id: roomId,
                        content: newEditor.getValue(),
                        user_id: wss.user_id,
                        problem_title: wss.getProblemTitle(),
                        problem_description: wss.getProblemDescription(),
                        question_meta: wss.getQuestionMeta(),
                        question_hints: wss.getQuestionHints(),
                        question_snippets: wss.getQuestionSnippets(),
                    };
                    wss.wss.send(JSON.stringify(message));
                }
            });
        });
    });
}

runWebsocketProcess();
