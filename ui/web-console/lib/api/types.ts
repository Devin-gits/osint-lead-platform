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

export interface DomainIntelResult extends LeadResultMeta {
  status: ModuleStatus;
  web_check?: { status: ModuleStatus; summary?: string };
  harvester?: { status: ModuleStatus; emails_found?: number };
}

export interface PhoneValidateResult extends LeadResultMeta {
  status: ModuleStatus;
  e164?: string;
  region?: string;
  line_type?: string;
  carrier?: string;
  risk_flags?: string[];
}

export interface SocialFootprintMatch {
  platform: string;
  handle: string;
  confidence: number;
  url?: string;
}

export interface SocialFootprintResult extends LeadResultMeta {
  status: ModuleStatus;
  matches?: SocialFootprintMatch[];
  summary?: string;
}

export interface RawLead {
  id: string;
  name?: string;
  email?: string;
  phone?: string;
  company?: string;
  domain?: string;
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
