import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2, Loader2 } from "lucide-react";
import { toast } from "sonner";
import {
  knowledgeBasesApi,
  type ChatbotKnowledgeBaseLink,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";

interface ChatbotKBLinksSectionProps {
  chatbotId: string;
}

export function ChatbotKBLinksSection({ chatbotId }: ChatbotKBLinksSectionProps) {
  const queryClient = useQueryClient();
  const [selectedKbId, setSelectedKbId] = useState<string>("");

  const { data: links = [], isLoading: linksLoading } = useQuery({
    queryKey: ["chatbot-kb-links", chatbotId],
    queryFn: () => knowledgeBasesApi.listChatbotLinks(chatbotId),
  });

  const { data: allKbs = [] } = useQuery({
    queryKey: ["knowledge-bases"],
    queryFn: () => knowledgeBasesApi.list(),
  });

  const availableKbs = allKbs.filter(
    (kb) => !links.some((link) => link.knowledge_base_id === kb.id),
  );

  const linkMutation = useMutation({
    mutationFn: (kbId: string) =>
      knowledgeBasesApi.linkToChatbot(chatbotId, kbId),
    onSuccess: () => {
      toast.success("Knowledge base linked");
      queryClient.invalidateQueries({
        queryKey: ["chatbot-kb-links", chatbotId],
      });
      setSelectedKbId("");
    },
    onError: () => toast.error("Failed to link knowledge base"),
  });

  const updateMutation = useMutation({
    mutationFn: ({
      kbId,
      data,
    }: {
      kbId: string;
      data: Partial<ChatbotKnowledgeBaseLink>;
    }) => knowledgeBasesApi.updateChatbotLink(chatbotId, kbId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["chatbot-kb-links", chatbotId],
      });
    },
    onError: () => toast.error("Failed to update knowledge base settings"),
  });

  const unlinkMutation = useMutation({
    mutationFn: (kbId: string) =>
      knowledgeBasesApi.unlinkFromChatbot(chatbotId, kbId),
    onSuccess: () => {
      toast.success("Knowledge base unlinked");
      queryClient.invalidateQueries({
        queryKey: ["chatbot-kb-links", chatbotId],
      });
    },
    onError: () => toast.error("Failed to unlink knowledge base"),
  });

  const kbNameFor = (kbId: string): string => {
    const kb = allKbs.find((k) => k.id === kbId);
    return kb?.name || "Unknown";
  };

  if (linksLoading) {
    return (
      <div className="flex items-center gap-2 text-muted-foreground text-sm">
        <Loader2 className="h-4 w-4 animate-spin" />
        Loading knowledge bases...
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Existing links */}
      {links.length === 0 ? (
        <p className="text-muted-foreground text-sm">
          No knowledge bases linked. This chatbot will not use RAG.
        </p>
      ) : (
        <div className="space-y-3">
          {links.map((link) => (
            <KBLinkRow
              key={link.knowledge_base_id}
              link={link}
              kbName={kbNameFor(link.knowledge_base_id)}
              onUpdate={(data) =>
                updateMutation.mutate({ kbId: link.knowledge_base_id, data })
              }
              onUnlink={() =>
                unlinkMutation.mutate(link.knowledge_base_id)
              }
              isUpdating={updateMutation.isPending}
              isUnlinking={unlinkMutation.isPending}
            />
          ))}
        </div>
      )}

      {/* Add new link */}
      {availableKbs.length > 0 && (
        <div className="flex items-end gap-2 border-t pt-3">
          <div className="flex-1 space-y-2">
            <Label className="text-xs">Link Knowledge Base</Label>
            <Select
              value={selectedKbId}
              onValueChange={setSelectedKbId}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select a knowledge base..." />
              </SelectTrigger>
              <SelectContent>
                {availableKbs.map((kb) => (
                  <SelectItem key={kb.id} value={kb.id}>
                    {kb.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={!selectedKbId || linkMutation.isPending}
            onClick={() => selectedKbId && linkMutation.mutate(selectedKbId)}
          >
            {linkMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <>
                <Plus className="mr-1 h-4 w-4" />
                Link
              </>
            )}
          </Button>
        </div>
      )}
    </div>
  );
}

interface KBLinkRowProps {
  link: ChatbotKnowledgeBaseLink;
  kbName: string;
  onUpdate: (data: Partial<ChatbotKnowledgeBaseLink>) => void;
  onUnlink: () => void;
  isUpdating: boolean;
  isUnlinking: boolean;
}

function KBLinkRow({
  link,
  kbName,
  onUpdate,
  onUnlink,
  isUnlinking,
}: KBLinkRowProps) {
  return (
    <div className="space-y-2 rounded-lg border p-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="font-medium text-sm">{kbName}</span>
          {!link.enabled && (
            <Badge variant="secondary" className="text-xs">
              Disabled
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Switch
            checked={link.enabled}
            onCheckedChange={(enabled) => onUpdate({ enabled })}
          />
          <Button
            type="button"
            size="sm"
            variant="ghost"
            className="text-destructive hover:text-destructive h-7 w-7 p-0"
            disabled={isUnlinking}
            onClick={onUnlink}
          >
            {isUnlinking ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Trash2 className="h-3.5 w-3.5" />
            )}
          </Button>
        </div>
      </div>
      <div className="grid grid-cols-3 gap-2">
        <div className="space-y-1">
          <Label className="text-muted-foreground text-xs">Priority</Label>
          <Input
            type="number"
            min="0"
            value={link.priority}
            className="h-8 text-sm"
            onChange={(e) =>
              onUpdate({ priority: parseInt(e.target.value) || 0 })
            }
          />
        </div>
        <div className="space-y-1">
          <Label className="text-muted-foreground text-xs">Max Chunks</Label>
          <Input
            type="number"
            min="1"
            value={link.max_chunks}
            className="h-8 text-sm"
            onChange={(e) =>
              onUpdate({ max_chunks: parseInt(e.target.value) || 1 })
            }
          />
        </div>
        <div className="space-y-1">
          <Label className="text-muted-foreground text-xs">
            Sim. Threshold
          </Label>
          <Input
            type="number"
            min="0"
            max="1"
            step="0.05"
            value={link.similarity_threshold}
            className="h-8 text-sm"
            onChange={(e) =>
              onUpdate({
                similarity_threshold: parseFloat(e.target.value) || 0,
              })
            }
          />
        </div>
      </div>
    </div>
  );
}
