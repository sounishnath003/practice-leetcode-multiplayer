gtclass WebRTCHandler {
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
            this.setupWebSocketHandlers();
        } catch (error) {
            console.error('Error accessing media devices:', error);
        }
    }

    setupWebSocketHandlers() {
        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
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
        };
    }

    async createPeerConnection(targetUserId) {
        const configuration = {
            iceServers: [
                { urls: 'stun:stun.l.google.com:19302' }
            ]
        };

        const peerConnection = new RTCPeerConnection(configuration);
        this.peerConnections[targetUserId] = peerConnection;

        // Add local stream
        this.localStream.getTracks().forEach(track => {
            peerConnection.addTrack(track, this.localStream);
        });

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

        // Handle incoming audio
        peerConnection.ontrack = (event) => {
            const audioElement = document.createElement('audio');
            audioElement.srcObject = event.streams[0];
            audioElement.autoplay = true;
            document.body.appendChild(audioElement);
        };

        return peerConnection;
    }

    async initiateCall(targetUserId) {
        const peerConnection = await this.createPeerConnection(targetUserId);
        const offer = await peerConnection.createOffer();
        await peerConnection.setLocalDescription(offer);

        this.ws.send(JSON.stringify({
            type: 'offer',
            room_id: this.roomId,
            target_user_id: targetUserId,
            sdp: offer,
            user_id: this.userId
        }));
    }

    async handleOffer(message) {
        const peerConnection = await this.createPeerConnection(message.user_id);
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
    }

    async handleAnswer(message) {
        const peerConnection = this.peerConnections[message.user_id];
        if (peerConnection) {
            await peerConnection.setRemoteDescription(new RTCSessionDescription(message.sdp));
        }
    }

    async handleIceCandidate(message) {
        const peerConnection = this.peerConnections[message.user_id];
        if (peerConnection) {
            await peerConnection.addIceCandidate(new RTCIceCandidate(message.ice_candidate));
        }
    }

    disconnect() {
        // Stop all tracks in local stream
        if (this.localStream) {
            this.localStream.getTracks().forEach(track => track.stop());
        }

        // Close all peer connections
        Object.values(this.peerConnections).forEach(pc => pc.close());
        this.peerConnections = {};
    }
}

export default WebRTCHandler; 