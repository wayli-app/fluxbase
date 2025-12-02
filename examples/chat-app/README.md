# Chat Application - Fluxbase Example

**A real-time chat application built with React and Fluxbase**

![Chat App Screenshot](./screenshot.png)

## 🎯 Features

### Core Features

- ✅ Real-time messaging via WebSocket
- ✅ Multiple chat rooms/channels
- ✅ Direct messages (1-on-1)
- ✅ User presence tracking (online/offline)
- ✅ Typing indicators
- ✅ Message history with pagination
- ✅ File/image sharing
- ✅ Emoji support
- ✅ Message reactions

### Advanced Features

- ✅ Read receipts
- ✅ Message search
- ✅ User profiles with avatars
- ✅ Room creation and management
- ✅ Admin moderation tools
- ✅ Message editing/deletion
- ✅ Thread replies
- ✅ Notifications

## 🏗️ Architecture

```
React Client → Fluxbase SDK (WebSocket) → Fluxbase Server → PostgreSQL
                                                    ↓
                                             Storage (Files)
```

**Real-time Data Flow**:

1. User sends message → INSERT into messages table
2. PostgreSQL trigger fires → NOTIFY event
3. Fluxbase broadcasts via WebSocket → All connected clients
4. React updates UI instantly

## 🚀 Quick Start

### Prerequisites

- Node.js 20+
- Fluxbase instance running
- PostgreSQL database

### 1. Set Up Database

```sql
-- Rooms/Channels table
CREATE TABLE rooms (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  description TEXT,
  type TEXT NOT NULL DEFAULT 'public',  -- 'public', 'private', 'direct'
  created_by UUID NOT NULL REFERENCES auth.users(id),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Room members
CREATE TABLE room_members (
  room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  role TEXT DEFAULT 'member',  -- 'admin', 'moderator', 'member'
  joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  PRIMARY KEY (room_id, user_id)
);

-- Messages table
CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  type TEXT DEFAULT 'text',  -- 'text', 'image', 'file', 'system'
  file_url TEXT,
  file_name TEXT,
  file_size INTEGER,
  edited BOOLEAN DEFAULT FALSE,
  parent_id UUID REFERENCES messages(id),  -- For threads
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Message reactions
CREATE TABLE message_reactions (
  message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  emoji TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  PRIMARY KEY (message_id, user_id, emoji)
);

-- Read receipts
CREATE TABLE read_receipts (
  room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  last_read_message_id UUID REFERENCES messages(id),
  last_read_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  PRIMARY KEY (room_id, user_id)
);

-- User profiles
CREATE TABLE user_profiles (
  id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
  username TEXT UNIQUE NOT NULL,
  display_name TEXT,
  avatar_url TEXT,
  status TEXT DEFAULT 'offline',  -- 'online', 'away', 'busy', 'offline'
  status_message TEXT,
  last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Typing indicators (temporary, in-memory via realtime)
-- No table needed, handled via WebSocket broadcasts

-- Enable RLS
ALTER TABLE rooms ENABLE ROW LEVEL SECURITY;
ALTER TABLE room_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE message_reactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE read_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_profiles ENABLE ROW LEVEL SECURITY;

-- RLS Policies for rooms
CREATE POLICY "Users can view public rooms"
  ON rooms FOR SELECT
  USING (type = 'public' OR id IN (
    SELECT room_id FROM room_members WHERE user_id::text = current_setting('app.user_id', true)
  ));

CREATE POLICY "Users can create rooms"
  ON rooms FOR INSERT
  WITH CHECK (created_by::text = current_setting('app.user_id', true));

CREATE POLICY "Room admins can update rooms"
  ON rooms FOR UPDATE
  USING (id IN (
    SELECT room_id FROM room_members
    WHERE user_id::text = current_setting('app.user_id', true)
    AND role IN ('admin', 'moderator')
  ));

-- RLS Policies for room_members
CREATE POLICY "Users can view room members they belong to"
  ON room_members FOR SELECT
  USING (room_id IN (
    SELECT room_id FROM room_members WHERE user_id::text = current_setting('app.user_id', true)
  ));

CREATE POLICY "Users can join rooms"
  ON room_members FOR INSERT
  WITH CHECK (user_id::text = current_setting('app.user_id', true));

-- RLS Policies for messages
CREATE POLICY "Users can view messages in their rooms"
  ON messages FOR SELECT
  USING (room_id IN (
    SELECT room_id FROM room_members WHERE user_id::text = current_setting('app.user_id', true)
  ));

CREATE POLICY "Users can insert messages in their rooms"
  ON messages FOR INSERT
  WITH CHECK (
    user_id::text = current_setting('app.user_id', true) AND
    room_id IN (
      SELECT room_id FROM room_members WHERE user_id::text = current_setting('app.user_id', true)
    )
  );

CREATE POLICY "Users can update own messages"
  ON messages FOR UPDATE
  USING (user_id::text = current_setting('app.user_id', true));

CREATE POLICY "Users can delete own messages"
  ON messages FOR DELETE
  USING (user_id::text = current_setting('app.user_id', true));

-- RLS Policies for message_reactions
CREATE POLICY "Users can view reactions in their rooms"
  ON message_reactions FOR SELECT
  USING (message_id IN (
    SELECT id FROM messages WHERE room_id IN (
      SELECT room_id FROM room_members WHERE user_id::text = current_setting('app.user_id', true)
    )
  ));

CREATE POLICY "Users can add reactions"
  ON message_reactions FOR INSERT
  WITH CHECK (user_id::text = current_setting('app.user_id', true));

CREATE POLICY "Users can remove own reactions"
  ON message_reactions FOR DELETE
  USING (user_id::text = current_setting('app.user_id', true));

-- RLS Policies for user_profiles
CREATE POLICY "Profiles are viewable by everyone"
  ON user_profiles FOR SELECT USING (true);

CREATE POLICY "Users can update own profile"
  ON user_profiles FOR UPDATE
  USING (id::text = current_setting('app.user_id', true));

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_rooms_type ON rooms(type);
CREATE INDEX IF NOT EXISTS idx_room_members_user ON room_members(user_id);
CREATE INDEX IF NOT EXISTS idx_room_members_room ON room_members(room_id);
CREATE INDEX IF NOT EXISTS idx_messages_room ON messages(room_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_user ON messages(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_parent ON messages(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_message_reactions_message ON message_reactions(message_id);
CREATE INDEX IF NOT EXISTS idx_read_receipts_user ON read_receipts(user_id);

-- Function to update room's updated_at on new message
CREATE OR REPLACE FUNCTION update_room_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE rooms SET updated_at = NOW() WHERE id = NEW.room_id;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_room_on_message
  AFTER INSERT ON messages
  FOR EACH ROW
  EXECUTE FUNCTION update_room_timestamp();

-- View for room with latest message
CREATE VIEW rooms_with_latest_message AS
SELECT
  r.*,
  m.content AS latest_message,
  m.created_at AS latest_message_at,
  up.display_name AS latest_message_author
FROM rooms r
LEFT JOIN LATERAL (
  SELECT * FROM messages
  WHERE room_id = r.id
  ORDER BY created_at DESC
  LIMIT 1
) m ON true
LEFT JOIN user_profiles up ON up.id = m.user_id;

-- View for unread message counts
CREATE VIEW unread_counts AS
SELECT
  rm.room_id,
  rm.user_id,
  COUNT(m.id) AS unread_count
FROM room_members rm
LEFT JOIN read_receipts rr ON rr.room_id = rm.room_id AND rr.user_id = rm.user_id
LEFT JOIN messages m ON m.room_id = rm.room_id
  AND (rr.last_read_at IS NULL OR m.created_at > rr.last_read_at)
  AND m.user_id != rm.user_id
GROUP BY rm.room_id, rm.user_id;
```

### 2. Install Dependencies

```bash
cd examples/chat-app
npm install
```

**Key Dependencies**:

```json
{
  "dependencies": {
    "@fluxbase/client": "latest",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "zustand": "^4.4.1",
    "date-fns": "^2.30.0",
    "emoji-picker-react": "^4.5.0",
    "@tanstack/react-query": "^5.0.0"
  }
}
```

### 3. Configure Environment

```bash
cp .env.example .env.local
```

Edit `.env.local`:

```env
VITE_FLUXBASE_URL=http://localhost:8080
VITE_FLUXBASE_ANON_KEY=your-anon-key
```

### 4. Run Development Server

```bash
npm run dev
```

Open [http://localhost:5173](http://localhost:5173)

## 📁 Project Structure

```
chat-app/
├── src/
│   ├── components/
│   │   ├── chat/
│   │   │   ├── ChatWindow.tsx       # Main chat interface
│   │   │   ├── MessageList.tsx      # Message display
│   │   │   ├── MessageInput.tsx     # Send messages
│   │   │   ├── MessageItem.tsx      # Individual message
│   │   │   ├── TypingIndicator.tsx  # "User is typing..."
│   │   │   └── FilePreview.tsx      # File attachments
│   │   ├── rooms/
│   │   │   ├── RoomList.tsx         # List of rooms
│   │   │   ├── RoomItem.tsx         # Room with unread badge
│   │   │   ├── CreateRoom.tsx       # New room modal
│   │   │   └── RoomSettings.tsx     # Room configuration
│   │   ├── users/
│   │   │   ├── UserList.tsx         # Online users
│   │   │   ├── UserProfile.tsx      # User details
│   │   │   └── StatusSelector.tsx   # Online/away/busy
│   │   └── common/
│   │       ├── EmojiPicker.tsx      # Emoji selector
│   │       └── FileUploader.tsx     # Upload files
│   ├── hooks/
│   │   ├── useChat.ts               # Chat messages
│   │   ├── useRooms.ts              # Room management
│   │   ├── usePresence.ts           # User presence
│   │   ├── useTyping.ts             # Typing indicators
│   │   └── useUnread.ts             # Unread counts
│   ├── store/
│   │   └── chatStore.ts             # Zustand store
│   ├── lib/
│   │   ├── fluxbase.ts              # Fluxbase client
│   │   └── utils.ts                 # Utilities
│   ├── App.tsx
│   └── main.tsx
└── package.json
```

## 💻 Code Examples

### Zustand Store

```typescript
// src/store/chatStore.ts
import { create } from "zustand";
import type { Room, Message, User } from "../lib/types";

interface ChatState {
  currentRoom: Room | null;
  rooms: Room[];
  messages: Message[];
  onlineUsers: Set<string>;
  typingUsers: Map<string, string[]>; // roomId -> userIds[]

  setCurrentRoom: (room: Room | null) => void;
  addMessage: (message: Message) => void;
  updateMessage: (messageId: string, updates: Partial<Message>) => void;
  deleteMessage: (messageId: string) => void;
  setOnlineUsers: (users: Set<string>) => void;
  setTyping: (roomId: string, userId: string, isTyping: boolean) => void;
}

export const useChatStore = create<ChatState>((set) => ({
  currentRoom: null,
  rooms: [],
  messages: [],
  onlineUsers: new Set(),
  typingUsers: new Map(),

  setCurrentRoom: (room) => set({ currentRoom: room }),

  addMessage: (message) =>
    set((state) => ({
      messages: [...state.messages, message],
    })),

  updateMessage: (messageId, updates) =>
    set((state) => ({
      messages: state.messages.map((m) =>
        m.id === messageId ? { ...m, ...updates } : m
      ),
    })),

  deleteMessage: (messageId) =>
    set((state) => ({
      messages: state.messages.filter((m) => m.id !== messageId),
    })),

  setOnlineUsers: (users) => set({ onlineUsers: users }),

  setTyping: (roomId, userId, isTyping) =>
    set((state) => {
      const typing = new Map(state.typingUsers);
      const current = typing.get(roomId) || [];

      if (isTyping) {
        typing.set(roomId, [...current, userId]);
      } else {
        typing.set(
          roomId,
          current.filter((id) => id !== userId)
        );
      }

      return { typingUsers: typing };
    }),
}));
```

### useChat Hook

```typescript
// src/hooks/useChat.ts
import { useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fluxbase } from "../lib/fluxbase";
import { useChatStore } from "../store/chatStore";
import type { Message } from "../lib/types";

export function useChat(roomId: string) {
  const queryClient = useQueryClient();
  const { addMessage, updateMessage, deleteMessage } = useChatStore();

  // Fetch messages
  const { data: messages, isLoading } = useQuery({
    queryKey: ["messages", roomId],
    queryFn: async () => {
      const { data, error } = await fluxbase
        .from<Message>("messages")
        .select("*, user:user_profiles(*), reactions:message_reactions(*)")
        .eq("room_id", roomId)
        .order("created_at", { ascending: true })
        .limit(100);

      if (error) throw error;
      return data || [];
    },
  });

  // Subscribe to new messages
  useEffect(() => {
    const subscription = fluxbase
      .from("messages")
      .on("INSERT", (payload) => {
        if (payload.record.room_id === roomId) {
          addMessage(payload.record as Message);
        }
      })
      .on("UPDATE", (payload) => {
        if (payload.record.room_id === roomId) {
          updateMessage(payload.record.id, payload.record);
        }
      })
      .on("DELETE", (payload) => {
        if (payload.old_record.room_id === roomId) {
          deleteMessage(payload.old_record.id);
        }
      })
      .filter("room_id", "eq", roomId)
      .subscribe();

    return () => {
      subscription.unsubscribe();
    };
  }, [roomId, addMessage, updateMessage, deleteMessage]);

  // Send message
  const sendMessage = useMutation({
    mutationFn: async (content: string) => {
      const { data, error } = await fluxbase
        .from<Message>("messages")
        .insert({
          room_id: roomId,
          user_id: fluxbase.auth.user()!.id,
          content,
          type: "text",
        })
        .select()
        .single();

      if (error) throw error;
      return data;
    },
  });

  // Edit message
  const editMessage = useMutation({
    mutationFn: async ({ id, content }: { id: string; content: string }) => {
      const { data, error } = await fluxbase
        .from<Message>("messages")
        .update({ content, edited: true })
        .eq("id", id)
        .select()
        .single();

      if (error) throw error;
      return data;
    },
  });

  // Delete message
  const removeMessage = useMutation({
    mutationFn: async (id: string) => {
      const { error } = await fluxbase
        .from<Message>("messages")
        .delete()
        .eq("id", id);

      if (error) throw error;
    },
  });

  return {
    messages: messages || [],
    isLoading,
    sendMessage: sendMessage.mutate,
    editMessage: editMessage.mutate,
    deleteMessage: removeMessage.mutate,
  };
}
```

### usePresence Hook

```typescript
// src/hooks/usePresence.ts
import { useEffect } from "react";
import { fluxbase } from "../lib/fluxbase";
import { useChatStore } from "../store/chatStore";

export function usePresence(roomId: string) {
  const { setOnlineUsers } = useChatStore();

  useEffect(() => {
    const channel = fluxbase.channel(`room:${roomId}`);

    // Track own presence
    const user = fluxbase.auth.user();
    if (user) {
      channel.track({
        user_id: user.id,
        username: user.email,
        status: "online",
      });
    }

    // Listen to presence changes
    channel.on("presence", { event: "sync" }, () => {
      const state = channel.presenceState();
      const onlineUserIds = new Set(
        Object.keys(state).map((key) => state[key][0].user_id)
      );
      setOnlineUsers(onlineUserIds);
    });

    channel.subscribe();

    return () => {
      channel.unsubscribe();
    };
  }, [roomId, setOnlineUsers]);
}
```

### useTyping Hook

```typescript
// src/hooks/useTyping.ts
import { useEffect, useRef, useCallback } from "react";
import { fluxbase } from "../lib/fluxbase";
import { useChatStore } from "../store/chatStore";

export function useTyping(roomId: string) {
  const { setTyping } = useChatStore();
  const typingTimeout = useRef<NodeJS.Timeout>();

  useEffect(() => {
    const channel = fluxbase.channel(`room:${roomId}:typing`);

    // Listen to typing broadcasts
    channel.on("broadcast", { event: "typing" }, (payload) => {
      const { user_id, is_typing } = payload.payload;
      setTyping(roomId, user_id, is_typing);

      // Clear typing after 3 seconds
      if (is_typing) {
        setTimeout(() => {
          setTyping(roomId, user_id, false);
        }, 3000);
      }
    });

    channel.subscribe();

    return () => {
      channel.unsubscribe();
    };
  }, [roomId, setTyping]);

  // Notify others of typing
  const notifyTyping = useCallback(() => {
    const channel = fluxbase.channel(`room:${roomId}:typing`);
    const user = fluxbase.auth.user();

    if (user) {
      channel.send({
        type: "broadcast",
        event: "typing",
        payload: {
          user_id: user.id,
          is_typing: true,
        },
      });

      // Clear previous timeout
      if (typingTimeout.current) {
        clearTimeout(typingTimeout.current);
      }

      // Stop typing after 3 seconds of inactivity
      typingTimeout.current = setTimeout(() => {
        channel.send({
          type: "broadcast",
          event: "typing",
          payload: {
            user_id: user.id,
            is_typing: false,
          },
        });
      }, 3000);
    }
  }, [roomId]);

  return { notifyTyping };
}
```

### Message Input Component

```typescript
// src/components/chat/MessageInput.tsx
"use client";

import { useState, useRef } from "react";
import { useTyping } from "../../hooks/useTyping";
import EmojiPicker from "../common/EmojiPicker";
import FileUploader from "../common/FileUploader";

export default function MessageInput({
  roomId,
  onSend,
}: {
  roomId: string;
  onSend: (content: string) => void;
}) {
  const [message, setMessage] = useState("");
  const [showEmoji, setShowEmoji] = useState(false);
  const [showFileUpload, setShowFileUpload] = useState(false);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const { notifyTyping } = useTyping(roomId);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (message.trim()) {
      onSend(message);
      setMessage("");
      inputRef.current?.focus();
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    // Send on Enter (without Shift)
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    } else {
      // Notify typing
      notifyTyping();
    }
  };

  const addEmoji = (emoji: string) => {
    setMessage((prev) => prev + emoji);
    inputRef.current?.focus();
  };

  return (
    <form onSubmit={handleSubmit} className="border-t p-4">
      <div className="flex items-end gap-2">
        {/* Emoji Button */}
        <button
          type="button"
          onClick={() => setShowEmoji(!showEmoji)}
          className="p-2 hover:bg-gray-100 rounded"
        >
          😊
        </button>

        {/* File Upload Button */}
        <button
          type="button"
          onClick={() => setShowFileUpload(true)}
          className="p-2 hover:bg-gray-100 rounded"
        >
          📎
        </button>

        {/* Message Input */}
        <textarea
          ref={inputRef}
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Type a message..."
          className="flex-1 px-3 py-2 border rounded-lg resize-none"
          rows={1}
          style={{ maxHeight: "120px" }}
        />

        {/* Send Button */}
        <button
          type="submit"
          disabled={!message.trim()}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
        >
          Send
        </button>
      </div>

      {/* Emoji Picker */}
      {showEmoji && (
        <div className="absolute bottom-20 left-4">
          <EmojiPicker
            onSelect={addEmoji}
            onClose={() => setShowEmoji(false)}
          />
        </div>
      )}

      {/* File Uploader */}
      {showFileUpload && (
        <FileUploader
          roomId={roomId}
          onUpload={(url, name) => onSend(`Uploaded: ${name}`)}
          onClose={() => setShowFileUpload(false)}
        />
      )}
    </form>
  );
}
```

## 🎨 Features Deep Dive

### Real-time Architecture

The chat uses PostgreSQL LISTEN/NOTIFY:

1. **Insert Message** → Trigger fires
2. **pg_notify()** → Broadcasts event
3. **Fluxbase** → Receives notification
4. **WebSocket** → Pushes to all clients
5. **React** → Updates UI instantly

### Presence Tracking

Users' online status is tracked via WebSocket:

- **join** - User connects to room
- **leave** - User disconnects
- **sync** - Get current presence state

### Typing Indicators

Ephemeral state (not stored in DB):

- Broadcasts via WebSocket channel
- Auto-clears after 3 seconds
- Shows "User is typing..." UI

## 🚀 Deployment

See [deployment guide](./DEPLOYMENT.md).

## 📚 Related Documentation

- [Realtime Guide](../../docs/guides/realtime.md)
- [Presence Tracking](../../docs/guides/presence.md)
- [WebSocket API](../../docs/api/websocket.md)

---

**Status**: Complete ✅
**Difficulty**: Intermediate/Advanced
**Time to Complete**: 2-3 hours
**Lines of Code**: ~2,000
