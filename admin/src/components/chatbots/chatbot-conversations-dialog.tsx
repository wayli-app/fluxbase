import { useState, useCallback } from "react";
import { formatDistanceToNow } from "date-fns";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import {
  ChevronRight,
  Clock,
  MessageSquare,
  User,
  Database,
  AlertCircle,
} from "lucide-react";
import {
  conversationsApi,
  type ConversationSummary,
  type MessageDetail,
} from "@/lib/api";

interface ChatbotConversationsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  chatbotId: string | null;
  chatbotName?: string;
}

export function ChatbotConversationsDialog({
  open,
  onOpenChange,
  chatbotId,
  chatbotName,
}: ChatbotConversationsDialogProps) {
  const [conversations, setConversations] = useState<ConversationSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedConv, setSelectedConv] = useState<ConversationSummary | null>(
    null,
  );
  const [messages, setMessages] = useState<MessageDetail[]>([]);
  const [loadingMessages, setLoadingMessages] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchConversations = useCallback(async () => {
    if (!chatbotId) return;
    setLoading(true);
    try {
      const data = await conversationsApi.list({ chatbot_id: chatbotId, limit: 50 });
      setConversations(data.conversations || []);
    } catch {
      setConversations([]);
    } finally {
      setLoading(false);
    }
  }, [chatbotId]);

  const fetchMessages = useCallback(async (conversationId: string) => {
    setLoadingMessages(true);
    try {
      const data = await conversationsApi.getMessages(conversationId);
      setMessages(data.messages || []);
    } catch {
      setMessages([]);
    } finally {
      setLoadingMessages(false);
    }
  }, []);

  const handleConversationClick = (conv: ConversationSummary) => {
    setSelectedConv(conv);
    setDetailOpen(true);
    fetchMessages(conv.id);
  };

  return (
    <>
      <Dialog
        open={open && !detailOpen}
        onOpenChange={(v) => {
          if (v) fetchConversations();
          onOpenChange(v);
        }}
      >
        <DialogContent className="max-w-3xl max-h-[80vh]">
          <DialogHeader>
            <DialogTitle>
              Conversations{chatbotName ? ` - ${chatbotName}` : ""}
            </DialogTitle>
            <DialogDescription>
              View conversation history for this chatbot
            </DialogDescription>
          </DialogHeader>

          <ScrollArea className="h-[60vh] pr-4">
            {loading ? (
              <p className="text-muted-foreground text-center py-8">
                Loading conversations...
              </p>
            ) : conversations.length === 0 ? (
              <p className="text-muted-foreground text-center py-8">
                No conversations found
              </p>
            ) : (
              <div className="space-y-2">
                {conversations.map((conv) => (
                  <Card
                    key={conv.id}
                    className="cursor-pointer hover:bg-accent transition-colors"
                    onClick={() => handleConversationClick(conv)}
                  >
                    <CardContent className="flex items-center justify-between p-4">
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          <span className="font-medium">
                            {conv.title || `Conversation ${conv.id.slice(0, 8)}`}
                          </span>
                          <Badge variant="outline">{conv.turn_count} turns</Badge>
                        </div>
                        <div className="text-muted-foreground flex items-center gap-3 text-xs">
                          {conv.user_email && (
                            <span className="flex items-center gap-1">
                              <User className="h-3 w-3" />
                              {conv.user_email}
                            </span>
                          )}
                          <span className="flex items-center gap-1">
                            <Clock className="h-3 w-3" />
                            {formatDistanceToNow(new Date(conv.last_message_at), {
                              addSuffix: true,
                            })}
                          </span>
                          <span>
                            {conv.total_prompt_tokens + conv.total_completion_tokens}{" "}
                            tokens
                          </span>
                        </div>
                      </div>
                      <ChevronRight className="text-muted-foreground h-5 w-5" />
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </ScrollArea>
        </DialogContent>
      </Dialog>

      <Sheet open={detailOpen} onOpenChange={setDetailOpen}>
        <SheetContent className="w-[600px] sm:max-w-[600px] overflow-y-auto">
          <SheetHeader>
            <SheetTitle>
              {selectedConv?.title || `Conversation ${selectedConv?.id.slice(0, 8)}`}
            </SheetTitle>
            {selectedConv && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Badge variant="outline">{selectedConv.turn_count} turns</Badge>
                <Badge variant="secondary">
                  {selectedConv.total_prompt_tokens + selectedConv.total_completion_tokens} tokens
                </Badge>
                {selectedConv.user_email && (
                  <span className="flex items-center gap-1">
                    <User className="h-3 w-3" />
                    {selectedConv.user_email}
                  </span>
                )}
              </div>
            )}
          </SheetHeader>

          <div className="mt-4 space-y-4">
            {loadingMessages ? (
              <p className="text-muted-foreground text-center py-8">
                Loading messages...
              </p>
            ) : messages.length === 0 ? (
              <p className="text-muted-foreground text-center py-8">
                No messages found
              </p>
            ) : (
              messages.map((msg) => <MessageBubble key={msg.id} message={msg} />)
            )}
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
}

function MessageBubble({ message }: { message: MessageDetail }) {
  const isUser = message.role === "user";
  const isTool = message.role === "tool" || message.tool_name;

  return (
    <div
      className={`flex ${isUser ? "justify-end" : "justify-start"}`}
    >
      <div
        className={`max-w-[85%] rounded-lg p-3 ${
          isUser
            ? "bg-primary text-primary-foreground"
            : isTool
              ? "bg-muted border"
              : "bg-accent"
        }`}
      >
        {isTool && message.tool_name && (
          <div className="mb-2 flex items-center gap-1 text-xs font-medium text-muted-foreground">
            {message.executed_sql ? (
              <Database className="h-3 w-3" />
            ) : (
              <MessageSquare className="h-3 w-3" />
            )}
            {message.tool_name}
          </div>
        )}

        {message.content && (
          <p className="text-sm whitespace-pre-wrap">{message.content}</p>
        )}

        {message.executed_sql && (
          <div className="mt-2">
            <div className="rounded bg-background/80 p-2">
              <code className="text-xs">{message.executed_sql}</code>
            </div>
            {message.sql_result_summary && (
              <p className="mt-1 text-xs text-muted-foreground">
                {message.sql_result_summary}
                {message.sql_row_count !== undefined &&
                  ` (${message.sql_row_count} rows)`}
                {message.sql_duration_ms !== undefined &&
                  ` - ${message.sql_duration_ms}ms`}
              </p>
            )}
            {message.sql_error && (
              <p className="mt-1 flex items-center gap-1 text-xs text-destructive">
                <AlertCircle className="h-3 w-3" />
                {message.sql_error}
              </p>
            )}
          </div>
        )}

        <p className="mt-1 text-xs opacity-60">
          {formatDistanceToNow(new Date(message.created_at), { addSuffix: true })}
        </p>
      </div>
    </div>
  );
}
