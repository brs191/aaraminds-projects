# WhisperX STT sidecar — CPU image.
#
# NOTE: correct by inspection but UNTESTED — docker (and the multi-GB
# torch/whisperx install) is not available in the environment this was
# authored in. For GPU, base on nvidia/cuda + cuDNN instead and run with
# `--gpus all`; see README.md.
FROM python:3.11-slim

# ffmpeg is required by whisperx.load_audio for decoding mp3/m4a/mp4/...
RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app.py .

# Model + HF caches; mount a volume here to avoid re-downloading ~1.5GB.
ENV HF_HOME=/root/.cache/huggingface
EXPOSE 9090

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "9090"]
