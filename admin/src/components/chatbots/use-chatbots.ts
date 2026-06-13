import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { chatbotsApi, type AIChatbotSummary } from "@/lib/api";
import { useTenantStore } from "@/stores/tenant-store";

export function useChatbots() {
  const queryClient = useQueryClient();
  const currentTenantId = useTenantStore((state) => state.currentTenant?.id);

  const query = useQuery({
    queryKey: ["chatbots", currentTenantId],
    queryFn: async () => {
      const data = await chatbotsApi.list();
      return data || [];
    },
  });

  const syncMutation = useMutation({
    mutationFn: () => chatbotsApi.sync(),
    onSuccess: (result) => {
      const { created, updated, deleted, errors } = result.summary;

      if (created > 0 || updated > 0 || deleted > 0) {
        const messages: string[] = [];
        if (created > 0) messages.push(`${created} created`);
        if (updated > 0) messages.push(`${updated} updated`);
        if (deleted > 0) messages.push(`${deleted} deleted`);
        toast.success(`Chatbots synced: ${messages.join(", ")}`);
      } else if (errors > 0) {
        toast.error(`Failed to sync chatbots: ${errors} errors`);
      } else {
        toast.info("No changes detected");
      }
      queryClient.invalidateQueries({ queryKey: ["chatbots"] });
    },
    onError: () => toast.error("Failed to sync chatbots from filesystem"),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ chatbot }: { chatbot: AIChatbotSummary }) =>
      chatbotsApi.toggle(chatbot.id, !chatbot.enabled),
    onSuccess: (_data, { chatbot }) => {
      toast.success(`Chatbot ${!chatbot.enabled ? "enabled" : "disabled"}`);
      queryClient.invalidateQueries({ queryKey: ["chatbots"] });
    },
    onError: () => toast.error("Failed to toggle chatbot"),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => chatbotsApi.delete(id),
    onSuccess: () => {
      toast.success("Chatbot deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["chatbots"] });
    },
    onError: () => toast.error("Failed to delete chatbot"),
  });

  return {
    chatbots: query.data || [],
    loading: query.isLoading,
    reloading: syncMutation.isPending,
    sync: syncMutation.mutate,
    toggle: (chatbot: AIChatbotSummary) =>
      toggleMutation.mutate({ chatbot }),
    deleteChatbot: deleteMutation.mutate,
    refetch: query.refetch,
  };
}
