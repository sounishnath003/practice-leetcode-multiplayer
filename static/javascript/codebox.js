// Initialize the WebSocket client when the page loads
document.addEventListener('DOMContentLoaded', () => {
    // Get the room ID from the page
    const roomIdElement = document.getElementById('roomId');
    if (roomIdElement) {
        const roomId = roomIdElement.textContent.trim();
        if (roomId && roomId !== 'None') {
            // Create a new instance of CodeEditorWebSocket
            const wsClient = new CodeEditorWebSocket();

            // Initialize the connection with the room ID
            wsClient.init(roomId);

            // Store the instance globally for access from other parts of the code
            window.wsClient = wsClient;
        }
    }
});

// Handle code execution
document.getElementById('runCode').addEventListener('click', async () => {
    const code = document.getElementById('codeEditor').value;
    // Send to your backend API
    // Handle the response
});

// Handle code clearing
document.getElementById('clearCode').addEventListener('click', () => {
    document.getElementById('codeEditor').value = '';
    document.getElementById('outputArea').textContent = '';
});

// When you need to disconnect
wsClient.disconnect();


// When a user creates a new room
function createNewRoom() {
    const roomId = generateRoomId(); // Your room ID generation logic
    const wsClient = new CodeEditorWebSocket();
    wsClient.init(roomId);
    window.wsClient = wsClient;
}

// When a user joins an existing room
function joinRoom(roomId) {
    const wsClient = new CodeEditorWebSocket();
    wsClient.init(roomId);
    window.wsClient = wsClient;
}

// When the page is unloaded
window.addEventListener('beforeunload', () => {
    if (window.wsClient) {
        window.wsClient.disconnect();
    }
}); 