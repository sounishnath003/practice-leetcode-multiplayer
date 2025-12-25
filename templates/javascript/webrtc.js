"use strict";

class WebRTCHandler {
    constructor(roomId, userId, ws) {
        this.roomId = roomId;
        this.userId = userId;
        this.ws = ws;
        this.peerConnections = {};
        this.localStream = null;
        this.mediaConstraints = {
            audio: true,
            video: false
        };
    }

    async initialize() {
        try {
            this.localStream = await navigator.mediaDevices.getUserMedia(this.mediaConstraints);
            console.log("Local audio stream initialized");
        } catch (error) {
            console.error('Error accessing media devices:', error);
            throw error; // Propagate error so caller knows init failed
        }
    }

    handleMessage(message) {
        switch (message.type) {
            case 'offer':
                this.handleOffer(message);
                break;
            case 'answer':
                this.handleAnswer(message);
                break;
            case 'ice-candidate':
                this.handleIceCandidate(message);
                break;
        }
    }

    async createPeerConnection(targetUserId) {
        if (this.peerConnections[targetUserId]) {
            return this.peerConnections[targetUserId];
        }

        // Ensure local stream is ready before creating connection
        if (!this.localStream) {
            try {
                console.log("Local stream not ready, initializing...");
                await this.initialize();
            } catch (err) {
                console.warn("Failed to initialize local stream on demand:", err);
                // Continue without local tracks if failed (listen-only?)
            }
        }

        const configuration = {
            iceServers: [
                { urls: 'stun:stun.l.google.com:19302' },
                { urls: 'stun:stun1.l.google.com:19302' }
            ]
        };

        const peerConnection = new RTCPeerConnection(configuration);
        this.peerConnections[targetUserId] = peerConnection;

        // Add local stream tracks to the connection
        if (this.localStream) {
            this.localStream.getTracks().forEach(track => {
                peerConnection.addTrack(track, this.localStream);
            });
        }

        // Handle ICE candidates
        peerConnection.onicecandidate = (event) => {
            if (event.candidate) {
                this.ws.send(JSON.stringify({
                    type: 'ice-candidate',
                    room_id: this.roomId,
                    target_user_id: targetUserId,
                    ice_candidate: event.candidate,
                    user_id: this.userId
                }));
            }
        };

        // Handle incoming audio stream
        peerConnection.ontrack = (event) => {
            console.log("Received remote track from:", targetUserId);
            const audioElement = document.createElement('audio');
            audioElement.srcObject = event.streams[0];
            audioElement.autoplay = true;
            audioElement.controls = false; // Hide controls
            audioElement.muted = false;
            
            audioElement.onplay = () => console.log("Remote audio started playing for:", targetUserId);
            
            // Ensure audio plays even if policy restricts
            audioElement.play().catch(e => console.error("Auto-play failed:", e));
            
            document.body.appendChild(audioElement);
        };

        peerConnection.onconnectionstatechange = () => {
            console.log(`Connection state with ${targetUserId}: ${peerConnection.connectionState}`);
            if (peerConnection.connectionState === 'disconnected' || peerConnection.connectionState === 'failed') {
                this.closeConnection(targetUserId);
            }
        };

        return peerConnection;
    }

    async initiateCall(targetUserId) {
        console.log("Initiating call to:", targetUserId);
        try {
            const peerConnection = await this.createPeerConnection(targetUserId);
            
            if (peerConnection.signalingState !== 'stable') {
                console.warn("Connection is negotiating. Skipping new offer.");
                return;
            }

            const offer = await peerConnection.createOffer();
            await peerConnection.setLocalDescription(offer);

            this.ws.send(JSON.stringify({
                type: 'offer',
                room_id: this.roomId,
                target_user_id: targetUserId,
                sdp: offer,
                user_id: this.userId
            }));
        } catch (err) {
            console.error("Error initiating call:", err);
        }
    }

    async handleOffer(message) {
        console.log("Handling offer from:", message.user_id);
        try {
            const peerConnection = await this.createPeerConnection(message.user_id);

            // Glare handling: If we are already negotiating...
            if (peerConnection.signalingState !== 'stable') {
                console.log(`Glare detected. Local state: ${peerConnection.signalingState}. Remote: ${message.user_id} vs Local: ${this.userId}`);
                // Tie-breaker: If remote ID is smaller, we win (ignore their offer).
                // If remote ID is larger, we yield (rollback and accept their offer).
                if (message.user_id < this.userId) {
                    console.log("Ignoring collision offer (we have priority).");
                    return;
                } else {
                    console.log("Yielding to collision offer (remote has priority). Rolling back.");
                    try {
                        await peerConnection.setLocalDescription({ type: "rollback" });
                    } catch (e) {
                        console.warn("Rollback failed (maybe not supported), resetting connection.", e);
                        // Hard reset if rollback fails
                        this.closeConnection(message.user_id);
                        // Recursively handle offer with fresh connection
                        return this.handleOffer(message);
                    }
                }
            }

            await peerConnection.setRemoteDescription(new RTCSessionDescription(message.sdp));

            const answer = await peerConnection.createAnswer();
            await peerConnection.setLocalDescription(answer);

            this.ws.send(JSON.stringify({
                type: 'answer',
                room_id: this.roomId,
                target_user_id: message.user_id,
                sdp: answer,
                user_id: this.userId
            }));
        } catch (err) {
            console.error("Error handling offer:", err);
        }
    }

    async handleAnswer(message) {
        console.log("Handling answer from:", message.user_id);
        const peerConnection = this.peerConnections[message.user_id];
        if (peerConnection) {
            if (peerConnection.signalingState === 'stable') {
                console.log("Connection is already stable. Ignoring answer (likely duplicate or late).");
                return;
            }
            try {
                await peerConnection.setRemoteDescription(new RTCSessionDescription(message.sdp));
            } catch (err) {
                console.error("Error setting remote description:", err);
            }
        }
    }

    async handleIceCandidate(message) {
        const peerConnection = this.peerConnections[message.user_id];
        if (peerConnection) {
            try {
                await peerConnection.addIceCandidate(new RTCIceCandidate(message.ice_candidate));
            } catch (err) {
                console.error("Error adding ICE candidate:", err);
            }
        }
    }

    closeConnection(targetUserId) {
        const pc = this.peerConnections[targetUserId];
        if (pc) {
            pc.close();
            delete this.peerConnections[targetUserId];
        }
        // Also remove audio elements associated with this user if we tracked them
    }

    disconnect() {
        console.log("Disconnecting WebRTC...");
        // Stop all tracks in local stream
        if (this.localStream) {
            this.localStream.getTracks().forEach(track => track.stop());
            this.localStream = null;
        }

        // Close all peer connections
        Object.keys(this.peerConnections).forEach(userId => {
            this.closeConnection(userId);
        });
        
        // Cleanup audio elements (rough cleanup)
        document.querySelectorAll('audio').forEach(el => el.remove());
    }
}