"use client";

import { useQuery, useMutation, useQueryClient, UseQueryResult } from "@tanstack/react-query";
import {
  apiGet,
  apiPost,
  ApiClientError,
} from "./client";
import {
  AuditEvent,
  ComplianceSummary,
  LeadRecord,
  LeadSummary,
  ListMeta,
  ListResponse,
  ModuleDetail,
  ModuleInfo,
  PipelineRun,
  PipelineRunRequest,
  ReadinessReport,
  ExportResponse,
  StageTransitionRequest,
  RunModulesRequest,
} from "./types";

export type LeadSearchParams = {
  stage?: string;
  risk?: string;
  module_status?: string;
  q?: string;
  page?: number;
  page_size?: number;
};

const LEADS_KEY = "leads";
const LEAD_KEY = "lead";
const MODULES_KEY = "modules";
const AUDIT_KEY = "audit";
const RUNS_KEY = "runs";
const COMPLIANCE_KEY = "compliance";

export function useLeads(params: LeadSearchParams = {}) {
  return useQuery<ListResponse<LeadSummary[]>, ApiClientError>({
    queryKey: [LEADS_KEY, params],
    queryFn: async () => {
      const res = await apiGet<LeadSummary[]>("/api/leads", params);
      const meta: ListMeta = res.meta ?? { page: 1, page_size: 25, total: 0 };
      return { data: res.data, meta };
    },
  });
}

export function useLead(id?: string) {
  return useQuery<LeadRecord>({
    queryKey: [LEAD_KEY, id],
    queryFn: async () => {
      if (!id) throw new ApiClientError("missing_id", "No lead id provided", 400);
      const res = await apiGet<LeadRecord>(`/api/leads/${id}`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useModules() {
  return useQuery<ModuleInfo[]>({
    queryKey: [MODULES_KEY],
    queryFn: async () => {
      const res = await apiGet<ModuleInfo[]>("/api/modules");
      return res.data;
    },
  });
}

export function useModule(name?: string) {
  return useQuery<ModuleDetail>({
    queryKey: [MODULES_KEY, name],
    queryFn: async () => {
      if (!name) throw new ApiClientError("missing_name", "No module name provided", 400);
      const res = await apiGet<ModuleDetail>(`/api/modules/${name}`);
      return res.data;
    },
    enabled: !!name,
  });
}

export function useAudit(params: { module?: string; status?: string; page?: number; page_size?: number } = {}) {
  return useQuery<ListResponse<AuditEvent[]>, ApiClientError>({
    queryKey: [AUDIT_KEY, params],
    queryFn: async () => {
      const res = await apiGet<AuditEvent[]>("/api/audit", params);
      const meta: ListMeta = res.meta ?? { page: 1, page_size: 25, total: 0 };
      return { data: res.data, meta };
    },
  });
}

export function useRuns(params: { page?: number; page_size?: number } = {}) {
  return useQuery<ListResponse<PipelineRun[]>, ApiClientError>({
    queryKey: [RUNS_KEY, params],
    queryFn: async () => {
      const res = await apiGet<PipelineRun[]>("/api/runs", params);
      const meta: ListMeta = res.meta ?? { page: 1, page_size: 25, total: 0 };
      return { data: res.data, meta };
    },
  });
}

export function useRun(id?: string) {
  return useQuery<PipelineRun>({
    queryKey: [RUNS_KEY, id],
    queryFn: async () => {
      if (!id) throw new ApiClientError("missing_id", "No run id provided", 400);
      const res = await apiGet<PipelineRun>(`/api/runs/${id}`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useComplianceSummary() {
  return useQuery<ComplianceSummary>({
    queryKey: [COMPLIANCE_KEY],
    queryFn: async () => {
      const res = await apiGet<ComplianceSummary>("/api/compliance/summary");
      return res.data;
    },
  });
}

export type CreateLeadInput = Partial<LeadRecord>;

export function useCreateLead() {
  const queryClient = useQueryClient();
  return useMutation<LeadSummary, ApiClientError, CreateLeadInput>({
    mutationFn: async (payload) => {
      const res = await apiPost<LeadSummary>("/api/leads", payload);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [LEADS_KEY] });
    },
  });
}

export function useRunLeadModules() {
  const queryClient = useQueryClient();
  return useMutation<LeadRecord, ApiClientError, { id: string; body: RunModulesRequest }>({
    mutationFn: async ({ id, body }) => {
      const res = await apiPost<LeadRecord>(`/api/leads/${id}/run`, body);
      return res.data;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: [LEAD_KEY, variables.id] });
      queryClient.invalidateQueries({ queryKey: [LEADS_KEY] });
      queryClient.invalidateQueries({ queryKey: [AUDIT_KEY] });
      queryClient.invalidateQueries({ queryKey: [RUNS_KEY] });
    },
  });
}

export function useRunPipeline() {
  const queryClient = useQueryClient();
  return useMutation<{ accepted: true; run_id: string }, ApiClientError, PipelineRunRequest>({
    mutationFn: async (body) => {
      const res = await apiPost<{ accepted: true; run_id: string }>("/api/pipelines/run", body);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [RUNS_KEY] });
      queryClient.invalidateQueries({ queryKey: [LEADS_KEY] });
    },
  });
}

export function useLeadReadiness(id?: string) {
  return useQuery<ReadinessReport, ApiClientError>({
    queryKey: ["readiness", id],
    queryFn: async () => {
      if (!id) throw new ApiClientError("missing_id", "No lead id provided", 400);
      const res = await apiGet<ReadinessReport>(`/api/leads/${id}/readiness`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function usePromoteLead() {
  const queryClient = useQueryClient();
  return useMutation<LeadRecord, ApiClientError, { id: string; body: StageTransitionRequest }>({
    mutationFn: async ({ id, body }) => {
      const res = await apiPost<LeadRecord>(`/api/leads/${id}/promote`, body);
      return res.data;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: [LEAD_KEY, variables.id] });
      queryClient.invalidateQueries({ queryKey: [LEADS_KEY] });
      queryClient.invalidateQueries({ queryKey: ["readiness", variables.id] });
    },
  });
}

export function useDemoteLead() {
  const queryClient = useQueryClient();
  return useMutation<LeadRecord, ApiClientError, { id: string; body: StageTransitionRequest }>({
    mutationFn: async ({ id, body }) => {
      const res = await apiPost<LeadRecord>(`/api/leads/${id}/demote`, body);
      return res.data;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: [LEAD_KEY, variables.id] });
      queryClient.invalidateQueries({ queryKey: [LEADS_KEY] });
      queryClient.invalidateQueries({ queryKey: ["readiness", variables.id] });
    },
  });
}

export function useExportLead() {
  return useMutation<ExportResponse, ApiClientError, string>({
    mutationFn: async (id) => {
      const res = await apiGet<ExportResponse>(`/api/leads/${id}/export`);
      return res.data;
    },
  });
}

export function useApiHealth(): UseQueryResult<unknown[], ApiClientError> {
  return useQuery<unknown[], ApiClientError>({
    queryKey: ["api-health"],
    queryFn: async () => {
      const res = await apiGet<unknown[]>("/api/leads", { page_size: 1 });
      return res.data;
    },
    retry: 1,
    refetchInterval: 30_000,
    refetchOnWindowFocus: false,
  });
}
