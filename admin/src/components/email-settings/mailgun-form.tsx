import { Eye, EyeOff, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import type { ProviderFormProps } from "./types";

export function MailgunForm({
  formState,
  settings,
  showPassword,
  onFormChange,
  onTogglePassword,
}: ProviderFormProps) {
  const isOverridden = (field: string) =>
    settings?._overrides?.[field]?.is_overridden ?? false;
  const getEnvVar = (field: string) =>
    settings?._overrides?.[field]?.env_var || "";

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <div className="space-y-2">
        <Label htmlFor="mailgun_api_key">
          API Key
          {isOverridden("mailgun_api_key") && (
            <Badge variant="outline" className="ml-2 text-xs">
              ENV: {getEnvVar("mailgun_api_key")}
            </Badge>
          )}
          {settings?.mailgun_api_key_set &&
            !isOverridden("mailgun_api_key") && (
              <Badge variant="secondary" className="ml-2 text-xs">
                <CheckCircle2 className="mr-1 h-3 w-3" />
                Set
              </Badge>
            )}
        </Label>
        <div className="relative">
          <Input
            id="mailgun_api_key"
            type={showPassword ? "text" : "password"}
            placeholder={
              settings?.mailgun_api_key_set ? "••••••••" : "key-xxxxx"
            }
            value={formState.mailgun_api_key}
            onChange={(e) => onFormChange("mailgun_api_key", e.target.value)}
            readOnly={isOverridden("mailgun_api_key")}
            autoComplete="off"
          />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="absolute top-0 right-0 h-full px-3 py-2 hover:bg-transparent"
            onClick={onTogglePassword}
          >
            {showPassword ? (
              <EyeOff className="h-4 w-4" />
            ) : (
              <Eye className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="mailgun_domain">
          Domain
          {isOverridden("mailgun_domain") && (
            <Badge variant="outline" className="ml-2 text-xs">
              ENV: {getEnvVar("mailgun_domain")}
            </Badge>
          )}
        </Label>
        <Input
          id="mailgun_domain"
          placeholder="mg.example.com"
          value={formState.mailgun_domain}
          onChange={(e) => onFormChange("mailgun_domain", e.target.value)}
          readOnly={isOverridden("mailgun_domain")}
          autoComplete="off"
        />
      </div>
    </div>
  );
}
