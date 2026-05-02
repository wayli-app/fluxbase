import type { EmailProviderSettings } from "@nimbleflux/fluxbase-sdk";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ProviderFormState } from "./types";

interface CommonFieldsProps {
  formState: ProviderFormState;
  settings: EmailProviderSettings | undefined;
  onFormChange: (
    field: keyof ProviderFormState,
    value: string | boolean,
  ) => void;
}

export function CommonFields({
  formState,
  settings,
  onFormChange,
}: CommonFieldsProps) {
  const isOverridden = (field: string) =>
    settings?._overrides?.[field]?.is_overridden ?? false;
  const getEnvVar = (field: string) =>
    settings?._overrides?.[field]?.env_var || "";

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <div className="space-y-2">
        <Label htmlFor="from_address">
          From Email
          {isOverridden("from_address") && (
            <Badge variant="outline" className="ml-2 text-xs">
              ENV: {getEnvVar("from_address")}
            </Badge>
          )}
        </Label>
        <Input
          id="from_address"
          type="email"
          placeholder="noreply@example.com"
          value={formState.from_address}
          onChange={(e) => onFormChange("from_address", e.target.value)}
          readOnly={isOverridden("from_address")}
          autoComplete="off"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="from_name">
          From Name
          {isOverridden("from_name") && (
            <Badge variant="outline" className="ml-2 text-xs">
              ENV: {getEnvVar("from_name")}
            </Badge>
          )}
        </Label>
        <Input
          id="from_name"
          placeholder="My App"
          value={formState.from_name}
          onChange={(e) => onFormChange("from_name", e.target.value)}
          readOnly={isOverridden("from_name")}
          autoComplete="off"
        />
      </div>
    </div>
  );
}
