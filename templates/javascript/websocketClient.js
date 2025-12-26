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

function codeboxInit(language, cachedContent) {
    const codeboxElement = document.querySelector('#codebox');
    // Destroy the existing editor instance if it exists
    if (currentEditor) {
        currentEditor.toTextArea(); // Restore the original <textarea>
        currentEditor = null;
    }

    // Normalize language
    language = language?.toLowerCase();
    if (language === 'c++') language = 'cpp';

    // Set the boilerplate code for the selected language
    let boilerplate = languageBoilerplate[language] || languageBoilerplate.default;

    // Check the Selected Language boiler plate
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

    // Use cached content if available, otherwise use boilerplate
    codeboxElement.value = cachedContent !== undefined ? cachedContent : boilerplate;

    // If the language is Java, use the "clike" mode
    if (language === 'java') language = 'text/x-java';
    if (language === 'cpp') language = 'text/x-c++src';

    // Initialize the CodeMirror editor
    currentEditor = CodeMirror.fromTextArea(codeboxElement, {
        lineNumbers: true,
        mode: { name: language ?? "text/x-java" },
        theme: "eclipse",
        font: "Fira Codee, Consolas, Monaco, 'Lucida Console', 'Liberation Mono', 'DejaVu Sans Mono', 'Bitstream Vera Sans Mono', 'Courier New', monospace",
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
        // extraKeys: { "Alt-F": "findPersistent", "Cmd-/": 'toggleComment' },
        extraKeys: { 'Ctrl-/': 'toggleComment', 'Cmd-/': 'toggleComment' },
        hintOptions: {
            completeSingle: false, // Prevent auto-selecting the first suggestion
        },
    });

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
            // Re-setup observer regardless of target, just to be safe if questionBlock was affected
            this.setupQuestionObserver();
            // Trigger sync to share the new question details
            this.#sendCode(this.editor.getValue());
        });

        this.wss.addEventListener('open', (e) => {
            console.log('WebSocket connection opened:', e);
        });

        this.wss.addEventListener('message', (e) => {
            const message = JSON.parse(e.data);
            console.log('Received message:', message);

            // Delegate WebRTC messages to handler
            if (this.webrtcHandler && ['offer', 'answer', 'ice-candidate'].includes(message.type)) {
                this.webrtcHandler.handleMessage(message);
            }

            if (message.type === 'join') {
                // Add user to room users map
                this.roomUsers.set(message.user_id, {
                    role: message.role,
                    userId: message.user_id
                });
                this.showNotification(`${message.role} joined the room`, 'success');
                this.updateJoinedUser(message.role);
                // Initialize WebRTC when a new user joins
                this.initializeWebRTC();
            } else if (message.type === 'leave') {
                // Remove user from room users map
                this.roomUsers.delete(message.user_id);
                this.showNotification(`${message.role} left the room`, 'warning');
                this.updateJoinedUser(null);
                // Disconnect WebRTC when user leaves
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
                this.endCall(false); // End local call without notifying peer back
                this.showNotification(`${message.role} ended the call`, 'info');
            } else if (message.type === 'code') {
                // Update editor content without triggering change event
                const currentCursor = this.editor.getCursor();
                this.editor.setValue(message.content);
                this.editor.setCursor(currentCursor);
            } else if (message.type === 'sync') {
                // Set identity from sync message
                this.user_id = message.user_id;
                this.role = message.role;

                // Sync initial state
                this.initializeWebRTC(); // Ensure WebRTC is ready for late joiners
                
                if (message.language && this.onLanguageChange) {
                    this.onLanguageChange(message.language);
                }
                if (message.content) {
                    this.editor.setValue(message.content);
                }
                if (message.connected_users) {
                    message.connected_users.forEach(u => {
                        this.roomUsers.set(u.user_id, {
                            role: u.role,
                            userId: u.user_id
                        });
                        // Update UI for each (or just once at end)
                        // Assuming max 2 users, so opposite role is singular
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
            }

            // Granular updates for question details
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
        this.editor.on('change', (cm, change) => {
            if (change.origin !== 'setValue') {
                const content = cm.getValue();
                this.#sendCode(content);
            }
        });

        // Create audio controls
        this.createAudioControls();
    }

    setupQuestionObserver() {
        this.questionBlock = document.getElementById('questionBlock');
        if (this.questionBlock) {
            // Disconnect existing observer if any
            if (this.observer) this.observer.disconnect();

            this.observer = new MutationObserver((mutations) => {
                // Check if the mutation is relevant to avoid infinite loops
                // (Though with granular updates, loops are less likely if we check content equality, 
                // but simplicity first: just send if it's a DOM change we didn't cause? 
                // Actually, since we update innerHTML of children, that triggers observer.
                // We need to temporarily disconnect observer during updates or use a flag.)
                this.#sendCode(this.editor.getValue());
            });

            this.observer.observe(this.questionBlock, {
                childList: true,
                subtree: true,
                characterData: true
            });
        }
    }

    // Flag to prevent observer loops during remote updates
    isRemoteUpdate = false;

    updateEditor(newEditor) {
        this.editor = newEditor;
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
        if (this.wss.readyState === WebSocket.OPEN && !this.isRemoteUpdate) {
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

    // Helper to pause observer during updates
    withObserverPaused(callback) {
        this.isRemoteUpdate = true;
        try {
            callback();
        } finally {
            setTimeout(() => {
                this.isRemoteUpdate = false;
            }, 0);
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

    createAudioControls() {
        const controlsContainer = document.createElement('div');
        controlsContainer.className = 'fixed bottom-4 right-4 z-50 flex flex-col gap-2 items-end';

        const statusText = document.createElement('div');
        statusText.id = 'callStatus';
        statusText.className = 'text-xs text-gray-500 font-medium hidden dark:text-gray-400';
        statusText.textContent = '';

        const callButton = document.createElement('button');
        callButton.id = 'callButton';
        callButton.className = 'px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 shadow-lg transition-colors';
        callButton.textContent = 'Start Call';
        callButton.onclick = () => this.handleCallButtonClick();

        controlsContainer.appendChild(statusText);
        controlsContainer.appendChild(callButton);
        document.body.appendChild(controlsContainer);
    }

    handleCallButtonClick() {
        if (this.localCallReady && this.remoteCallReady) {
            this.endCall(); // If connected, button ends call
        } else if (this.localCallReady) {
            this.endCall(); // If waiting, cancel
        } else {
            this.startCall(); // If idle, start
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
                btn.className = 'px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 shadow-lg transition-colors';
                status.textContent = '';
                status.classList.add('hidden');
                break;
            case 'waiting':
                this.stopCallTimer();
                btn.textContent = 'Cancel Call';
                btn.className = 'px-4 py-2 bg-yellow-500 text-white rounded-lg hover:bg-yellow-600 shadow-lg transition-colors';
                status.textContent = 'Waiting for peer...';
                status.classList.remove('hidden');
                break;
            case 'connected':
                btn.textContent = 'End Call';
                btn.className = 'px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 shadow-lg transition-colors animate-pulse';
                status.className = 'text-xs text-red-500 font-bold dark:text-red-400';
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
        
        update(); // Initial update
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
        
        // Notify peer we are ready
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

        // Get the opposite role's user ID
        const oppositeRole = this.role === 'Author' ? 'Collaborator' : 'Author';
        const targetUserId = this.getOppositeUserId(oppositeRole);

        console.log('Starting call with:', {
            myUserId: this.user_id,
            myRole: this.role,
            oppositeRole,
            targetUserId
        });

        if (targetUserId) {
            try {
                // Determine who initiates based on ID to avoid glare (though handled in webrtc.js too)
                // Or just both try, and glare logic handles it. 
                // Since both are 'ready', auto-connecting is fine.
                await this.webrtcHandler.initiateCall(targetUserId);
                this.showNotification('Connecting audio...', 'success');
            } catch (error) {
                console.error('Failed to start call:', error);
                this.showNotification('Failed to start call', 'error');
                this.updateCallUI('idle'); // Reset on error
                this.localCallReady = false;
            }
        } else {
            // Even if we can't find them in the map yet, they might be there.
            // But getOppositeUserId relies on roomUsers map.
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
        console.log('Current room users:', Array.from(this.roomUsers.entries()));
        console.log('Looking for role:', oppositeRole);

        // Find the user with the opposite role
        for (const [userId, userData] of this.roomUsers.entries()) {
            if (userData.role === oppositeRole) {
                console.log('Found opposite user:', userId);
                return userId;
            }
        }

        console.log('No opposite user found');
        return null;
    }
}

function runWebsocketProcess() {
    const roomId = document.querySelector("span#roomId").textContent.trim();
    const languageSelector = document.querySelector('#programmingLanguages');
    
    // Client-side cache for code per language
    const codeCache = new Map();
    let lastLanguage = 'default';

    // Initialize with the default language
    let codeEditor = codeboxInit(); 

    // Callback for remote language changes
    const onRemoteLanguageChange = (newLanguage) => {
        const normalizedLanguage = newLanguage.toLowerCase();
        
        // Save current code before switching
        if (codeEditor) {
            codeCache.set(lastLanguage, codeEditor.getValue());
        }

        if (languageSelector.value.toLowerCase() !== normalizedLanguage) {
            // Find and select the matching option case-insensitively
            for (let i = 0; i < languageSelector.options.length; i++) {
                if (languageSelector.options[i].value.toLowerCase() === normalizedLanguage) {
                    languageSelector.selectedIndex = i;
                    break;
                }
            }
            
            // Update tracking and init editor with cached code
            lastLanguage = normalizedLanguage;
            codeEditor = codeboxInit(normalizedLanguage, codeCache.get(normalizedLanguage));
            wss.updateEditor(codeEditor);
        }
    };

    // Initialize WebSocket connection
    let wss = new WebSocketClient(roomId, codeEditor, onRemoteLanguageChange);

    // Listen for HTMX swaps to re-initialize editor with new question boilerplate
    document.body.addEventListener('htmx:afterSwap', (event) => {
        // Only trigger if the question block was swapped
        if (event.target.id === 'questionBlock' || event.detail.target.id === 'questionBlock') {
            console.log("Question swapped, refreshing editor boilerplate...");
            
            // Clear cache as we have a new question
            codeCache.clear();
            
            // Re-initialize editor with current language to pull new boilerplate from DOM
            const currentLang = languageSelector.value.toLowerCase();
            codeEditor = codeboxInit(currentLang);
            
            // Update tracking
            lastLanguage = currentLang;
            
            // Update WebSocket client reference
            wss.updateEditor(codeEditor);
        }
    });

    // Reload the code editor and WebSocket connection when the programming language changes
    languageSelector.addEventListener('change', (event) => {
        const selectedLanguage = event.target.value.toLowerCase();

        // Save current code before switching
        if (codeEditor) {
            codeCache.set(lastLanguage, codeEditor.getValue());
        }

        // Notify peers about language change
        wss.sendLanguageChange(selectedLanguage);

        // Update tracking and reinitialize the editor with cached code or boilerplate
        lastLanguage = selectedLanguage;
        codeEditor = codeboxInit(selectedLanguage, codeCache.get(selectedLanguage));

        // Update the editor reference in the WebSocket client
        wss.updateEditor(codeEditor);

        // Send a sync message to update the code in the room
        if (wss.wss.readyState === WebSocket.OPEN) {
            const message = {
                type: 'code',
                room_id: roomId,
                content: codeEditor.getValue(),
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
}

runWebsocketProcess();
