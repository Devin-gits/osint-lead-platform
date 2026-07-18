import { CreateLeadFlow } from "@/components/leads/CreateLeadFlow";

export const metadata = {
  title: "Create lead | OSINT Lead Console",
};

export default function CreateLeadPage() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground md:text-3xl">
        Create lead
      </h1>
      <CreateLeadFlow />
    </div>
  );
}
