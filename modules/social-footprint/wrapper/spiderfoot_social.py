#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Scope-disciplined SpiderFoot wrapper for the social-footprint module.

SpiderFoot is invoked here as a *Python library* (its MIT license permits
embedding -- see evaluations/spiderfoot.md), NOT by starting its web UI or
sfcli.py. The Go module invokes THIS script as a subprocess: the subprocess
boundary is only the Go<->Python language bridge, mirroring the Maigret/Sherlock
wrappers.

Compliance guardrails enforced in code:

  1. SCOPE CAP. Only a single SpiderFoot module is used (`sfp_accounts`, the
     Account Finder) and it is fed a curated, hard-coded subset of the
     WhatsMyName site list covering ~15 major social/profile platforms. The
     wrapper pre-seeds SpiderFoot's cache with this filtered list, so the
     module cannot fan out to WhatsMyName's full ~500 sites.
  2. NO RECURSION. A single `USERNAME` event is fed to the module; discovered
     accounts are emitted but never fed back in as new seeds.
  3. NO PROXY/TOR. The SpiderFoot config leaves all SOCKS/proxy fields empty
     and `verify=False` is the module's own default; we do not enable Tor or
     residential proxies.
  4. MINIMAL COLLECTION. Only a per-platform claimed/available/unknown signal
     plus the public profile URL are returned. Profile fields (name, bio,
     location, linked accounts) are discarded.

Output: a single JSON object on stdout. Exit code is 0 for any run that
produced a parseable result (including all-unknown / import failures) and
non-zero only for invalid CLI arguments.
"""
import argparse
import json
import os
import queue
import sys
import tempfile
from datetime import datetime, timezone

# Hard-coded maximum number of platforms per invocation, independent of the
# caller's --max-sites, so the list cannot be widened at runtime.
ABSOLUTE_MAX_SITES = 30

# Map the platform names used by the Go module to the exact site names in the
# curated WhatsMyName subset below.
PLATFORM_TO_SITE = {
    "GitHub": "GitHub (User)",
    "GitLab": "GitLab",
    "Reddit": "Reddit",
    "Twitter": "X",
    "Instagram": "Instagram",
    "Pinterest": "Pinterest",
    "Medium": "Medium",
    "Telegram": "Telegram",
    "Keybase": "Keybase",
    "HackerNews": "Hacker News",
    "Steam": "Steam",
    "SoundCloud": "SoundCloud",
    "Vimeo": "Vimeo",
    "About.me": "about.me",
    "Patreon": "Patreon",
}

# Build the reverse mapping once.
SITE_TO_PLATFORM = {v: k for k, v in PLATFORM_TO_SITE.items()}

# Curated WhatsMyName-style site definitions for the Account Finder module.
# These are copied from the WebBreacher/WhatsMyName project (CC BY-SA 4.0) and
# cover the same ~15 platforms the Go module considers high-signal for a
# per-lead spot check.
CURATED_SITES = [
    {
        "name": "about.me",
        "uri_check": "https://about.me/{account}",
        "e_code": 200,
        "e_string": " | about.me",
        "m_string": "<title>about.me</title>",
        "m_code": 404,
        "known": ["john", "jill"],
        "cat": "social",
    },
    {
        "name": "GitHub (User)",
        "uri_check": "https://api.github.com/users/{account}",
        "uri_pretty": "https://github.com/{account}",
        "headers": {"User-Agent": "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0"},
        "e_code": 200,
        "e_string": '"id":',
        "m_string": '"status": "404"',
        "m_code": 404,
        "known": ["test", "WebBreacher"],
        "cat": "coding",
        "protection": ["user-agent"],
    },
    {
        "name": "GitLab",
        "uri_check": "https://gitlab.com/api/v4/users?username={account}",
        "uri_pretty": "https://gitlab.com/{account}",
        "e_code": 200,
        "e_string": '"id":',
        "m_string": "[]",
        "m_code": 200,
        "known": ["skennedy", "KennBro"],
        "cat": "coding",
    },
    {
        "name": "Hacker News",
        "uri_check": "https://hacker-news.firebaseio.com/v0/user/{account}.json?print=pretty",
        "uri_pretty": "https://news.ycombinator.com/user?id={account}",
        "e_code": 200,
        "e_string": '"id" :',
        "m_string": "null",
        "m_code": 200,
        "known": ["mubix", "hggh"],
        "cat": "tech",
    },
    {
        "name": "Instagram",
        "uri_check": "https://www.instagram.com/{account}/",
        "e_code": 200,
        "e_string": "Posts - See Instagram photos and videos from",
        "m_string": '"routePath":null',
        "m_code": 200,
        "known": ["jennaortega", "cristiano"],
        "cat": "social",
    },
    {
        "name": "Keybase",
        "uri_check": "https://keybase.io/_/api/1.0/user/lookup.json?usernames={account}",
        "uri_pretty": "https://keybase.io/{account}",
        "e_code": 200,
        "e_string": '"id":',
        "m_string": '"them":[null]',
        "m_code": 200,
        "known": ["test", "mubix"],
        "cat": "social",
    },
    {
        "name": "Medium",
        "uri_check": "https://medium.com/@{account}/about",
        "strip_bad_char": ".",
        "e_code": 200,
        "e_string": "Medium member since",
        "m_string": "Out of nothing, something",
        "m_code": 404,
        "known": ["zulie", "jessicalexicus"],
        "cat": "news",
    },
    {
        "name": "Patreon",
        "uri_check": "https://www.patreon.com/{account}",
        "e_code": 200,
        "e_string": 'full_name":',
        "m_string": 'errorCode\": 404,',
        "m_code": 404,
        "known": ["mubix", "doughboys"],
        "cat": "finance",
    },
    {
        "name": "Pinterest",
        "uri_check": "https://www.pinterest.com/{account}/",
        "e_code": 200,
        "e_string": " - Profile | Pinterest",
        "m_string": 'id="home-main-title',
        "m_code": 301,
        "known": ["test123", "frickcollection"],
        "cat": "social",
    },
    {
        "name": "Reddit",
        "uri_check": "https://www.reddit.com/user/{account}/about.json",
        "uri_pretty": "https://www.reddit.com/user/{account}",
        "e_code": 200,
        "e_string": '"id":',
        "m_string": '"error":404',
        "m_code": 404,
        "known": ["koavf", "alabasterheart"],
        "cat": "social",
    },
    {
        "name": "SoundCloud",
        "uri_check": "https://soundcloud.com/{account}",
        "e_code": 200,
        "e_string": '"hydratable":"user"',
        "m_string": "<title>SoundCloud - Hear the world’s sounds</title>",
        "m_code": 404,
        "known": ["wakaflockaflame", "scumgang6ix9ine"],
        "cat": "music",
    },
    {
        "name": "Steam",
        "uri_check": "https://steamcommunity.com/id/{account}",
        "e_code": 200,
        "e_string": "g_rgProfileData =",
        "m_string": "Steam Community :: Error",
        "m_code": 200,
        "known": ["test"],
        "cat": "gaming",
    },
    {
        "name": "Telegram",
        "uri_check": "https://t.me/{account}",
        "e_code": 200,
        "e_string": "tgme_page_title",
        "m_string": "noindex, nofollow",
        "m_code": 200,
        "known": ["alice", "giovanni"],
        "cat": "social",
    },
    {
        "name": "Vimeo",
        "uri_check": "https://vimeo.com/{account}",
        "e_code": 200,
        "e_string": "og:type",
        "m_string": "VimeUhOh",
        "m_code": 404,
        "known": ["john", "alice"],
        "cat": "video",
    },
    {
        "name": "X",
        "uri_check": "https://api.x.com/i/users/username_available.json?username={account}",
        "uri_pretty": "https://x.com/{account}",
        "e_code": 200,
        "e_string": '"reason":"taken"',
        "m_string": '"reason":"available"',
        "m_code": 200,
        "known": ["WebBreacher", "OSINT_Tactical"],
        "cat": "social",
    },
]


def now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def emit(obj, code: int = 0):
    """Print one compact JSON object to stdout and exit."""
    json.dump(obj, sys.stdout, separators=(",", ":"), ensure_ascii=False)
    sys.stdout.write("\n")
    sys.stdout.flush()
    sys.exit(code)


def parse_args(argv):
    p = argparse.ArgumentParser(description="scope-limited SpiderFoot social username check")
    p.add_argument("--username", required=True, help="username/handle to check")
    p.add_argument("--sites", required=True, help="comma-separated platform names to check")
    p.add_argument("--timeout", type=int, default=12, help="per-site HTTP timeout in seconds")
    p.add_argument("--max-sites", type=int, default=20, help="hard cap on number of sites checked")
    return p.parse_args(argv)


def bootstrap_spiderfoot():
    """Import SpiderFoot and the Account Finder module, adding the
    SpiderFoot root directory to sys.path when the caller has supplied one.

    Returns (SpiderFoot, SpiderFootTarget, SpiderFootEvent, sfp_accounts).
    """
    root = os.environ.get("SOCIAL_FOOTPRINT_SPIDERFOOT_ROOT") or os.environ.get("SPIDERFOOT_ROOT", "")
    if root and os.path.isdir(root):
        if root not in sys.path:
            sys.path.insert(0, root)

    # If sflib is importable, make sure the directory that contains sflib.py
    # (which is also the modules/ directory parent) is on sys.path.
    try:
        import sflib
        sflib_dir = os.path.dirname(sflib.__file__)
        if sflib_dir and sflib_dir not in sys.path:
            sys.path.insert(0, sflib_dir)
    except Exception:
        pass

    try:
        from sflib import SpiderFoot
        from spiderfoot import SpiderFootTarget, SpiderFootEvent, SpiderFootHelpers
        import modules.sfp_accounts as sfp_accounts
    except Exception as e:
        raise ImportError("spiderfoot library/modules not importable; set SOCIAL_FOOTPRINT_SPIDERFOOT_ROOT to the SpiderFoot clone: %s" % e)

    # The Account Finder loads wordlists at setup time. We avoid requiring
    # those data files (and any false-positive filtering based on them) by
    # neutering the helpers for this scoped run.
    SpiderFootHelpers.humanNamesFromWordlists = staticmethod(lambda *a, **k: [])
    SpiderFootHelpers.dictionaryWordsFromWordlists = staticmethod(lambda *a, **k: [])
    SpiderFootHelpers.usernamesFromWordlists = staticmethod(lambda *a, **k: "")

    return SpiderFoot, SpiderFootTarget, SpiderFootEvent, sfp_accounts


def spiderfoot_config(timeout: int, cache_dir: str) -> dict:
    """Return a minimal SpiderFoot config that does not touch any proxy/Tor
    settings and suppresses logging."""
    return {
        "_debug": False,
        "_maxthreads": 5,
        "__logging": False,
        "__outputfilter": None,
        "_useragent": "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0",
        "_dnsserver": "",
        "_fetchtimeout": timeout,
        "_internettlds": [],
        "_internettlds_cache": 72,
        "_genericusers": "admin,user,test,guest,info,support,postmaster,root,webmaster",
        "__database": os.path.join(cache_dir, "spiderfoot.db"),
    }


def seed_cache(sf, cache_dir: str, sites: list) -> None:
    """Pre-populate the cache so sfp_accounts uses our curated list and does
    not fetch the full WhatsMyName database or perform its distrusted-site
    random-user probe."""
    sf.cachePut("sfaccountsv2", json.dumps({"sites": sites}))
    sf.cachePut("sfaccounts_state_v3", "None")


def select_sites(requested_platforms: list, max_sites: int) -> list:
    """Return the curated site definitions that match the requested platform
    names, preserving the order requested and capping at max_sites."""
    selected = []
    by_name = {site["name"]: site for site in CURATED_SITES}
    for platform in requested_platforms:
        site_name = PLATFORM_TO_SITE.get(platform, platform)
        site = by_name.get(site_name)
        if site:
            selected.append(site)
        if len(selected) >= max_sites:
            break
    return selected


def build_result(handle: str, requested: list, platforms: list, error: str = "") -> dict:
    """Build the JSON object printed to stdout."""
    claimed = sum(1 for p in platforms if p["status"] == "claimed")
    return {
        "tool": "spiderfoot",
        "version": "4.0",
        "username": handle,
        "sites_requested": requested,
        "handle": handle,
        "platforms": platforms,
        "results": platforms,
        "claimed_count": claimed,
        "checked_at": now_iso(),
        "source_tool": "SpiderFoot 4.0 (embedded Python library via wrapper subprocess)",
        "error": error,
    }


def main(argv=None):
    args = parse_args(argv or sys.argv[1:])
    requested = [s.strip() for s in args.sites.split(",") if s.strip()]
    if not requested:
        emit(build_result(args.username, [], [], error="no platforms requested"), 2)

    cap = min(args.max_sites, ABSOLUTE_MAX_SITES)
    sites = select_sites(requested, cap)
    if not sites:
        emit(build_result(args.username, requested, [], error="none of the requested platforms are in the SpiderFoot curated list"), 2)

    try:
        SpiderFoot, SpiderFootTarget, SpiderFootEvent, sfp_accounts = bootstrap_spiderfoot()
    except Exception as e:
        emit(build_result(args.username, requested, [], error=str(e)), 0)

    # Use a fresh temporary cache directory for each run so this wrapper is
    # stateless and cannot accidentally reuse a broader site list from a
    # previous SpiderFoot installation.
    cache_dir = tempfile.mkdtemp(prefix="sf_social_")
    os.environ["SPIDERFOOT_CACHE"] = cache_dir
    os.environ["SPIDERFOOT_DATA"] = cache_dir

    try:
        config = spiderfoot_config(args.timeout, cache_dir)
        sf = SpiderFoot(config)
        target = SpiderFootTarget(args.username, "USERNAME")

        seed_cache(sf, cache_dir, sites)

        mod = sfp_accounts.sfp_accounts()
        mod.__name__ = "sfp_accounts"
        mod.setTarget(target)
        mod.setDbh(None)
        mod.setup(sf, config)

        outq = queue.Queue()
        mod.outgoingEventQueue = outq
        mod.incomingEventQueue = queue.Queue()

        event = SpiderFootEvent("USERNAME", args.username, "spiderfoot_social_wrapper", None)
        mod.handleEvent(event)

        # Collect positive (claimed) events.
        claimed_urls = {}
        while True:
            try:
                evt = outq.get_nowait()
            except queue.Empty:
                break
            if evt.eventType == "ACCOUNT_EXTERNAL_OWNED":
                data = evt.data
                name = data.split(" (Category:")[0] if " (Category:" in data else ""
                url = ""
                if "<SFURL>" in data and "</SFURL>" in data:
                    url = data.split("<SFURL>")[1].split("</SFURL>")[0]
                if name:
                    claimed_urls[name] = url

        # Build the per-platform result table. A site that was checked and not
        # found is reported as "available"; sites with no siteResults entry
        # (e.g. skipped due to an error) are "unknown".
        results = []
        site_results = getattr(mod, "siteResults", {}) or {}
        for platform in requested:
            site_name = PLATFORM_TO_SITE.get(platform, platform)
            site = next((s for s in sites if s["name"] == site_name), None)
            if not site:
                results.append({"platform": platform, "status": "unknown", "url": "", "http_status": 0})
                continue

            ret_url = site.get("uri_pretty", site["uri_check"]).format(account=args.username)
            retname = f"{site['name']} (Category: {site['cat']})\n<SFURL>{ret_url}</SFURL>"

            if site_name in claimed_urls:
                status = "claimed"
                url = claimed_urls[site_name]
            elif retname in site_results and not site_results[retname]:
                status = "available"
                url = ret_url
            else:
                status = "unknown"
                url = ret_url

            results.append({"platform": platform, "status": status, "url": url, "http_status": 0})

        error = ""
        if mod.errorState:
            error = "one or more SpiderFoot modules entered an error state"

        emit(build_result(args.username, requested, results, error=error), 0)
    except Exception as e:
        emit(build_result(args.username, requested, [], error="spiderfoot check failed: %s" % e), 0)


if __name__ == "__main__":
    main()
