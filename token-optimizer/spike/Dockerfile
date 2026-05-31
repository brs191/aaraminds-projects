# Custom LiteLLM proxy image with the LLMLingua-2 compression hook baked in.
#
# Pinned 2026-05-27 to v1.83.14-stable.patch.3 — the newest patch on
# BerriAI/litellm's explicit `-stable` maintenance branch. This branch is
# tagged separately from the moving "main-stable" float and from the faster
# feature track (v1.84+), and is what to follow for production-adjacent use.
#
# Before running the spike, verify against:
#   https://github.com/BerriAI/litellm/security/advisories
# If your security policy requires immutable digest pinning, run:
#   docker pull ghcr.io/berriai/litellm:v1.83.14-stable.patch.3
#   docker inspect --format='{{index .RepoDigests 0}}' \
#     ghcr.io/berriai/litellm:v1.83.14-stable.patch.3
# and replace the tag below with @sha256:<digest>.
#
# To re-pin: re-check tags at https://api.github.com/repos/BerriAI/litellm/tags
# Pick the newest tag matching `vX.Y.Z-stable.patch.N` (NOT `-rc`, `-dev`,
# `-nightly`, or `main-stable`).
FROM ghcr.io/berriai/litellm:v1.83.14-stable.patch.3

WORKDIR /app

# Install the compression dependencies (torch is large — this layer is slow
# to build the first time; it caches afterwards).
COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

# Hook + config. The hook module must be importable, so /app is on PYTHONPATH.
COPY compression_hook.py /app/compression_hook.py
COPY litellm_config.yaml /app/litellm_config.yaml
ENV PYTHONPATH=/app

# Metrics are written here; docker-compose mounts it to the host.
RUN mkdir -p /app/metrics

EXPOSE 4000
CMD ["--config", "/app/litellm_config.yaml", "--port", "4000"]
