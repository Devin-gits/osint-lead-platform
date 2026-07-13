#!/usr/bin/env python3
"""Scope-disciplined Maigret wrapper for the social-footprint module.

Maigret is embedded here as a *Python library* (its MIT license permits direct
embedding — see evaluations/maigret.md §2), NOT by shelling out to the `maigret`
CLI. The Go module (../socialfootprint.go) invokes THIS script as a subprocess:
that subprocess boundary is only the Go<->Python language bridge, mirroring the
subprocess pattern domain-intel uses for theHarvester. Inside the boundary we
call Maigret's async `maigret()` entrypoint directly.

Compliance guardrails enforced here in code (not just docs), per the Stage 1
decision (docs/decisions/stage-1-decision.md -> social-footprint) and
evaluations/maigret.md §6:

  1. SCOPE CAP. Maigret's default fans out to hundreds/thousands of sites. We
     restrict the search to an explicit, caller-supplied allow-list of sites and
     additionally hard-cap the count at --max-sites (default 20). Passing the
     full DB is impossible through this wrapper.
  2. NO RECURSION. Recursive pivoting onto discovered usernames/IDs is disabled
     (is_recursive_search=off) so a single lead never expands into a graph of
     other people's identities.
  3. NO BLOCK-EVASION. Residential proxies, Tor/I2P, and Cloudflare-bypass are
     never enabled — routing lead traffic through those to evade site WAFs is a
     ToS-circumvention posture the evaluation explicitly warns against.
  4. MINIMAL COLLECTION. We return only a per-platform claimed/available/unknown
     signal plus the public profile URL and HTTP status. We deliberately do NOT
     emit scraped profile fields (fullname, bio, location, linked accounts),
     matching docs/compliance.md's "avoid over-collecting beyond a simple
     match/no-match + confidence score" rule for the social-footprint category.

Output: a single JSON object on stdout (see build_output). Exit code is 0 when a
check ran (even with per-site errors, reported in-band) and non-zero only when
the wrapper could not run at all (bad args, Maigret import failure) — again with
a JSON error object on stdout so the Go caller can parse a reason either way.
"""
import argparse
import asyncio
import json
import logging
import os
import sys
from datetime import datetime, timezone

# Absolute ceiling on sites per invocation, independent of --max-sites, so no
# caller (or bug) can turn this into a bulk sweep. This is a hard code-level
# guardrail, not a default that can be raised via a flag.
ABSOLUTE_MAX_SITES = 30


def now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def build_output(username, requested, results, version, error=None):
    return {
        "tool": "maigret",
        "version": version,
        "username": username,
        "sites_requested": requested,
        "results": results,
        "checked_at": now_iso(),
        "error": error,
    }


def emit(obj, code):
    json.dump(obj, sys.stdout)
    sys.stdout.write("\n")
    sys.stdout.flush()
    sys.exit(code)


def parse_args(argv):
    p = argparse.ArgumentParser(description="scope-limited Maigret username check")
    p.add_argument("--username", required=True)
    p.add_argument("--sites", required=True,
                   help="comma-separated Maigret site names to check (allow-list)")
    p.add_argument("--timeout", type=int, default=15,
                   help="per-site request timeout in seconds")
    p.add_argument("--max-sites", type=int, default=20,
                   help="hard cap on number of sites checked this call")
    p.add_argument("--db", default="",
                   help="path to Maigret data.json (defaults to the bundled DB)")
    return p.parse_args(argv)


def resolve_db_path(explicit):
    if explicit:
        return explicit
    import maigret
    return os.path.join(os.path.dirname(maigret.__file__), "resources", "data.json")


def select_sites(db, requested, max_sites):
    """Intersect the requested allow-list with the DB, preserving request order,
    then truncate to the enforced cap. Unknown names are dropped (reported back
    to the caller as the difference between sites_requested and results)."""
    available = db.sites_dict
    cap = min(max_sites, ABSOLUTE_MAX_SITES)
    selected = {}
    for name in requested:
        if name in available and name not in selected:
            selected[name] = available[name]
        if len(selected) >= cap:
            break
    return selected


async def run_search(username, site_dict, timeout):
    # Maigret's low-level async entrypoint checks each site in site_dict exactly
    # once and returns. Recursive pivoting (guardrail 2) is a higher-level CLI
    # behavior built ON TOP of this function by re-invoking it with usernames
    # extracted from hits; by calling this function directly and never feeding
    # its discovered IDs back in, recursion simply never happens.
    from maigret.maigret import maigret as maigret_search

    logger = logging.getLogger("maigret")
    logger.setLevel(logging.CRITICAL)  # keep Maigret's chatter off our stdout/stderr

    raw = await maigret_search(
        username=username,
        site_dict=site_dict,
        logger=logger,
        timeout=timeout,
        is_parsing_enabled=False,    # guardrail 4: do not scrape profile fields
        proxy=None,                  # guardrail 3: no block-evasion
        tor_proxy=None,
        i2p_proxy=None,
        cloudflare_bypass=None,
        no_progressbar=True,
        retries=0,
    )
    return raw


# Maigret's status enum stringifies as Claimed/Available/Unknown/Illegal; we
# normalize to the module's lowercase claimed/available/unknown vocabulary.
_STATUS_MAP = {"CLAIMED": "claimed", "AVAILABLE": "available",
               "UNKNOWN": "unknown", "ILLEGAL": "unknown"}


def normalize(raw):
    results = []
    for site, data in raw.items():
        status_obj = data.get("status") if isinstance(data, dict) else None
        name = getattr(getattr(status_obj, "status", None), "name", "UNKNOWN")
        results.append({
            "platform": site,
            "status": _STATUS_MAP.get(name, "unknown"),
            "url": (data.get("url_user") if isinstance(data, dict) else None) or "",
            "http_status": (data.get("http_status") if isinstance(data, dict) else None),
        })
    results.sort(key=lambda r: r["platform"].lower())
    return results


def main(argv):
    args = parse_args(argv)
    requested = [s.strip() for s in args.sites.split(",") if s.strip()]

    try:
        import maigret
        version = getattr(maigret, "__version__", "unknown")
    except Exception as e:  # noqa: BLE001 - report any import failure to the caller
        emit(build_output(args.username, requested, [], "unknown",
                          error="maigret import failed: %s" % e), 3)

    if not requested:
        emit(build_output(args.username, requested, [], version,
                          error="no sites requested"), 2)

    try:
        from maigret.sites import MaigretDatabase
        db = MaigretDatabase().load_from_path(resolve_db_path(args.db))
    except Exception as e:  # noqa: BLE001
        emit(build_output(args.username, requested, [], version,
                          error="could not load Maigret site DB: %s" % e), 3)

    site_dict = select_sites(db, requested, args.max_sites)
    if not site_dict:
        emit(build_output(args.username, requested, [], version,
                          error="none of the requested sites exist in the Maigret DB"), 2)

    try:
        raw = asyncio.run(run_search(args.username, site_dict, args.timeout))
    except Exception as e:  # noqa: BLE001 - a whole-run failure still returns JSON
        emit(build_output(args.username, list(site_dict.keys()), [], version,
                          error="maigret search failed: %s" % e), 1)

    emit(build_output(args.username, list(site_dict.keys()), normalize(raw), version), 0)


if __name__ == "__main__":
    main(sys.argv[1:])
