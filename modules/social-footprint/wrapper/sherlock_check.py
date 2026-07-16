#!/usr/bin/env python3
"""Scope-disciplined Sherlock wrapper for the social-footprint module.

Sherlock is embedded here as a Python library (MIT license). The Go module
invokes this script as a subprocess, parsing its JSON stdout.

Guardrails (same as wrapper/maigret_check.py):
1. SCOPE CAP: only explicit allow-listed sites are checked, hard-capped at ABSOLUTE_MAX_SITES.
2. NO PROXY/TOR: proxy=None always.
3. MINIMAL COLLECTION: only claimed/available/unknown + URL + http_status.
4. LOCAL DB: bundled data.json only — never fetch the live manifest.
5. NO NSFW: remove_nsfw_sites() called unconditionally.
"""
import argparse
import json
import sys
from datetime import datetime, timezone

ABSOLUTE_MAX_SITES = 30

_STATUS_MAP = {
    "Claimed":   "claimed",
    "Available": "available",
    "Unknown":   "unknown",
    "Illegal":   "unknown",   # format disallowed by site — check not performed
    "WAF":       "unknown",   # WAF block — result unreliable
}


def now_iso():
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def build_output(username, requested, results, version, error=None):
    return {
        "tool":            "sherlock",
        "version":         version,
        "username":        username,
        "sites_requested": requested,
        "results":         results,
        "checked_at":      now_iso(),
        "error":           error,
    }


def emit(obj, code):
    sys.stdout.write(json.dumps(obj, separators=(",", ":")) + "\n")
    sys.stdout.flush()
    sys.exit(code)


def parse_args(argv):
    p = argparse.ArgumentParser(description="Sherlock username wrapper")
    p.add_argument("--username",  required=True)
    p.add_argument("--sites",     required=True,  help="comma-separated site names from Sherlock DB")
    p.add_argument("--timeout",   type=int, default=15)
    p.add_argument("--max-sites", type=int, default=20)
    return p.parse_args(argv)


def main(argv=None):
    args = parse_args(argv or sys.argv[1:])
    requested = [s.strip() for s in args.sites.split(",") if s.strip()]

    # --- Import Sherlock as a library ---
    try:
        import os
        import sherlock_project
        from sherlock_project.sherlock import sherlock
        from sherlock_project.sites import SitesInformation
        from sherlock_project.notify import QueryNotifyPrint
        from sherlock_project.result import QueryStatus
        version = getattr(sherlock_project, "__version__", "0.16.1")
    except Exception as e:
        emit(build_output(args.username, requested, [], "unknown",
                          error="sherlock import failed: %s" % e), 3)

    # --- Load BUNDLED data.json — never the live manifest ---
    try:
        local_db = os.path.join(
            os.path.dirname(sherlock_project.__file__), "resources", "data.json"
        )
        sites = SitesInformation(local_db, honor_exclusions=False)
        sites.remove_nsfw_sites()  # guardrail: unconditional
    except Exception as e:
        emit(build_output(args.username, requested, [], version,
                          error="could not load Sherlock DB: %s" % e), 3)

    # --- Intersect allow-list with DB; enforce cap ---
    cap = min(args.max_sites, ABSOLUTE_MAX_SITES)
    site_data_all = {site.name: site.information for site in sites}
    site_data = {}
    for name in requested:
        if name in site_data_all and name not in site_data:
            site_data[name] = site_data_all[name]
        if len(site_data) >= cap:
            break

    if not site_data:
        emit(build_output(args.username, requested, [], version,
                          error="none of the requested sites exist in the Sherlock DB"), 2)

    # --- Run Sherlock (synchronous, thread-pool internally) ---
    try:
        query_notify = QueryNotifyPrint(
            result=None, verbose=False, print_all=False, browse=False
        )
        raw = sherlock(
            username=args.username,
            site_data=site_data,
            query_notify=query_notify,
            proxy=None,           # guardrail: no proxy/block-evasion
            timeout=args.timeout,
        )
    except Exception as e:
        emit(build_output(args.username, list(site_data.keys()), [], version,
                          error="sherlock search failed: %s" % e), 1)

    # --- Normalize to shared schema ---
    results = []
    for site_name, data in raw.items():
        status_obj = data.get("status")
        # QueryResult.status is a QueryStatus enum; str() gives "Claimed" etc.
        status_str = (
            str(status_obj.status)
            if status_obj and hasattr(status_obj, "status")
            else "Unknown"
        )
        http_status = data.get("http_status")
        # Sherlock returns "?" when no HTTP response was obtained — normalise to null
        if http_status == "?" or not isinstance(http_status, int):
            http_status = None
        results.append({
            "platform":    site_name,
            "status":      _STATUS_MAP.get(status_str, "unknown"),
            "url":         data.get("url_user", ""),
            "http_status": http_status,
        })
    results.sort(key=lambda r: r["platform"].lower())

    emit(build_output(args.username, list(site_data.keys()), results, version), 0)


if __name__ == "__main__":
    main()
