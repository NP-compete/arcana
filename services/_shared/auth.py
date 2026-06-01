"""Shared authentication middleware for Arcana services."""
from __future__ import annotations

import os
import hmac
import hashlib
import time
from typing import Any

from fastapi import Depends, HTTPException, Request
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

_bearer = HTTPBearer(auto_error=False)
_AUTH_MODE = os.getenv("AUTH_MODE", "open")
_JWT_KEY = os.getenv("JWT_SIGNING_KEY", "")
_INTERNAL_TOKEN = os.getenv("ARCANA_INTERNAL_TOKEN", "")


def _decode_jwt_hs256(token: str, key: str) -> dict[str, Any]:
    """Minimal HS256 JWT decode without external dependencies."""
    import base64, json
    parts = token.split(".")
    if len(parts) != 3:
        raise ValueError("invalid JWT structure")

    def _b64(s: str) -> bytes:
        s += "=" * (-len(s) % 4)
        return base64.urlsafe_b64decode(s)

    header = json.loads(_b64(parts[0]))
    if header.get("alg") != "HS256":
        raise ValueError(f"unsupported algorithm: {header.get('alg')}")

    payload = json.loads(_b64(parts[1]))

    signing_input = f"{parts[0]}.{parts[1]}".encode()
    expected_sig = hmac.new(key.encode(), signing_input, hashlib.sha256).digest()
    actual_sig = _b64(parts[2])
    if not hmac.compare_digest(expected_sig, actual_sig):
        raise ValueError("invalid signature")

    if "exp" in payload and payload["exp"] < time.time():
        raise ValueError("token expired")

    return payload


async def require_auth(
    request: Request,
    credentials: HTTPAuthorizationCredentials | None = Depends(_bearer),
) -> dict[str, Any]:
    """FastAPI dependency that validates authentication.

    AUTH_MODE controls behavior:
    - "open": accept all requests (development)
    - "jwt": require valid JWT Bearer token (production)

    Also accepts X-Arcana-Internal-Token for inter-service calls.
    """
    if _AUTH_MODE == "open":
        return {"sub": "anonymous", "mode": "open"}

    internal = request.headers.get("x-arcana-internal-token")
    if internal and _INTERNAL_TOKEN and hmac.compare_digest(internal, _INTERNAL_TOKEN):
        return {"sub": "internal-service", "mode": "internal"}

    if not credentials:
        raise HTTPException(status_code=401, detail="missing authorization header")

    try:
        payload = _decode_jwt_hs256(credentials.credentials, _JWT_KEY)
        return payload
    except ValueError as e:
        raise HTTPException(status_code=401, detail=str(e))
