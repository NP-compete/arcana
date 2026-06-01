"""Shared embedding utilities for Arcana services."""
from __future__ import annotations

import hashlib
import math

EMBEDDING_DIM = 64


def embed_text(text: str) -> list[float]:
    """Generate a deterministic embedding vector from text using SHA-256."""
    digest = hashlib.sha256(text.encode("utf-8")).digest()
    values = [((digest[i % len(digest)] / 255.0) * 2.0) - 1.0 for i in range(EMBEDDING_DIM)]
    norm = math.sqrt(sum(v * v for v in values)) or 1.0
    return [v / norm for v in values]


# Alias for backward compatibility
text_to_embedding = embed_text


def embedding_dim() -> int:
    """Return the dimension of embeddings."""
    return EMBEDDING_DIM


def cosine_similarity(a: list[float], b: list[float]) -> float:
    """Compute cosine similarity between two vectors."""
    if len(a) != len(b):
        min_len = min(len(a), len(b))
        a = a[:min_len]
        b = b[:min_len]
    dot = sum(x * y for x, y in zip(a, b, strict=False))
    norm_a = math.sqrt(sum(x * x for x in a)) or 1.0
    norm_b = math.sqrt(sum(x * x for x in b)) or 1.0
    return dot / (norm_a * norm_b)
