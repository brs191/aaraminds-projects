FROM python:3.11-slim
WORKDIR /app

COPY pyproject.toml ./
COPY scrum_orchestrator ./scrum_orchestrator
RUN pip install --no-cache-dir .

# Runs one Daily Brief through the graph (honors AUTO_APPROVE). For a long-running
# scheduler, this entrypoint is replaced in P1.
ENTRYPOINT ["run-daily-brief"]
