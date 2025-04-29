"""
===================================================
Simple Efficient Code Runner Engine
Deployed on "Google Cloud Functions"
===================================================
Supports Language: CPP, Java, NodeJS, Python, Go
===================================================
"""
"""Code a Efficient Code Runner Engine"""

import os
import base64
import resource
import tempfile
import subprocess
import signal
from dataclasses import dataclass
from typing import Optional
from contextlib import contextmanager

import flask
import flask.typing
import functions_framework

# Constants :
SAMPLE_PY_CODE = """# Hello world program in python3.
def main():
    inputs=input("enter your code:")
    for i in range(4):
        print(f"User input {i=} val=HelloWorld {inputs=}")

main()
"""

SAMPLE_CPP_CODE="""
#include<iostream>
using namespace std;
int main() {
    cout << "Hello world! :-)" << endl;
    return 0;
}
"""

SAMPLE_GO_CODE="""
package main

import (
"fmt"
)

func main() {
    fmt.Println("Hello world!, from Golang")
}
"""

SAMPLE_NODE_CODE="""
console.log("i am executing javascript code....");
"""

class TimeoutException(Exception):
    pass

@contextmanager
def timeout(seconds):
    def signal_handler(signum, frame):
        raise TimeoutException("Timed out")
    
    # Set the signal handler and alarm
    signal.signal(signal.SIGALRM, signal_handler)
    signal.alarm(seconds)
    
    try:
        yield
    finally:
        # Disable the alarm
        signal.alarm(0)

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

        with timeout(5):
            process = subprocess.run(
                execute_cmd,
                input=stdin.encode("utf-8"),
                capture_output=True,
                cwd=temp_dir
            )

        return CodeOutput(
            stdout=process.stdout.decode("utf-8"),
            stderr=process.stderr.decode("utf-8"),
            message="Execution finished",
            error=bool(process.stderr)
        )

    except TimeoutException:
        return CodeOutput(
            stdout="",
            stderr="[TimeLimitExceeded]: Code execution timed out after 5 seconds",
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

def main():
    output = execute_code("python", encode_to_base64(SAMPLE_PY_CODE), "1 2 3")
    print(output)
    output = execute_code("cpp", encode_to_base64(SAMPLE_CPP_CODE), "")
    print(output)
    output = execute_code("nodejs", encode_to_base64(SAMPLE_NODE_CODE), "")
    print(output)
    output = execute_code("go", encode_to_base64(SAMPLE_GO_CODE), "")
    print(output)

@functions_framework.http
def main_function_handler(request:flask.Request) -> flask.typing.ResponseReturnValue:
    return "Hello World, from sounish-code-execution-engine"

if __name__ == "__main__":
    main()
    main_function_handler(None)
