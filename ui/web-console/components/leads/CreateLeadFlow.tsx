"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft, ArrowRight, Clock, Loader2 } from "lucide-react";
import { useCreateLead, useModules, useRunLeadModules } from "@/lib/api/hooks";
import { useToast } from "@/components/providers/ToastProvider";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils/cn";
import { PermissionReferenceField } from "./PermissionReferenceField";
import type { ModuleName } from "@/lib/api/types";

function durationHint(name: ModuleName): string {
  switch (name) {
    case "email-validate":
    case "phone-validate":
      return "Fast";
    case "domain-intel":
      return "May take longer";
    case "social-footprint":
      return "Rate-limited / potentially slow; no ETA";
    default:
      return "";
  }
}

function devStatusLabel(status: string): string {
  switch (status) {
    case "available":
      return "Available";
    case "in_development":
      return "In development";
    case "planned":
      return "Planned";
    case "not_configured":
      return "Not configured";
    default:
      return status;
  }
}

function devStatusVariant(
  status: string
): "success" | "warning" | "muted" | "outline" {
  switch (status) {
    case "available":
      return "success";
    case "in_development":
      return "warning";
    case "planned":
      return "muted";
    default:
      return "outline";
  }
}

interface FormState {
  permission_ref: string;
  name: string;
  email: string;
  phone: string;
  domain: string;
  company: string;
}

interface FormErrors {
  permission_ref?: string;
  contact?: string;
}

function hasContact(values: FormState): boolean {
  return [
    values.email.trim(),
    values.phone.trim(),
    values.domain.trim(),
    values.company.trim(),
  ].some(Boolean);
}

function validate(values: FormState): FormErrors {
  const errors: FormErrors = {};
  if (!values.permission_ref.trim()) {
    errors.permission_ref =
      "Permission reference is required before module selection.";
  }
  if (!hasContact(values)) {
    errors.contact =
      "Provide at least one contact point: email, phone, domain, or company.";
  }
  return errors;
}

export function CreateLeadFlow() {
  const router = useRouter();
  const { addToast } = useToast();
  const [step, setStep] = useState<1 | 2>(1);
  const [form, setForm] = useState<FormState>({
    permission_ref: "",
    name: "",
    email: "",
    phone: "",
    domain: "",
    company: "",
  });
  const [errors, setErrors] = useState<FormErrors>({});
  const [touched, setTouched] = useState(false);
  const [selected, setSelected] = useState<Set<ModuleName>>(new Set());
  const [runError, setRunError] = useState<{
    id: string;
    message: string;
  } | null>(null);
  const presetRef = useRef(false);

  const create = useCreateLead();
  const runModules = useRunLeadModules();
  const {
    data: modules,
    isLoading: modulesLoading,
    error: modulesError,
  } = useModules();

  useEffect(() => {
    if (presetRef.current || !modules) return;
    presetRef.current = true;
    const email = modules.find(
      (m) => m.name === "email-validate" && m.dev_status === "available"
    );
    if (email) {
      setSelected(new Set(["email-validate"]));
    }
  }, [modules]);

  const updateForm = (field: keyof FormState, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    if (touched) {
      setErrors(validate({ ...form, [field]: value }));
    }
  };

  const handleContinue = () => {
    setTouched(true);
    const validation = validate(form);
    setErrors(validation);
    if (Object.keys(validation).length === 0) {
      setStep(2);
    }
  };

  const toggleModule = (name: ModuleName) => {
    const next = new Set(selected);
    if (next.has(name)) next.delete(name);
    else next.add(name);
    setSelected(next);
  };

  const handleSubmit = async (modulesToRun: ModuleName[]) => {
    if (runError) return; // Lead already created; prevent duplicate
    setRunError(null);
    const validation = validate(form);
    if (Object.keys(validation).length > 0) {
      setErrors(validation);
      setStep(1);
      return;
    }

    try {
      const created = await create.mutateAsync({
        ...form,
        source_id: "",
      });

      if (modulesToRun.length > 0) {
        try {
          await runModules.mutateAsync({
            id: created.id,
            body: {
              modules: modulesToRun,
              permission_ref: form.permission_ref.trim(),
            },
          });
        } catch (runErr) {
          setRunError({
            id: created.id,
            message:
              runErr instanceof Error
                ? runErr.message
                : "Module run failed after the lead was created.",
          });
          return;
        }
      }

      addToast("Lead created", "success");
      router.push(`/leads/${created.id}`);
    } catch {
      // create.error is surfaced below the button
    }
  };

  const isSubmitting = create.isPending || runModules.isPending;

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div className="flex items-center gap-2 text-sm text-foreground-muted">
        <span
          className={cn(
            "flex h-6 w-6 items-center justify-center rounded-full text-xs font-medium",
            step >= 1
              ? "bg-primary text-background"
              : "bg-surface-elevated text-foreground-secondary"
          )}
          aria-current={step === 1 ? "step" : undefined}
        >
          1
        </span>
        <span className={step === 1 ? "text-foreground" : undefined}>
          Lead & permission
        </span>
        <span className="mx-2 h-px w-8 bg-border" aria-hidden="true" />
        <span
          className={cn(
            "flex h-6 w-6 items-center justify-center rounded-full text-xs font-medium",
            step >= 2
              ? "bg-primary text-background"
              : "bg-surface-elevated text-foreground-secondary"
          )}
          aria-current={step === 2 ? "step" : undefined}
        >
          2
        </span>
        <span className={step === 2 ? "text-foreground" : undefined}>
          Module selection
        </span>
      </div>

      {step === 1 ? (
        <Card className="p-6">
          <h1 className="text-xl font-semibold text-foreground">
            Create a permissioned lead
          </h1>
          <p className="mt-1 text-sm text-foreground-muted">
            Start with the lawful basis, then add the contact information you
            have.
          </p>

          <div className="mt-6 space-y-6">
            <PermissionReferenceField
              value={form.permission_ref}
              onChange={(value) => updateForm("permission_ref", value)}
              error={touched ? errors.permission_ref : undefined}
              disabled={create.isPending}
            />

            <div className="grid gap-4 sm:grid-cols-2">
              <Input
                label="Name"
                value={form.name}
                onChange={(e) => updateForm("name", e.target.value)}
                disabled={create.isPending}
              />
              <Input
                label="Email"
                type="email"
                value={form.email}
                onChange={(e) => updateForm("email", e.target.value)}
                disabled={create.isPending}
              />
              <Input
                label="Phone"
                value={form.phone}
                onChange={(e) => updateForm("phone", e.target.value)}
                disabled={create.isPending}
              />
              <Input
                label="Domain"
                value={form.domain}
                onChange={(e) => updateForm("domain", e.target.value)}
                disabled={create.isPending}
              />
              <Input
                label="Company"
                value={form.company}
                onChange={(e) => updateForm("company", e.target.value)}
                disabled={create.isPending}
              />
            </div>

            {touched && errors.contact && (
              <p className="text-sm text-danger" role="alert">
                {errors.contact}
              </p>
            )}

            {create.error && (
              <div
                className="rounded-md border border-danger/20 bg-danger/10 p-3 text-sm text-danger"
                role="alert"
              >
                {create.error.message || "Failed to create lead."}
              </div>
            )}

            <div className="flex justify-end">
              <Button
                onClick={handleContinue}
                disabled={create.isPending}
                className="min-w-[8rem]"
              >
                {create.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating…
                  </>
                ) : (
                  <>
                    Continue
                    <ArrowRight className="ml-2 h-4 w-4" />
                  </>
                )}
              </Button>
            </div>
          </div>
        </Card>
      ) : (
        <Card className="p-6">
          <button
            onClick={() => setStep(1)}
            className="mb-2 inline-flex items-center text-sm text-foreground-secondary hover:text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
          >
            <ArrowLeft className="mr-1 h-4 w-4" />
            Back to lead details
          </button>
          <h1 className="text-xl font-semibold text-foreground">
            Select modules
          </h1>
          <p className="mt-1 text-sm text-foreground-muted">
            Choose which available checks to run now. Module runs can also be
            started later from the lead detail page.
          </p>

          {modulesError ? (
            <div
              className="mt-4 rounded-md border border-warning/20 bg-warning/10 p-3 text-sm text-warning"
              role="alert"
            >
              <p className="font-medium">Could not load modules</p>
              <p className="mt-1">{modulesError.message}</p>
              <p className="mt-1">
                You can still create the lead without running any modules.
              </p>
            </div>
          ) : (
            <div className="mt-6 space-y-3">
              {modulesLoading && (
                <p className="text-sm text-foreground-muted">
                  Loading modules…
                </p>
              )}
              {!modulesLoading && modules?.length === 0 && (
                <p className="text-sm text-foreground-muted">
                  No modules available.
                </p>
              )}
              {modules?.map((module) => {
                const available = module.dev_status === "available";
                const checked = selected.has(module.name);
                return (
                  <label
                    key={module.name}
                    className={cn(
                      "flex items-start gap-3 rounded-lg border p-4 transition-colors",
                      available
                        ? "cursor-pointer hover:border-primary/30"
                        : "cursor-not-allowed opacity-60",
                      checked
                        ? "border-primary bg-primary/[0.04]"
                        : "border-border bg-surface"
                    )}
                  >
                    <input
                      type="checkbox"
                      className="mt-1 h-4 w-4 rounded border-border text-primary focus:ring-primary/50"
                      checked={checked}
                      disabled={!available}
                      onChange={() => toggleModule(module.name)}
                      aria-describedby={`${module.name}-hint`}
                    />
                    <div className="flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium text-foreground">
                          {module.display_name}
                        </span>
                        <Badge variant={devStatusVariant(module.dev_status)}>
                          {devStatusLabel(module.dev_status)}
                        </Badge>
                        {available && (
                          <span className="inline-flex items-center gap-1 text-xs text-foreground-muted">
                            <Clock
                              className="h-3 w-3"
                              aria-hidden="true"
                            />
                            {durationHint(module.name)}
                          </span>
                        )}
                      </div>
                      <p className="mt-1 text-sm text-foreground-muted">
                        {module.description}
                      </p>
                      <p
                        id={`${module.name}-hint`}
                        className="mt-1 text-xs text-foreground-muted"
                      >
                        Minimum input: {module.min_input_field}
                      </p>
                    </div>
                  </label>
                );
              })}
            </div>
          )}

          {runError && (
            <div
              className="mt-4 rounded-md border border-danger/20 bg-danger/10 p-3 text-sm text-danger"
              role="alert"
            >
              <p className="font-medium">Lead created, but module run failed</p>
              <p className="mt-1">{runError.message}</p>
              <p className="mt-2">
                <Link
                  href={`/leads/${runError.id}`}
                  className="font-medium underline hover:no-underline"
                >
                  Open lead detail
                </Link>
                {" "}to retry or review.
              </p>
            </div>
          )}

          {runError ? (
            <div className="mt-6 flex justify-end">
              <Link
                href={`/leads/${runError.id}`}
                className="inline-flex items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-background transition-colors hover:brightness-110 focus:outline-none focus:ring-2 focus:ring-primary/50"
              >
                Open lead detail
              </Link>
            </div>
          ) : (
            <div className="mt-6 flex justify-end gap-3">
              <Button
                variant="ghost"
                onClick={() => handleSubmit([])}
                disabled={isSubmitting}
              >
                Create without running modules
              </Button>
              <Button
                onClick={() => handleSubmit(Array.from(selected))}
                disabled={isSubmitting || modulesError !== null}
              >
                {isSubmitting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Working…
                  </>
                ) : (
                  <>
                    Create lead
                    {selected.size > 0 &&
                      ` and run ${selected.size} module${selected.size === 1 ? "" : "s"}`}
                  </>
                )}
              </Button>
            </div>
          )}
        </Card>
      )}
    </div>
  );
}
