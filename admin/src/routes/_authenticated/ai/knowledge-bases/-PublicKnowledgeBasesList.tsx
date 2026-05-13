import { useQuery } from "@tanstack/react-query";
import { userKnowledgeBasesApi, type KnowledgeBaseSummary } from "@/lib/api";
import { KnowledgeBaseCard } from "./-KnowledgeBaseCard";

function PublicKnowledgeBasesList() {
  const { data, isLoading, error } = useQuery<KnowledgeBaseSummary[]>({
    queryKey: ["user-knowledge-bases"],
    queryFn: () => userKnowledgeBasesApi.list(),
  });

  if (isLoading) return <div className="text-center py-8">Loading...</div>;
  if (error)
    return (
      <div className="text-red-500 py-8">Error loading knowledge bases</div>
    );

  const publicKBs =
    data?.filter((kb) => kb.visibility === "public") || [];

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {publicKBs.map((kb) => (
        <KnowledgeBaseCard key={kb.id} kb={kb} isOwner={false} />
      ))}
      {publicKBs.length === 0 && (
        <div className="col-span-full text-center py-12 text-muted-foreground">
          No public knowledge bases available.
        </div>
      )}
    </div>
  );
}

export { PublicKnowledgeBasesList };
