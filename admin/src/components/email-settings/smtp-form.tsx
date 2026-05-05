import { Eye, EyeOff, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import type { ProviderFormProps } from "./types";

export function SmtpForm({
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
    <>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="smtp_host">
            SMTP Host
            {isOverridden("smtp_host") && (
              <Badge variant="outline" className="ml-2 text-xs">
                ENV: {getEnvVar("smtp_host")}
              </Badge>
            )}
          </Label>
          <Input
            id="smtp_host"
            placeholder="smtp.example.com"
            value={formState.smtp_host}
            onChange={(e) => onFormChange("smtp_host", e.target.value)}
            readOnly={isOverridden("smtp_host")}
            autoComplete="off"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="smtp_port">
            SMTP Port
            {isOverridden("smtp_port") && (
              <Badge variant="outline" className="ml-2 text-xs">
                ENV: {getEnvVar("smtp_port")}
              </Badge>
            )}
          </Label>
          <Input
            id="smtp_port"
            type="number"
            placeholder="587"
            value={formState.smtp_port}
            onChange={(e) => onFormChange("smtp_port", e.target.value)}
            readOnly={isOverridden("smtp_port")}
            autoComplete="off"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="smtp_username">
            Username
            {isOverridden("smtp_username") && (
              <Badge variant="outline" className="ml-2 text-xs">
                ENV: {getEnvVar("smtp_username")}
              </Badge>
            )}
          </Label>
          <Input
            id="smtp_username"
            placeholder="username"
            value={formState.smtp_username}
            onChange={(e) => onFormChange("smtp_username", e.target.value)}
            readOnly={isOverridden("smtp_username")}
            autoComplete="off"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="smtp_password">
            Password
            {isOverridden("smtp_password") && (
              <Badge variant="outline" className="ml-2 text-xs">
                ENV: {getEnvVar("smtp_password")}
              </Badge>
            )}
            {settings?.smtp_password_set && !isOverridden("smtp_password") && (
              <Badge variant="secondary" className="ml-2 text-xs">
                <CheckCircle2 className="mr-1 h-3 w-3" />
                Set
              </Badge>
            )}
          </Label>
          <div className="relative">
            <Input
              id="smtp_password"
              type={showPassword ? "text" : "password"}
              placeholder={
                settings?.smtp_password_set ? "••••••••" : "Enter password"
              }
              value={formState.smtp_password}
              onChange={(e) => onFormChange("smtp_password", e.target.value)}
              readOnly={isOverridden("smtp_password")}
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
      </div>
      <div className="flex items-center space-x-2">
        <Switch
          id="smtp_tls"
          checked={formState.smtp_tls}
          onCheckedChange={(checked) => onFormChange("smtp_tls", checked)}
          disabled={isOverridden("smtp_tls")}
        />
        <Label htmlFor="smtp_tls">
          Enable TLS
          {isOverridden("smtp_tls") && (
            <Badge variant="outline" className="ml-2 text-xs">
              ENV: {getEnvVar("smtp_tls")}
            </Badge>
          )}
        </Label>
      </div>
    </>
  );
}
