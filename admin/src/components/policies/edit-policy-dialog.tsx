import { useState } from "react";
import { Loader2 } from "lucide-react";
import type { RLSPolicy } from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

interface EditPolicyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  policy: RLSPolicy;
  onSubmit: (data: {
    roles?: string[];
    using?: string | null;
    with_check?: string | null;
  }) => void;
  isLoading: boolean;
}

export function EditPolicyDialog({
  open,
  onOpenChange,
  policy,
  onSubmit,
  isLoading,
}: EditPolicyDialogProps) {
  const [formData, setFormData] = useState({
    roles: policy.roles,
    using: policy.using || "",
    with_check: policy.with_check || "",
  });

  const [prevPolicy, setPrevPolicy] = useState(policy);
  if (policy !== prevPolicy) {
    setFormData({
      roles: policy.roles,
      using: policy.using || "",
      with_check: policy.with_check || "",
    });
    setPrevPolicy(policy);
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      roles: formData.roles,
      using: formData.using || null,
      with_check: formData.with_check || null,
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Edit Policy</DialogTitle>
          <DialogDescription>
            Edit the policy &quot;{policy.policy_name}&quot; on {policy.schema}.
            {policy.table}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Policy Name</Label>
              <Input value={policy.policy_name} readOnly className="bg-muted" />
              <p className="text-muted-foreground text-xs">
                Policy name cannot be changed
              </p>
            </div>
            <div className="space-y-2">
              <Label>Command</Label>
              <Input value={policy.command} readOnly className="bg-muted" />
              <p className="text-muted-foreground text-xs">
                Command type cannot be changed
              </p>
            </div>
          </div>

          <div className="space-y-2">
            <Label>Mode</Label>
            <Input value={policy.permissive} readOnly className="bg-muted" />
            <p className="text-muted-foreground text-xs">
              Permissive/restrictive mode cannot be changed
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-roles">Roles (comma-separated)</Label>
            <Input
              id="edit-roles"
              value={formData.roles.join(", ")}
              onChange={(e) =>
                setFormData({
                  ...formData,
                  roles: e.target.value
                    .split(",")
                    .map((r) => r.trim())
                    .filter(Boolean),
                })
              }
              placeholder="e.g., authenticated, anon"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-using">USING Expression</Label>
            <Textarea
              id="edit-using"
              value={formData.using}
              onChange={(e) =>
                setFormData({ ...formData, using: e.target.value })
              }
              placeholder="e.g., auth.uid() = user_id"
              className="min-h-[80px] font-mono text-sm"
            />
            <p className="text-muted-foreground text-xs">
              Controls which rows can be selected, updated, or deleted
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-with-check">WITH CHECK Expression</Label>
            <Textarea
              id="edit-with-check"
              value={formData.with_check}
              onChange={(e) =>
                setFormData({ ...formData, with_check: e.target.value })
              }
              placeholder="e.g., auth.uid() = user_id"
              className="min-h-[80px] font-mono text-sm"
            />
            <p className="text-muted-foreground text-xs">
              Controls which rows can be inserted or updated (new values)
            </p>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Saving...
                </>
              ) : (
                "Save Changes"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
