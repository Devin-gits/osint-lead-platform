#!/usr/bin/env python3
"""Scope-disciplined Osintgram wrapper for the social-footprint module.

Osintgram is GPL-3.0, so this wrapper keeps it at arm's length: it invokes
Osintgram's main.py as a subprocess CLI only and normalizes the output to the
module's shared JSON contract. The wrapper itself is MIT-licensed platform code
and never imports Osintgram source.

Guardrails (enforced in code, not just docs):
  1. COMMAND ALLOWLIST: only "info" is permitted. All other Osintgram commands
     (followers, fwersemail, photos, addrs, ...) are rejected before spawn.
  2. MINIMAL COLLECTION: the "info" output is reduced to counts, privacy flags,
     and boolean presence signals. Biography text, full contact strings, HD
     profile images, and media downloads are intentionally NOT emitted.
  3. NO RECURSION: discovered IDs/usernames are never fed back in.
  4. NO PROXY/BLOCK-EVASION: no proxy/Tor/Cloudflare flags are passed.
  5. SECRETS: HIKERAPI_TOKEN is passed through env only; it never appears in
     stdout or in audit data.
  6. TEMP OUTPUT: each invocation writes to its own temp dir under /tmp and
     deletes it afterwards, so concurrent leads never race and the Osintgram
     install tree is not polluted.

Output: a single JSON object on stdout matching the shared wrapper schema.
"""
import argparse
import glob
import json
import os
import shutil
import subprocess
import sys
import tempfile
from datetime import datetime, timezone

# Optional path to a credentials.ini file. If set, the wrapper copies it into
# the Osintgram checkout so main.py can read config/credentials.ini relative to
# its working directory. The password itself is never read or logged by the
# wrapper; it is only copied into place for the subprocess to consume.
CREDENTIALS_ENV = "SOCIAL_FOOTPRINT_OSINTGRAM_CREDENTIALS"

# The ONLY Osintgram command this module is allowed to run in v1.
ALLOWED_COMMANDS = {"info"}

HOME_ENV = "SOCIAL_FOOTPRINT_OSINTGRAM_HOME"
TOKEN_ENV = "HIKERAPI_TOKEN"


def now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def emit(obj, code=0):
    json.dump(obj, sys.stdout)
    sys.stdout.write("\n")
    sys.stdout.flush()
    sys.exit(code)


def build_output(handle, results, error="", version="unknown"):
    return {
        "tool": "osintgram",
        "version": version,
        "username": handle,
        "sites_requested": ["Instagram"],
        "results": results,
        "checked_at": now_iso(),
        "error": error or "",
    }


def find_json_file(tmpdir, handle):
    """Discover the JSON file Osintgram wrote.

    Different versions/paths write either
      <tmpdir>/<handle>/<handle>_info.json
    or
      <tmpdir>/<handle>_info.json
    We glob recursively under tmpdir for any plausible info JSON and prefer the
    most handle-specific match.
    """
    patterns = [
        os.path.join(tmpdir, "**", f"{handle}*info*.json"),
        os.path.join(tmpdir, "**", "*info*.json"),
    ]
    found = []
    for p in patterns:
        found.extend(glob.glob(p, recursive=True))
    if not found:
        return None
    handle_matches = [p for p in found if os.path.basename(p).startswith(handle)]
    candidates = handle_matches or found
    # Prefer shorter/more specific paths; deterministic tie-break by path string.
    return min(candidates, key=lambda p: (len(p), p))


def _intish(v, default=0):
    if v is None:
        return default
    if isinstance(v, int):
        return v
    if isinstance(v, dict):
        v = v.get("count", 0)
    try:
        return int(v)
    except (TypeError, ValueError):
        return default


def normalize_info(data, handle):
    """Map Osintgram's info JSON to the shared platform schema."""
    user_id = data.get("id") or data.get("pk") or data.get("user_id") or ""
    follower_count = _intish(data.get("follower_count") or data.get("edge_followed_by"))
    following_count = _intish(data.get("following_count") or data.get("edge_follow"))
    media_count = _intish(data.get("media_count"))

    public_email = data.get("public_email") or data.get("email") or ""
    has_public_email = bool(public_email)

    is_private = bool(data.get("is_private"))
    is_verified = bool(data.get("is_verified"))
    is_business = bool(
        data.get("is_business") or data.get("is_business_account")
    )

    return {
        "platform": "Instagram",
        "status": "claimed",
        "url": f"https://www.instagram.com/{handle}/",
        "http_status": 200,
        "instagram": {
            "user_id": str(user_id),
            "is_private": is_private,
            "is_verified": is_verified,
            "is_business": is_business,
            "follower_count": follower_count,
            "following_count": following_count,
            "media_count": media_count,
            "has_public_email": has_public_email,
            "checked_via": "osintgram-cli",
        },
    }


def available_result(handle):
    return {
        "platform": "Instagram",
        "status": "available",
        "url": f"https://www.instagram.com/{handle}/",
        "http_status": 404,
        "instagram": {"checked_via": "osintgram-cli"},
    }


def parse_args(argv):
    p = argparse.ArgumentParser(description="scope-limited Osintgram wrapper")
    p.add_argument("--handle", required=True, help="Instagram handle to check")
    p.add_argument(
        "--command",
        default="info",
        help="Osintgram command to run (v1 allowlist: info only)",
    )
    p.add_argument(
        "--timeout",
        type=int,
        default=120,
        help="per-invocation subprocess timeout in seconds",
    )
    return p.parse_args(argv)


def main(argv=None):
    args = parse_args(argv or sys.argv[1:])
    handle = args.handle

    if args.command not in ALLOWED_COMMANDS:
        emit(
            build_output(
                handle,
                [],
                error=(
                    f"command {args.command!r} is not allowed; "
                    f"only {sorted(ALLOWED_COMMANDS)!r} are permitted"
                ),
            ),
            0,
        )

    home = (os.environ.get(HOME_ENV, "") or "").strip()
    if not home:
        emit(
            build_output(
                handle,
                [],
                error=(
                    f"{HOME_ENV} is not set; install Osintgram separately "
                    "and point it at the checkout"
                ),
            ),
            0,
        )

    main_py = os.path.join(home, "main.py")
    if not os.path.isfile(main_py):
        emit(
            build_output(
                handle,
                [],
                error=f"Osintgram main.py not found at {main_py}",
            ),
            0,
        )

    python = sys.executable or os.environ.get("PYTHON", "python3")

    # If a credentials.ini path is supplied, copy it into the Osintgram checkout
    # so main.py can find it without the wrapper ever reading the password.
    creds_path = os.environ.get(CREDENTIALS_ENV)
    if creds_path:
        try:
            config_dir = os.path.join(home, "config")
            os.makedirs(config_dir, exist_ok=True)
            shutil.copy(creds_path, os.path.join(config_dir, "credentials.ini"))
        except Exception as e:
            emit(
                build_output(
                    handle,
                    [],
                    error=f"could not copy {CREDENTIALS_ENV}={creds_path!r} into Osintgram config: {e}",
                ),
                0,
            )

    tmpdir = tempfile.mkdtemp(prefix="osintgram_wrapper_")
    try:
        env = os.environ.copy()
        # Both the --json flag and the JSON=y toggle are set; one of them is
        # enough across the Osintgram versions we support.
        env["JSON"] = "y"

        cmd = [
            python,
            "main.py",
            handle,
            "--command", args.command,
            "--json",
            "--output", tmpdir,
        ]

        try:
            proc = subprocess.run(
                cmd,
                cwd=home,
                env=env,
                capture_output=True,
                text=True,
                timeout=args.timeout,
            )
        except subprocess.TimeoutExpired as exc:
            emit(
                build_output(
                    handle,
                    [],
                    error=f"osintgram subprocess timed out after {args.timeout}s",
                ),
                0,
            )

        # Prefer the generated JSON file.
        json_file = find_json_file(tmpdir, handle)
        if json_file:
            try:
                with open(json_file, "r", encoding="utf-8") as f:
                    data = json.load(f)
                if not isinstance(data, dict):
                    data = {}
            except Exception as e:
                emit(
                    build_output(
                        handle,
                        [],
                        error=f"could not parse Osintgram JSON output: {e}",
                    ),
                    0,
                )
            emit(build_output(handle, [normalize_info(data, handle)]), 0)

        # No JSON file: inspect stdout/stderr for known states.
        combined = ((proc.stdout or "") + "\n" + (proc.stderr or "")).lower()
        not_found_markers = ["non exist", "not exist", "user not found", "not found"]
        if proc.returncode == 2 or any(m in combined for m in not_found_markers):
            emit(build_output(handle, [available_result(handle)]), 0)

        error_msg = (
            (proc.stderr or "").strip()
            or f"Osintgram exited {proc.returncode} without producing JSON output"
        )
        emit(build_output(handle, [], error=error_msg), 0)
    finally:
        # Best-effort cleanup of temp output; do not leak on failure.
        try:
            import shutil
            shutil.rmtree(tmpdir, ignore_errors=True)
        except Exception:
            pass


if __name__ == "__main__":
    main()
