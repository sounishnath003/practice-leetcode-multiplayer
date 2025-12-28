# Practice Leetcode Multiplayer

> NOTE: Project is just a thought. Do not expect to be maintain nicely. Not sure, about future plans.

This project is a multiplayer platform for practicing Leetcode problems. It allows users to collaborate and compete in solving coding challenges in real-time with integrated audio calling and cloud code execution.

## YouTube Demo

[Practice Leetcode Multiplayer | Collaborate and practice together | Go + HTMX
](https://www.youtube.com/watch?v=3QiOIUQptu8)

## Deployed Service URL: 

[practice-leetcode-multiplayer-797087556919.asia-south1.run.app](https://practice-leetcode-multiplayer-797087556919.asia-south1.run.app)

## Latest Features

- **Collaborative Code Editor**: Real-time synchronized editor with syntax highlighting for Python, Java, JavaScript, and C++.
- **Remote Code Execution**: Execute code directly in the cloud using a dedicated **Serverless Execution Engine** (hosted on Google Cloud Run) with support for Python, Java, JavaScript, and C++.
- **Smart Search with Suggestions**: Find any LeetCode problem by name with a real-time suggestions dropdown showing the top 5 matches.
- **Integrated Audio Calls**: Seamless pair programming experience with built-in **WebRTC audio calling**.
- **Automatic Boilerplate**: Selecting a problem or changing languages automatically fetches the correct function stubs and starter code from LeetCode.
- **Full State Sync**: All participants stay in sync with the same code, programming language, and problem details via WebSockets.

## Architecture

- **Backend**: Go (Golang) with standard library `net/http` and `gorilla/websocket`.
- **Frontend**: HTML + Tailwind CSS + HTMX for dynamic interactions.
- **Code Execution**: Python-based Flask app running in a Docker container on Google Cloud Run, strictly isolated with execution timeouts.
- **Real-time**: WebSockets for state synchronization and WebRTC for peer-to-peer audio communication.

## UI Screens:

### Create room or Join room:

![signin](assets/signin.png)

### Multiplayer Screen (with Audio & Code Execution):

![multiplayer](assets/multiplayer.png)

## Getting Started

1. Clone the repository:
    ```bash
    git clone https://github.com/sounishnath003/practice-leetcode-multiplayer.git
    ```
2. Install dependencies:
    ```bash
    make install
    ```
3. Set up the Code Execution Engine URL (optional, defaults to production):
    ```bash
    export CODE_EXECUTION_ENGINE_URL="https://your-engine-url.run.app"
    ```
4. Start the application:
    ```bash
    make run
    ```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Port for the Go server | `3000` |
| `CODE_EXECUTION_ENGINE_URL` | URL of the deployed Cloud Run engine | `https://your-engine-url.run.app` |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to GCP service account JSON (for authenticated calls) | N/A |


## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.