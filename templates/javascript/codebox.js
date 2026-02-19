"use strict";

// This script handles the "Run Code" button logic.
console.log("Codebox execution script loaded.");

function setIoPanelOpen(isOpen) {
    const panel = document.getElementById("io-panel");
    const btn = document.getElementById("io-toggle-btn");
    if (!panel || !btn) return;

    panel.classList.toggle("hidden", !isOpen);
    btn.textContent = isOpen ? "Hide" : "Show";
    try {
        localStorage.setItem("ioPanelOpen", isOpen ? "1" : "0");
    } catch (_) {
        // ignore
    }
}

function getIoPanelInitialState() {
    try {
        return localStorage.getItem("ioPanelOpen") === "1";
    } catch (_) {
        return false;
    }
}

function setupIoToggle() {
    const ioToggleBtn = document.getElementById("io-toggle-btn");
    if (!ioToggleBtn) return;

    // De-dupe listeners (HTMX swaps can re-run this script)
    const newBtn = ioToggleBtn.cloneNode(true);
    ioToggleBtn.parentNode.replaceChild(newBtn, ioToggleBtn);

    // Apply initial state (default closed)
    setIoPanelOpen(getIoPanelInitialState());

    newBtn.addEventListener("click", () => {
        const panel = document.getElementById("io-panel");
        if (!panel) return;
        const isOpen = !panel.classList.contains("hidden");
        setIoPanelOpen(!isOpen);
    });
}

function setupRunCode() {
    const runCodeBtn = document.getElementById('run-code-btn');
    
    // Remove old listener if any (by cloning) - simple way to ensure no duplicates
    // But easier is just to attach to body for delegation or check if attached.
    // Let's stick to direct attachment but be careful.
    
    if (runCodeBtn) {
        // Cloning the node removes all event listeners
        const newBtn = runCodeBtn.cloneNode(true);
        runCodeBtn.parentNode.replaceChild(newBtn, runCodeBtn);
        
        newBtn.addEventListener('click', async () => {
            console.log("Run Code button clicked.");
            setIoPanelOpen(true);

            // Access the global currentEditor instance created in websocketClient.js
            // or try to find the CodeMirror instance on the textarea
            let editorInstance = currentEditor;
            if (!editorInstance) {
                const cmElement = document.querySelector('.CodeMirror');
                if (cmElement && cmElement.CodeMirror) {
                    editorInstance = cmElement.CodeMirror;
                }
            }

            if (!editorInstance) {
                console.error("CodeMirror editor instance not found.");
                alert("Editor not initialized. Please refresh the page.");
                return;
            }

            const languageSelect = document.getElementById('programmingLanguages');
            const testcasesArea = document.getElementById('testcases');
            const outputArea = document.getElementById('output');

            let code = editorInstance.getValue();
            const normalizedCode = code.replace(/\t/g, '    ');
            if (normalizedCode !== code) {
                code = normalizedCode;
            }
            const language = languageSelect ? languageSelect.value : 'python'; 
            const input = testcasesArea ? testcasesArea.value : '';
            const roomId = document.querySelector("span#roomId")?.textContent.trim() || "";
            const userId = window.wssClient?.user_id || "";

            console.log(`Executing ${language} code...`);

            if (!code.trim()) {
                if (outputArea) outputArea.value = "Please enter some code to run.";
                return;
            }

            // Set loading state
            const originalBtnText = newBtn.innerHTML;
            newBtn.disabled = true;
            newBtn.innerHTML = `<span>Running...</span>`;
            if (outputArea) outputArea.value = "Executing on Cloud Runner...";

            try {
                const response = await fetch('/api/execute-code', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        language: language,
                        code: code,
                        stdin: input,
                        room_id: roomId,
                        user_id: userId
                    })
                });

                const result = await response.json();
                console.log("Execution result:", result);

                if (outputArea) {
                    if (response.ok) {
                        if (result.error) {
                            outputArea.value = `Error:\n${result.stderr || result.message}`;
                        } else {
                            outputArea.value = result.stdout;
                            if (result.stderr) {
                                outputArea.value += `\n--- Stderr ---\n${result.stderr}`;
                            }
                        }
                    } else {
                        outputArea.value = `Server Error: ${result.error || response.statusText}`;
                    }
                }

            } catch (error) {
                console.error("Run code error:", error);
                if (outputArea) outputArea.value = `Request Failed: ${error.message}`;
            } finally {
                newBtn.disabled = false;
                newBtn.innerHTML = originalBtnText;
            }
        });
        console.log("Run Code listener attached.");
    } else {
        console.warn("Run Code button not found in DOM.");
    }
}

// Run setup immediately
setupIoToggle();
setupRunCode();

// Also observe for DOM changes (in case of HTMX swaps)
const observer = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
        if (mutation.addedNodes.length) {
            if (document.getElementById('run-code-btn')) {
               // Debounce or just check if we need to re-attach?
               // Since we clone-replace, running it again is safe-ish but wasteful.
               // Better to rely on the script being re-executed if the script tag is inside the swapped content,
               // but it's not (it's in home.html which is likely the swap target or container).
            }
        }
    }
});

// If HTMX is used, listen for afterSwap to re-wire controls after template swaps
document.body.addEventListener('htmx:afterSwap', () => {
    console.log("HTMX swap detected, re-setting up CodeBox controls.");
    setupIoToggle();
    setupRunCode();
});
