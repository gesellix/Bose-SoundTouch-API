# Use an official Python runtime as a parent image
FROM python:3.12-slim

# Set environment variables
# These can be overridden at runtime
ENV BASE_URL=""
ENV DATA_DIR="/data"
ENV PORT=8000

# Expose the port the app runs on
EXPOSE ${PORT}

# Create a directory for data persistence
RUN mkdir -p /data

# Set the working directory in the container
WORKDIR /app

# Install system dependencies if needed (none identified so far)
# RUN apt-get update && apt-get install -y --no-install-recommends gcc && rm -rf /var/lib/apt/lists/*

# Copy the requirements file into the container
COPY requirements.txt .

# Install any needed packages specified in requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy the rest of the application code into the container
COPY . .

# Run the application
# We use fastapi run to match the package structure and benefit from FastAPI's production optimizations
# We use exec to ensure the application receives signals (like SIGTERM) as PID 1
CMD ["sh", "-c", "exec fastapi run soundcork/main.py --port ${PORT} --host 0.0.0.0"]
