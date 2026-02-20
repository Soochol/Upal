#!/usr/bin/env python3
"""
Z-IMAGE inference server for Upal.

Standalone FastAPI server that runs Z-IMAGE text-to-image generation
using the HuggingFace diffusers ZImagePipeline.

Usage:
    # Install dependencies (full):
    pip install -r scripts/requirements-zimage.txt

    # Run with real model (requires GPU + downloaded weights):
    python scripts/zimage_server.py --port 8090

    # Run in mock mode (no GPU, no model — returns test images):
    pip install fastapi uvicorn Pillow pydantic
    python scripts/zimage_server.py --mock --port 8090

    # Self-test (load model, generate one image, verify, exit):
    python scripts/zimage_server.py --self-test

Endpoints:
    POST /generate  — Generate an image from a text prompt
    GET  /health    — Health check
"""

import argparse
import base64
import io
import logging
import os
import sys
import time
from datetime import datetime

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import uvicorn

logger = logging.getLogger("zimage-server")

app = FastAPI(title="Z-IMAGE Server")
pipe = None
mock_mode = False
mock_image_b64: str = ""
mock_delay: float = 0.0
output_dir: str = ""


class GenerateRequest(BaseModel):
    prompt: str
    width: int = 1024
    height: int = 1024
    steps: int = 28
    guidance_scale: float = 4.0


class GenerateResponse(BaseModel):
    image: str  # base64-encoded image
    mime_type: str = "image/png"
    file_path: str = ""  # saved file path (if output_dir configured)


@app.get("/health")
def health():
    return {
        "status": "ok",
        "model_loaded": pipe is not None,
        "mock": mock_mode,
    }


@app.post("/generate", response_model=GenerateResponse)
def generate(req: GenerateRequest):
    if mock_mode:
        if mock_delay > 0:
            time.sleep(mock_delay)
        return GenerateResponse(image=mock_image_b64, mime_type="image/png")

    if pipe is None:
        raise HTTPException(status_code=503, detail="Model not loaded")

    try:
        t0 = time.time()
        result = pipe(
            req.prompt,
            width=req.width,
            height=req.height,
            num_inference_steps=req.steps,
            guidance_scale=req.guidance_scale,
        )
        elapsed = time.time() - t0
        img = result.images[0]

        buf = io.BytesIO()
        img.save(buf, format="PNG")
        b64 = base64.b64encode(buf.getvalue()).decode("utf-8")

        saved_path = ""
        if output_dir:
            ts = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"{ts}_{req.width}x{req.height}.png"
            saved_path = os.path.join(output_dir, filename)
            img.save(saved_path)
            logger.info(f"Image saved: {saved_path}")

        logger.info(f"Generated {req.width}x{req.height} in {elapsed:.1f}s ({req.steps} steps)")
        return GenerateResponse(image=b64, mime_type="image/png", file_path=saved_path)
    except Exception as e:
        logger.exception("Generation failed")
        raise HTTPException(status_code=500, detail=str(e))


def _detect_device():
    """Detect best available device: cuda > rocm (hip) > cpu."""
    import torch

    if torch.cuda.is_available():
        name = torch.cuda.get_device_name(0)
        logger.info(f"Using CUDA: {name}")
        return "cuda"
    if hasattr(torch, "hip") or "rocm" in str(getattr(torch.version, "hip", "")):
        try:
            if torch.cuda.is_available():  # ROCm exposes via cuda API
                name = torch.cuda.get_device_name(0)
                logger.info(f"Using ROCm: {name}")
                return "cuda"
        except Exception:
            pass
    logger.info("Using CPU (slow)")
    return "cpu"


def load_model(model_id: str):
    """Load the Z-IMAGE model via diffusers ZImagePipeline."""
    global pipe
    import torch
    from diffusers import ZImagePipeline

    logger.info(f"Loading model: {model_id}")
    t0 = time.time()
    pipe = ZImagePipeline.from_pretrained(
        model_id,
        torch_dtype=torch.bfloat16,
    )
    device = _detect_device()
    if device != "cpu":
        pipe = pipe.to(device)
    else:
        pipe.enable_model_cpu_offload()

    elapsed = time.time() - t0
    logger.info(f"Model loaded in {elapsed:.1f}s")


def init_mock():
    """Initialize mock mode with a small test image."""
    global mock_mode, mock_image_b64
    from PIL import Image

    mock_mode = True
    img = Image.new("RGB", (64, 64), color=(100, 149, 237))  # cornflower blue
    buf = io.BytesIO()
    img.save(buf, format="PNG")
    mock_image_b64 = base64.b64encode(buf.getvalue()).decode("utf-8")
    logger.info("Mock mode initialized (64x64 test image)")


def run_self_test(model_id: str):
    """Load model, generate one test image, verify output, exit."""
    import torch
    from PIL import Image

    load_model(model_id)

    logger.info("Running self-test: generating test image...")
    t0 = time.time()
    result = pipe(
        "a red circle on a white background",
        width=512,
        height=512,
        num_inference_steps=28,
        guidance_scale=4.0,
        generator=torch.Generator(pipe.device).manual_seed(42),
    )
    elapsed = time.time() - t0
    img = result.images[0]

    if not isinstance(img, Image.Image):
        logger.error("Self-test FAILED: output is not a PIL Image")
        sys.exit(1)

    w, h = img.size
    if w != 512 or h != 512:
        logger.error(f"Self-test FAILED: expected 512x512, got {w}x{h}")
        sys.exit(1)

    out_path = "scripts/test_output.png"
    img.save(out_path)
    logger.info(f"Self-test PASSED: {w}x{h} image in {elapsed:.1f}s -> {out_path}")
    sys.exit(0)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Z-IMAGE inference server")
    parser.add_argument("--port", type=int, default=8090)
    parser.add_argument("--host", default="0.0.0.0")
    # Resolve default model path: use project-local models/ if it exists,
    # otherwise fall back to HuggingFace model ID (downloads to ~/.cache).
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_dir = os.path.dirname(script_dir)
    local_model = os.path.join(project_dir, "models", "z-image")
    default_model = local_model if os.path.isdir(local_model) else "Tongyi-MAI/Z-Image"

    parser.add_argument(
        "--model",
        default=default_model,
        help="HuggingFace model ID or local path (default: models/z-image if exists)",
    )
    parser.add_argument(
        "--mock",
        action="store_true",
        help="Mock mode: return test images without loading a model",
    )
    parser.add_argument(
        "--mock-delay",
        type=float,
        default=0.5,
        help="Simulated generation delay in seconds (mock mode only)",
    )
    parser.add_argument(
        "--self-test",
        action="store_true",
        help="Load model, generate one test image, verify, and exit",
    )
    parser.add_argument(
        "--output-dir",
        default=os.path.join(project_dir, "output", "zimage"),
        help="Directory to save generated images (default: output/zimage/)",
    )
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(name)s %(levelname)s %(message)s",
    )

    output_dir = args.output_dir
    os.makedirs(output_dir, exist_ok=True)
    logger.info(f"Output directory: {output_dir}")

    if args.self_test:
        run_self_test(args.model)
    elif args.mock:
        mock_delay = args.mock_delay
        init_mock()
    else:
        load_model(args.model)

    uvicorn.run(app, host=args.host, port=args.port)
