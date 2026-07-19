export type PipelineStage =
  | "raw"
  | "enriched"
  | "validated"
  | "crm_ready";

export type RiskLevel = "low" | "medium" | "high" | "unknown";

export type ModuleStatus = "ok" | "unknown" | "skipped" | "pending" | "not_run";

export type ModuleName =
  | "email-validate"
  | "domain-intel"
  | "phone-validate"
  | "social-footprint"
  | "extraction"
  | "company-enrich";

export interface LeadResultMeta {
  checked_at?: string;
  source_tool?: string;
  error?: string;
}

export interface EmailValidateResult extends LeadResultMeta {
  status: "ok" | "unknown" | "skipped";
  deliverable: "yes" | "no" | "unknown";
  syntax_valid?: boolean;
  has_mx_records?: boolean;
  is_disposable?: boolean;
  is_role_account?: boolean;
  is_free_provider?: boolean;
}

export interface DNSRecord {
  a?: string[];
  aaaa?: string[];
  cname?: string[];
  mx?: string[];
  ns?: string[];
  txt?: string[];
}

export interface SSLInfo {
  subject?: string;
  issuer?: string;
  valid?: boolean;
  not_before?: string;
  not_after?: string;
  days_until_expiry?: number;
  protocol?: string;
  sans?: string[];
}

export interface HTTPInfo {
  status_code?: number;
  server?: string;
  headers?: Record<string, string>;
}

export interface WhoisInfo {
  registrar?: string;
  created_date?: string;
  domain_age_days?: number;
}

export interface WebCheckResult extends LeadResultMeta {
  status: ModuleStatus;
  resolvable?: boolean;
  dns?: DNSRecord;
  ssl?: SSLInfo;
  http?: HTTPInfo;
  whois?: WhoisInfo;
}

export interface HarvesterResult extends LeadResultMeta {
  status: ModuleStatus;
  emails?: string[];
  hosts?: unknown[];
  ips?: string[];
  sources?: string[];
}

export interface DomainIntelResult extends LeadResultMeta {
  status: ModuleStatus;
  web_check?: WebCheckResult;
  harvester?: HarvesterResult;
  source_tools?: string[];
}

export interface PhoneValidateResult extends LeadResultMeta {
  status: ModuleStatus;
  e164?: string;
  region?: string;
  line_type?: string;
  carrier?: string;
  risk_flags?: string[];
}

export interface PlatformSignal {
  platform: string;
  status: "claimed" | "available" | "unknown";
  url?: string;
  http_status?: number;
}

export interface HandleResult {
  handle: string;
  origin: string;
  status: ModuleStatus;
  platforms: PlatformSignal[];
  claimed_count: number;
  checked_at: string;
  source_tool: string;
  error?: string;
}

export interface SocialFootprintResult extends LeadResultMeta {
  status: ModuleStatus;
  reason?: string;
  handles_checked?: string[];
  handles?: HandleResult[];
  active_signals?: number;
  confidence?: number;
  metadata?: Record<string, unknown>;
  rate_limit_note?: string;
  // Legacy fields kept for backwards compatibility with older payloads.
  matches?: { platform: string; handle: string; confidence: number; url?: string }[];
  summary?: string;
}

export interface ExtractionResult extends LeadResultMeta {
  status: ModuleStatus;
  url?: string;
  final_url?: string;
  source_tool?: string;
  confidence?: number;
  fields?: {
    company_name?: string;
    emails?: string[];
    phones?: string[];
    addresses?: string[];
    social_links?: string[];
    contact_urls?: string[];
    description?: string;
    title?: string;
  };
  raw_markdown?: string;
  provenance?: {
    field: string;
    value: string;
    source_url: string;
    method: string;
    timestamp: string;
  }[];
  metadata?: {
    backend?: string;
    legal_basis?: string;
    permission_ref?: string;
    http_status?: number;
    truncated?: boolean;
    raw_bytes?: number;
    duration_ms?: number;
    limits_applied?: string;
  };
}

export interface RawLead {
  id: string;
  name?: string;
  email?: string;
  phone?: string;
  company?: string;
  domain?: string;
  url?: string;
  source_id?: string;
  permission_ref?: string;
  created_at: string;
  updated_at: string;
}

export interface LeadRecord extends RawLead {
  stage: PipelineStage;
  risk_level: RiskLevel;
  risk_score?: number;
  email_validate?: EmailValidateResult;
  domain_intel?: DomainIntelResult;
  phone_validate?: PhoneValidateResult;
  social_footprint?: SocialFootprintResult;
  extraction?: ExtractionResult;
  audit_events: AuditEvent[];
}

export type LeadSummary = Omit<LeadRecord, "audit_events">;

export interface AuditEvent {
  id: string;
  lead_id: string;
  run_id?: string;
  module: ModuleName | "pipeline";
  tool: string;
  checked_at: string;
  status: "ok" | "unknown" | "skipped";
  legal_basis: string;
  subject?: {
    email?: string;
    domain?: string;
    phone_redacted?: string;
    handle?: string;
    url?: string;
  };
  raw_stderr_json?: string;
}

export type ModuleDevStatus =
  | "available"
  | "in_development"
  | "planned"
  | "not_configured";

export interface ModuleConfigSchemaField {
  key: string;
  label: string;
  type: "string" | "secret" | "number" | "boolean";
  required: boolean;
  placeholder?: string;
}

export interface ModuleInfo {
  name: ModuleName;
  display_name: string;
  category: "ingest" | "enrich" | "validate";
  dev_status: ModuleDevStatus;
  namespaced_key: string;
  backing_tools: string[];
  description: string;
  min_input_field: string;
  risk_level_note: string;
  last_run_at?: string;
  error_rate_24h?: number;
  config_schema?: ModuleConfigSchemaField[];
}

export interface ModuleDetail extends ModuleInfo {
  docs?: string;
}

export interface PipelineRun {
  id: string;
  type: "single" | "batch";
  started_at: string;
  finished_at?: string;
  status: "running" | "completed" | "failed" | "partial";
  lead_ids: string[];
  modules_executed: string[];
  audit_event_ids: string[];
  legal_basis: string;
  permission_refs: string[];
  error?: string;
}

export interface ComplianceRule {
  id: number;
  title: string;
  summary: string;
}

export interface ComplianceRiskRule {
  category: string;
  risk_level: string;
  notes: string;
}

export interface ChecklistItem {
  id: string;
  label: string;
}

export interface ComplianceSummary {
  hard_rules: ComplianceRule[];
  risk_table: ComplianceRiskRule[];
  checklist: ChecklistItem[];
  exclusions: string[];
}

export interface RunModulesRequest {
  modules: ModuleName[];
  permission_ref?: string;
  legal_basis?: string;
}

export interface PipelineRunRequest {
  lead_ids: string[];
  modules: ModuleName[];
  permission_ref?: string;
  legal_basis?: string;
}

export interface ApiError {
  code: string;
  message: string;
}

export interface ListMeta {
  page: number;
  page_size: number;
  total: number;
}

export interface ApiResponse<T> {
  data: T;
  meta?: ListMeta;
  error?: ApiError;
}

export interface ListResponse<T> {
  data: T;
  meta: ListMeta;
}
