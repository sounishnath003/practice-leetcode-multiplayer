"""
===================================================
Simple Efficient Code Runner Engine
Deployed on "Google Cloud Run"
===================================================
Supports Language: CPP, Java, NodeJS, Python, Go
===================================================
"""

import os
import base64
import resource
import tempfile
import subprocess
from dataclasses import dataclass
from typing import Optional

import flask

def limit_resource_memory(memory_limit_bytes: int):
    """Sets memory limit for child process"""
    def preexec_fn():
        resource.setrlimit(resource.RLIMIT_AS, (memory_limit_bytes, memory_limit_bytes))
    return preexec_fn

def encode_to_base64(content: str) -> str:
    """Encode the content to base 64 encoded string"""
    return base64.b64encode(content.encode("utf-8")).decode("utf-8")

def decode_from_base64(b64string: str) -> str:
    """Decode the base64 encoded string into original"""
    return base64.b64decode(b64string.encode("utf-8")).decode("utf-8")

@dataclass
class LanguageConfig:
    language: str
    filename: str
    execute_command: list[str]
    compile_command: Optional[list[str]] = None

LANGUAGE_CONFIG = {
    "cpp": LanguageConfig(
        language="cpp", 
        filename="solution.cpp",
        compile_command=["g++", "-O3", "-std=c++17", "solution.cpp", "-o", "solution"],
        execute_command=["./solution"]
    ),
    "python": LanguageConfig(
        language="python3", 
        filename="solution.py",
        execute_command=["python3"]
    ),
    "nodejs": LanguageConfig(
        language="nodejs", 
        filename="solution.js",
        execute_command=["node"]
    ),
    "java": LanguageConfig(
        language="java",
        filename="Solution.java",
        compile_command=["javac", "Solution.java"],
        execute_command=["java", "Solution"]
    ),
    "go": LanguageConfig(
        language="go",
        filename="solution.go", 
        execute_command=["go", "run"]
    )
}

@dataclass
class CodeOutput:
    stdout: str
    stderr: str
    message: str
    error: bool

def execute_code(language: str, codeb64encoded: str, stdin: str) -> Optional[CodeOutput]:
    code_file_name = None
    compiled_file = None
    temp_dir = None
    try:
        config = LANGUAGE_CONFIG.get(language)
        if not config:
            raise ValueError(f"Code language is not yet supported: {language}")

        # Create temporary directory
        temp_dir = tempfile.mkdtemp()
        
        # Write code to temporary file in temp directory
        code_file_name = os.path.join(temp_dir, config.filename)
        with open(code_file_name, 'w') as codefile:
            codefile.write(decode_from_base64(codeb64encoded))

        # Compile if needed
        if config.compile_command:
            compile_cmd = [cmd.replace("solution.cpp", code_file_name) if cmd == "solution.cpp"
                         else cmd.replace("solution", os.path.join(temp_dir, "solution"))
                         for cmd in config.compile_command]
            compile_process = subprocess.run(
                compile_cmd,
                capture_output=True,
                text=True,
                cwd=temp_dir
            )
            if compile_process.returncode != 0:
                return CodeOutput(
                    stdout="",
                    stderr=compile_process.stderr,
                    message="Compilation failed",
                    error=True
                )
            compiled_file = os.path.join(temp_dir, "solution")

        # Execute code with timeout
        execute_cmd = config.execute_command
        if compiled_file:
            execute_cmd = [cmd.replace("./solution", compiled_file) for cmd in execute_cmd]
        else:
            execute_cmd = execute_cmd + [code_file_name]

        # Use subprocess built-in timeout
        process = subprocess.run(
            execute_cmd,
            input=stdin.encode("utf-8"),
            capture_output=True,
            cwd=temp_dir,
            timeout=3
        )

        return CodeOutput(
            stdout=process.stdout.decode("utf-8"),
            stderr=process.stderr.decode("utf-8"),
            message="Execution finished",
            error=bool(process.stderr)
        )

    except subprocess.TimeoutExpired:
        return CodeOutput(
            stdout="",
            stderr="[TimeLimitExceeded]: Code execution timed out after 3 seconds",
            message="Time limit exceeded",
            error=True
        )
    except Exception as e:
        return CodeOutput(
            stdout="",
            stderr=str(e),
            message="Execution failed",
            error=True
        )
    finally:
        # Cleanup temporary files and directory
        if temp_dir and os.path.exists(temp_dir):
            for file in os.listdir(temp_dir):
                os.remove(os.path.join(temp_dir, file))
            os.rmdir(temp_dir)

app = flask.Flask(__name__)

# Helper function for allowed origins and referers
def is_allowed_request():
    # This function checks origin/referer/host headers for our allowed domains
    allowed_domains = [
        "http://localhost:3000",
        "https://practice-leetcode-multiplayer-797087556919.asia-south1.run.app",
        # support with/without trailing slash
        "practice-leetcode-multiplayer-797087556919.asia-south1.run.app",
    ]
    origin = flask.request.headers.get("Origin", "").rstrip("/")
    referer = flask.request.headers.get("Referer", "").rstrip("/")
    host = flask.request.host
    # Also allow from code itself
    if origin in allowed_domains:
        return True
    if referer:
        # e.g. Referer might include paths (/), so check startswith
        for dom in allowed_domains:
            if referer.startswith(dom):
                return True
    # For backend-to-backend through internal Cloud Run
    # "Host" could be "practice-leetcode-multiplayer-797087556919.asia-south1.run.app" without protocol
    for dom in allowed_domains:
        if dom.startswith("http"):
            dom2 = dom.split("://", 1)[-1]
        else:
            dom2 = dom
        if host.lower() == dom2.lower():
            return True
    return False

@app.route("/health", methods=["GET"])
def health_check():
    return "OK", 200

@app.route("/", methods=["POST", "OPTIONS"])
def execute_code_handler():
    """HTTP Handler to execute code, only for allowed domains."""
    allowed_origins = [
        "http://localhost:3000",
        "https://practice-leetcode-multiplayer-797087556919.asia-south1.run.app"
    ]

    # Set CORS headers for the preflight request
    if flask.request.method == "OPTIONS":
        req_origin = flask.request.headers.get("Origin", "")
        if req_origin in allowed_origins:
            cors_headers = {
                "Access-Control-Allow-Origin": req_origin,
                "Access-Control-Allow-Methods": "POST",
                "Access-Control-Allow-Headers": "Content-Type",
                "Access-Control-Max-Age": "3600"
            }
        else:
            # Not an allowed origin
            return ("", 403, {"Access-Control-Allow-Origin": "null"})
        return ("", 204, cors_headers)

    # For the POST (execution), check if sender is allowed
    if not is_allowed_request():
        # Disallowed; also echo with null origin for CORS
        return ({"error": "Forbidden: requests only allowed from specific domains."}, 403, {"Access-Control-Allow-Origin": "null"})

    # Now: set Access-Control-Allow-Origin header for allowed origins, if present
    req_origin = flask.request.headers.get("Origin", "")
    headers = {"Access-Control-Allow-Origin": req_origin if req_origin in allowed_origins else allowed_origins[0]}

    try:
        request_json = flask.request.get_json(silent=True)
        if not request_json:
            return ({"error": "Invalid JSON"}, 400, headers)

        # Normalize language to lowercase
        language = request_json.get("language", "").lower()
        code_b64 = request_json.get("code")
        stdin = request_json.get("stdin", "")

        if not language or not code_b64:
            return ({"error": "Missing 'language' or 'code' field"}, 400, headers)

        # Map frontend language names to engine language names if necessary
        lang_map = {
            "javascript": "nodejs",
            "python": "python",
            "java": "java",
            "go": "go",
            "cpp": "cpp"
        }
        
        engine_lang = lang_map.get(language, language)

        result = execute_code(engine_lang, code_b64, stdin)
        
        if result is None:
             return ({"error": "Execution failed internally"}, 500, headers)

        return (flask.jsonify({
            "stdout": result.stdout,
            "stderr": result.stderr,
            "message": result.message,
            "error": result.error
        }), 200, headers)

    except Exception as e:
        return ({"error": str(e)}, 500, headers)

if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8080))
    app.run(host="0.0.0.0", port=port)
