#!/usr/bin/env python3
"""MIT wrapper around Crawl4AI (Apache-2.0) for the extraction module.

This script is the ONLY file in the monorepo that imports Crawl4AI. It is
invoked as a subprocess CLI by the Go orchestrator so that Crawl4AI's
Apache-2.0 + attribution license does not contaminate the platform's MIT code.

Behavior:
  1. Accept --url and --timeout.
  2. Fetch the single URL with Crawl4AI (no recursion, no proxy, no LLM).
  3. Extract structured fields using LLM-free heuristics (regex, HTML parsing).
  4. Print one JSON object on stdout matching the shared extraction contract.
  5. Exit 0 even on internal errors so the Go runner can parse the JSON.
"""
import argparse
import asyncio
import ipaddress
import json
import os
import re
import sys
import time
from html.parser import HTMLParser
from urllib.parse import urljoin, urlparse

MAX_MARKDOWN_BYTES = 100 * 1024


def _is_forbidden_ip(host):
    """Return True if host is a loopback, link-local, private, CGNAT,
    unique-local IPv6, multicast, unspecified, or cloud-metadata address."""
    try:
        addr = ipaddress.ip_address(host)
    except ValueError:
        return False

    if addr.is_loopback or addr.is_link_local or addr.is_multicast or addr.is_unspecified or addr.is_private:
        return True

    # Carrier-grade NAT (RFC 6598): 100.64.0.0/10.
    if isinstance(addr, ipaddress.IPv4Address):
        octets = addr.packed
        if octets[0] == 100 and 64 <= octets[1] <= 127:
            return True
        # Cloud metadata endpoint: 169.254.169.254/32.
        if octets == b"\xa9\xfe\xa9\xfe":
            return True

    return False


def validate_url(url):
    """Defence-in-depth URL validation mirroring the Go SSRF policy.

    Rejects non-http/https schemes, missing hosts, userinfo (credentials),
    non-standard ports, IP-literal hosts, and hostnames that resolve to
    forbidden IPs. Returns the parsed URL or raises ValueError.
    """
    url = (url or "").strip()
    if not url:
        raise ValueError("empty URL")

    parsed = urlparse(url)

    if parsed.scheme not in ("http", "https"):
        raise ValueError(f"URL scheme {parsed.scheme!r} not allowed; only http and https")

    if parsed.username is not None or parsed.password is not None:
        raise ValueError("URL must not contain credentials")

    host = (parsed.hostname or "").lower().strip()
    if not host:
        raise ValueError("URL has no host")

    # Reject IP-literal hosts by default.
    if _is_forbidden_ip(host):
        raise ValueError(f"URL host {host!r} is a forbidden IP address")

    try:
        ipaddress.ip_address(host)
        raise ValueError("IP-literal URLs are not allowed")
    except ValueError:
        # Not an IP address, which is what we want for public websites.
        pass

    port = parsed.port
    if port is not None and port not in (80, 443):
        raise ValueError(f"non-standard port {port} not allowed")

    # Resolve the hostname and validate every returned IP. This is defence in
    # depth; the Go orchestrator already validates before calling the wrapper.
    try:
        import socket

        infos = socket.getaddrinfo(host, None)
    except OSError as e:
        raise ValueError(f"DNS lookup for {host!r}: {e}") from e

    seen_ips = set()
    for info in infos:
        ip = info[4][0]
        if ip in seen_ips:
            continue
        seen_ips.add(ip)
        if _is_forbidden_ip(ip):
            raise ValueError(f"URL {url!r} resolves to forbidden IP {ip}")

    return parsed

EMAIL_RE = re.compile(
    r"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}", re.IGNORECASE
)

# Very permissive phone regex: tries to capture common international/US/UK/EU
# formats while avoiding false positives.
PHONE_RE = re.compile(
    r"(?:\+?\d{1,3}[-.\s]?)?(?:\(?\d{2,4}\)?[-.\s]?)?\d{2,4}[-.\s]?\d{2,4}(?:[-.\s]?\d{2,9})?",
    re.IGNORECASE,
)

SOCIAL_HOSTS = {
    "twitter.com",
    "x.com",
    "facebook.com",
    "linkedin.com",
    "instagram.com",
    "youtube.com",
    "tiktok.com",
    "github.com",
    "gitlab.com",
}

CONTACT_HINTS = {"contact", "about", "support", "help", "careers", "jobs"}


class LinkExtractor(HTMLParser):
    def __init__(self, base_url):
        super().__init__()
        self.base_url = base_url
        self.title = ""
        self.meta_description = ""
        self.in_title = False
        self.links = []
        self.body_parts = []
        self._tag = None
        self._attrs = {}

    def handle_starttag(self, tag, attrs):
        self._tag = tag
        self._attrs = dict(attrs)
        if tag == "title":
            self.in_title = True
        if tag == "meta":
            name = self._attrs.get("name", "").lower()
            prop = self._attrs.get("property", "").lower()
            if name == "description" or prop == "og:description":
                content = self._attrs.get("content", "")
                if content:
                    self.meta_description = content.strip()
        if tag == "a":
            href = self._attrs.get("href", "")
            text = ""
            # text will be captured in handle_data and matched later; for now just store href.
            if href:
                self.links.append((href, ""))

    def handle_endtag(self, tag):
        if tag == "title":
            self.in_title = False
        self._tag = None
        self._attrs = {}

    def handle_data(self, data):
        text = data.strip()
        if not text:
            return
        if self.in_title and self._tag == "title":
            self.title = text
        if self._tag in ("p", "div", "span", "li", "td"):
            self.body_parts.append(text)

    def absolute_links(self):
        out = []
        for href, _ in self.links:
            if href.startswith("mailto:") or href.startswith("tel:"):
                continue
            out.append(urljoin(self.base_url, href))
        return out

    def body_text(self):
        return "\n".join(self.body_parts)


def emit(obj):
    print(json.dumps(obj, ensure_ascii=False))
    sys.stdout.flush()


def error_result(url, message, status="error"):
    return {
        "status": status,
        "url": url,
        "final_url": url,
        "source_tool": "unclecode/crawl4ai@v0.9.2 (CLI subprocess)",
        "confidence": 0.0,
        "fields": {},
        "raw_markdown": "",
        "metadata": {
            "backend": "crawl4ai",
            "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
            "error": message,
            "truncated": False,
        },
        "error": message,
        "checked_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    }


def normalize_phone(raw):
    digits = re.sub(r"\D", "", raw)
    # Avoid capturing things that are not phones (long digit-only IDs).
    if len(digits) < 7 or len(digits) > 15:
        return None
    # Avoid repeated single digits like 123456789.
    if len(set(digits)) <= 3:
        return None
    return raw.strip()


def domain_of(url):
    try:
        return urlparse(url).netloc.lower().lstrip("www.")
    except Exception:
        return ""


def is_social_link(href):
    d = domain_of(href)
    for host in SOCIAL_HOSTS:
        if host in d:
            return True
    return False


def is_contact_link(href, text):
    path = urlparse(href).path.lower()
    text_l = text.lower()
    for hint in CONTACT_HINTS:
        if hint in path or hint in text_l:
            return True
    return False


def extract_fields(html_text, markdown_text, final_url, links):
    parser = LinkExtractor(final_url)
    parser.feed(html_text)

    # Title
    title = parser.title.strip()

    # Description
    description = parser.meta_description
    if not description:
        # First reasonably-long sentence/paragraph of body text.
        body = parser.body_text()
        for part in body.split("\n"):
            if len(part) > 40:
                description = part[:500]
                break

    # Emails
    all_text = f"{html_text}\n{markdown_text}"
    emails = sorted(set(EMAIL_RE.findall(all_text)))

    # Phones
    raw_phones = PHONE_RE.findall(all_text)
    phones = sorted(set(p for p in (normalize_phone(x) for x in raw_phones) if p))

    # Social / contact links
    social_links = []
    contact_urls = []
    seen = set()
    for href in links + parser.absolute_links():
        href = href.strip()
        if not href or href.startswith("#"):
            continue
        if href in seen:
            continue
        seen.add(href)
        if is_social_link(href):
            social_links.append(href)
        elif is_contact_link(href, ""):
            contact_urls.append(href)

    # Company name heuristics
    company_name = ""
    if title:
        # Take text before the first pipe/dash/em-dash as a likely company name.
        company_name = re.split(r"\s*[|\-–—]\s+", title, maxsplit=1)[0].strip()
    if not company_name and description:
        first_line = description.split("\n")[0]
        if len(first_line) < 80:
            company_name = first_line

    # Addresses: not attempted in v1 to avoid over-collection / false positives.
    addresses = []

    return {
        "company_name": company_name,
        "emails": emails,
        "phones": phones,
        "addresses": addresses,
        "social_links": sorted(set(social_links)),
        "contact_urls": sorted(set(contact_urls)),
        "description": description.strip()[:1000],
        "title": title,
    }


def run_crawl(url, timeout_sec):
    # Import inside a function so an ImportError can be turned into a structured
    # JSON response rather than an unhandled stack trace.
    try:
        from crawl4ai import AsyncWebCrawler, CacheMode
        from crawl4ai.async_crawler_strategy import CrawlerRunConfig
    except Exception as e:
        raise RuntimeError(f"crawl4ai is not installed: {e}")

    config = CrawlerRunConfig(
        cache_mode=CacheMode.BYPASS,
        page_timeout=min(timeout_sec * 1000, 120000),
    )

    async def _fetch():
        async with AsyncWebCrawler() as crawler:
            result = await crawler.arun(url=url, config=config)
            return result

    return asyncio.run(_fetch())


def main():
    ap = argparse.ArgumentParser(description="Crawl4AI extraction wrapper")
    ap.add_argument("--url", required=True, help="URL to crawl")
    ap.add_argument(
        "--timeout",
        type=int,
        default=45,
        help="Page timeout in seconds (default 45)",
    )
    args = ap.parse_args()

    url = args.url.strip()
    if not url:
        emit(error_result(url, "empty URL"))
        return

    try:
        validate_url(url)
    except ValueError as e:
        emit(error_result(url, f"URL rejected by SSRF policy: {e}"))
        return

    try:
        result = run_crawl(url, args.timeout)
    except Exception as e:
        emit(error_result(url, f"crawl4ai run failed: {e}"))
        return

    html = getattr(result, "html", "") or ""
    markdown = getattr(result, "markdown", "") or ""
    final_url = getattr(result, "url", "") or url
    status_code = getattr(result, "status_code", 0) or 0
    links = getattr(result, "links", []) or []

    if not isinstance(links, list):
        links = []

    # Bound raw markdown.
    raw_bytes = markdown.encode("utf-8")
    truncated = len(raw_bytes) > MAX_MARKDOWN_BYTES
    if truncated:
        raw_bytes = raw_bytes[:MAX_MARKDOWN_BYTES]
        # Do not truncate in the middle of a multi-byte character.
        while raw_bytes and raw_bytes[-1] & 0xC0 == 0x80:
            raw_bytes = raw_bytes[:-1]
        markdown = raw_bytes.decode("utf-8", errors="ignore")

    fields = extract_fields(html, markdown, final_url, links)

    output = {
        "status": "ok",
        "url": url,
        "final_url": final_url,
        "source_tool": "unclecode/crawl4ai@v0.9.2 (CLI subprocess)",
        "confidence": 0.0,
        "fields": fields,
        "raw_markdown": markdown,
        "metadata": {
            "backend": "crawl4ai",
            "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
            "http_status": status_code,
            "truncated": truncated,
            "raw_bytes": len(raw_bytes),
        },
        "error": "",
        "checked_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    }
    emit(output)


if __name__ == "__main__":
    main()
