import { useQuery } from "@tanstack/react-query";
import { userKnowledgeBasesApi, type KnowledgeBaseSummary } from "@/lib/api";
import { KnowledgeBaseCard } from "./-KnowledgeBaseCard";

function MyKnowledgeBasesList() {
  const { data, isLoading, error } = useQuery<KnowledgeBaseSummary[]>({
    queryKey: ["user-knowledge-bases"],
    queryFn: () => userKnowledgeBasesApi.list(),
  });

  if (isLoading) return <div className="text-center py-8">Loading...</div>;
  if (error)
    return (
      <div className="text-red-500 py-8">Error loading knowledge bases</div>
    );

  const myKBs =
    data?.filter((kb) => kb.user_permission === "owner") || [];

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {myKBs.map((kb) => (
        <KnowledgeBaseCard key={kb.id} kb={kb} isOwner={true} />
      ))}
      {myKBs.length === 0 && (
        <div className="col-span-full text-center py-12 text-muted-foreground">
          No knowledge bases yet. Create your first one!
        </div>
      )}
    </div>
  );
}

export { MyKnowledgeBasesList };
