FROM python:3.12-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy script
COPY pubsub-subscribe.py .
COPY pubsub-listener.py .

RUN chmod +x pubsub-subscribe.py
RUN chmod +x pubsub-listener.py

# Run as non-root user
RUN useradd -m -u 1000 subscriber
USER subscriber

ENTRYPOINT ["python", "pubsub-listener.py"]
