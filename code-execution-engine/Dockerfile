# Use an official Python runtime as a parent image
FROM python:3.11-slim

# Set environment variables
ENV PYTHONUNBUFFERED=True
ENV APP_HOME=/app
WORKDIR $APP_HOME

# Install system dependencies
# We need Java, Node.js, Go, and C++ (g++)
# We create the man directory to avoid issues with some java installers on slim images
RUN mkdir -p /usr/share/man/man1 && \
    apt-get update && apt-get install -y \
    default-jdk \
    nodejs \
    npm \
    golang \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
# We install dependencies directly from the updated pyproject.toml logic
# or just install them manually here to be safe and explicit
RUN pip install flask gunicorn

# Copy the local code to the container
COPY . ./ 

# Cloud Run sets the PORT environment variable (default 8080)
# Start Gunicorn
CMD exec gunicorn --bind :$PORT --workers 1 --threads 8 --timeout 0 main:app