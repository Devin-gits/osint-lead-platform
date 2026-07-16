import Link from "next/link";

export function Footer() {
  return (
    <footer className="border-t border-border bg-surface px-4 py-3 text-xs text-foreground-muted">
      <div className="mx-auto flex max-w-7xl flex-col items-center justify-between gap-2 sm:flex-row">
        <p>
          Legal basis: <span className="text-foreground-secondary">GDPR Art.6(1)(f) legitimate-interest</span>
        </p>
        <Link href="/compliance" className="text-primary hover:underline">
          Compliance →
        </Link>
      </div>
    </footer>
  );
}
