import { Eye, EyeOff, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import type { ProviderFormProps } from "./types";

export function SesForm({
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
        <Label htmlFor="ses_access_key">
          Access Key
          {isOverridden("ses_access_key") && (
            <Badge variant="outline" className="ml-2 text-xs">
              ENV: {getEnvVar("ses_access_key")}
            </Badge>
          )}
          {settings?.ses_access_key_set && !isOverridden("ses_access_key") && (
            <Badge variant="secondary" className="ml-2 text-xs">
              <CheckCircle2 className="mr-1 h-3 w-3" />
              Set
            </Badge>
          )}
        </Label>
        <div className="relative">
          <Input
            id="ses_access_key"
            type={showPassword ? "text" : "password"}
            placeholder={
              settings?.ses_access_key_set ? "••••••••" : "AKIAXXXXX"
            }
            value={formState.ses_access_key}
            onChange={(e) => onFormChange("ses_access_key", e.target.value)}
            readOnly={isOverridden("ses_access_key")}
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
        <Label htmlFor="ses_secret_key">
          Secret Key
          {isOverridden("ses_secret_key") && (
            <Badge variant="outline" className="ml-2 text-xs">
              ENV: {getEnvVar("ses_secret_key")}
            </Badge>
          )}
          {settings?.ses_secret_key_set && !isOverridden("ses_secret_key") && (
            <Badge variant="secondary" className="ml-2 text-xs">
              <CheckCircle2 className="mr-1 h-3 w-3" />
              Set
            </Badge>
          )}
        </Label>
        <div className="relative">
          <Input
            id="ses_secret_key"
            type={showPassword ? "text" : "password"}
            placeholder={
              settings?.ses_secret_key_set ? "••••••••" : "Secret key"
            }
            value={formState.ses_secret_key}
            onChange={(e) => onFormChange("ses_secret_key", e.target.value)}
            readOnly={isOverridden("ses_secret_key")}
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
        <Label htmlFor="ses_region">
          Region
          {isOverridden("ses_region") && (
            <Badge variant="outline" className="ml-2 text-xs">
              ENV: {getEnvVar("ses_region")}
            </Badge>
          )}
        </Label>
        <Input
          id="ses_region"
          placeholder="us-east-1"
          value={formState.ses_region}
          onChange={(e) => onFormChange("ses_region", e.target.value)}
          readOnly={isOverridden("ses_region")}
          autoComplete="off"
        />
      </div>
    </div>
  );
}
