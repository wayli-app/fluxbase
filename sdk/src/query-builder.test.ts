/**
 * Comprehensive Query Builder Tests
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { QueryBuilder } from "./query-builder";
import type { FluxbaseFetch } from "./fetch";

// Helper to create mock Headers
function createMockHeaders(headersInit?: Record<string, string>): Headers {
  const headers = new Headers();
  if (headersInit) {
    for (const [key, value] of Object.entries(headersInit)) {
      headers.set(key, value);
    }
  }
  return headers;
}

// Mock FluxbaseFetch
class MockFetch implements FluxbaseFetch {
  public lastUrl: string = "";
  public lastMethod: string = "";
  public lastBody: unknown = null;
  public lastHeaders: Record<string, string> = {};
  public mockResponse: unknown = [];
  public mockError: Error | null = null;
  public mockContentRangeHeader: string | null = null;

  constructor(
    public baseUrl: string = "http://localhost:8080",
    public headers: Record<string, string> = {},
  ) {}

  async get<T>(path: string): Promise<T> {
    this.lastUrl = path;
    this.lastMethod = "GET";
    if (this.mockError) {
      throw this.mockError;
    }
    return this.mockResponse as T;
  }

  async getWithHeaders<T>(path: string): Promise<{ data: T; headers: Headers; status: number }> {
    this.lastUrl = path;
    this.lastMethod = "GET";
    if (this.mockError) {
      throw this.mockError;
    }
    const responseHeaders = createMockHeaders(
      this.mockContentRangeHeader ? { "Content-Range": this.mockContentRangeHeader } : undefined
    );
    return {
      data: this.mockResponse as T,
      headers: responseHeaders,
      status: 200,
    };
  }

  async post<T>(
    path: string,
    body?: unknown,
    options?: { headers?: Record<string, string> },
  ): Promise<T> {
    this.lastUrl = path;
    this.lastMethod = "POST";
    this.lastBody = body;
    this.lastHeaders = options?.headers || {};
    return body as T;
  }

  async patch<T>(path: string, body?: unknown): Promise<T> {
    this.lastUrl = path;
    this.lastMethod = "PATCH";
    this.lastBody = body;
    return body as T;
  }

  async delete(path: string): Promise<void> {
    this.lastUrl = path;
    this.lastMethod = "DELETE";
  }

  setAuthToken(token: string | null): void {
    if (token) {
      this.headers["Authorization"] = `Bearer ${token}`;
    } else {
      delete this.headers["Authorization"];
    }
  }
}

describe("QueryBuilder - Select Operations", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "users");
  });

  it("should select all columns by default", async () => {
    await builder.execute();
    // When selectQuery is '*' (default), it's not included in the URL (optimization)
    expect(fetch.lastUrl).toBe("/api/v1/tables/users");
  });

  it("should select specific columns", async () => {
    await builder.select("id, name, email").execute();
    expect(fetch.lastUrl).toContain("select=id");
    expect(fetch.lastUrl).toContain("name");
    expect(fetch.lastUrl).toContain("email");
  });

  it("should select with aggregations", async () => {
    await builder.select("count, sum(price), avg(rating)").execute();
    expect(fetch.lastUrl).toContain("count");
    expect(fetch.lastUrl).toContain("sum");
    expect(fetch.lastUrl).toContain("avg");
  });
});

describe("QueryBuilder - Filter Operators", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "products");
  });

  it("should filter with eq (equals)", async () => {
    await builder.eq("price", 29.99).execute();
    expect(fetch.lastUrl).toContain("price=eq.29.99");
  });

  it("should filter with neq (not equals)", async () => {
    await builder.neq("status", "deleted").execute();
    expect(fetch.lastUrl).toContain("status=neq.deleted");
  });

  it("should filter with gt (greater than)", async () => {
    await builder.gt("stock", 10).execute();
    expect(fetch.lastUrl).toContain("stock=gt.10");
  });

  it("should filter with gte (greater than or equal)", async () => {
    await builder.gte("price", 50).execute();
    expect(fetch.lastUrl).toContain("price=gte.50");
  });

  it("should filter with lt (less than)", async () => {
    await builder.lt("discount", 0.5).execute();
    expect(fetch.lastUrl).toContain("discount=lt.0.5");
  });

  it("should filter with lte (less than or equal)", async () => {
    await builder.lte("rating", 3).execute();
    expect(fetch.lastUrl).toContain("rating=lte.3");
  });

  it("should filter with like (pattern matching)", async () => {
    await builder.like("name", "%Product%").execute();
    expect(fetch.lastUrl).toContain("name=like.%25Product%25");
  });

  it("should filter with ilike (case-insensitive like)", async () => {
    await builder.ilike("email", "%@gmail.com").execute();
    expect(fetch.lastUrl).toContain("email=ilike");
  });

  it("should filter with in (list)", async () => {
    await builder
      .in("category", ["electronics", "books", "clothing"])
      .execute();
    expect(fetch.lastUrl).toContain("category=in.");
  });

  it("should filter with is null", async () => {
    await builder.is("deleted_at", null).execute();
    expect(fetch.lastUrl).toContain("deleted_at=is.null");
  });

  it("should chain multiple filters", async () => {
    await builder
      .eq("status", "active")
      .gte("price", 10)
      .lte("price", 100)
      .execute();

    expect(fetch.lastUrl).toContain("status=eq.active");
    expect(fetch.lastUrl).toContain("price=gte.10");
    expect(fetch.lastUrl).toContain("price=lte.100");
  });
});

describe("QueryBuilder - Ordering", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "posts");
  });

  it("should order by column ascending", async () => {
    await builder.order("created_at", { ascending: true }).execute();
    expect(fetch.lastUrl).toContain("order=created_at.asc");
  });

  it("should order by column descending", async () => {
    await builder.order("views", { ascending: false }).execute();
    expect(fetch.lastUrl).toContain("order=views.desc");
  });

  it("should support multiple order by", async () => {
    await builder
      .order("category", { ascending: true })
      .order("price", { ascending: false })
      .execute();

    expect(fetch.lastUrl).toContain("order=category.asc");
    expect(fetch.lastUrl).toContain("price.desc");
  });
});

describe("QueryBuilder - Pagination", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "articles");
  });

  it("should limit results", async () => {
    await builder.limit(10).execute();
    expect(fetch.lastUrl).toContain("limit=10");
  });

  it("should offset results", async () => {
    await builder.offset(20).execute();
    expect(fetch.lastUrl).toContain("offset=20");
  });

  it("should combine limit and offset", async () => {
    await builder.limit(10).offset(20).execute();
    expect(fetch.lastUrl).toContain("limit=10");
    expect(fetch.lastUrl).toContain("offset=20");
  });

  it("should support pagination pattern", async () => {
    const page = 3;
    const pageSize = 25;
    await builder
      .limit(pageSize)
      .offset((page - 1) * pageSize)
      .execute();

    expect(fetch.lastUrl).toContain("limit=25");
    expect(fetch.lastUrl).toContain("offset=50");
  });
});

describe("QueryBuilder - Insert Operations", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "users");
  });

  it("should insert a single row", async () => {
    const user = { name: "John Doe", email: "john@example.com" };
    await builder.insert(user);

    expect(fetch.lastMethod).toBe("POST");
    expect(fetch.lastUrl).toContain("/api/v1/tables/users");
    expect(fetch.lastBody).toEqual(user);
  });

  it("should insert multiple rows", async () => {
    const users = [
      { name: "Alice", email: "alice@example.com" },
      { name: "Bob", email: "bob@example.com" },
    ];
    await builder.insert(users);

    expect(fetch.lastMethod).toBe("POST");
    expect(fetch.lastBody).toEqual(users);
  });
});

describe("QueryBuilder - Upsert Operations", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "products");
  });

  it("should upsert with merge-duplicates header", async () => {
    const product = { id: 1, name: "Product", price: 29.99 };
    await builder.upsert(product);

    expect(fetch.lastMethod).toBe("POST");
    expect(fetch.lastHeaders["Prefer"]).toBe("resolution=merge-duplicates");
  });

  it("should upsert multiple rows", async () => {
    const products = [
      { id: 1, name: "Product 1", price: 19.99 },
      { id: 2, name: "Product 2", price: 29.99 },
    ];
    await builder.upsert(products);

    expect(fetch.lastBody).toEqual(products);
    expect(fetch.lastHeaders["Prefer"]).toBe("resolution=merge-duplicates");
  });
});

describe("QueryBuilder - Update Operations", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "posts");
  });

  it("should update with filters", async () => {
    await builder.eq("id", 123).update({ title: "Updated Title" });

    expect(fetch.lastMethod).toBe("PATCH");
    expect(fetch.lastUrl).toContain("id=eq.123");
    expect(fetch.lastBody).toEqual({ title: "Updated Title" });
  });

  it("should update multiple fields", async () => {
    const updates = {
      title: "New Title",
      content: "New Content",
      updated_at: new Date().toISOString(),
    };

    await builder.eq("status", "draft").update(updates);

    expect(fetch.lastUrl).toContain("status=eq.draft");
    expect(fetch.lastBody).toEqual(updates);
  });
});

describe("QueryBuilder - Delete Operations", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "comments");
  });

  it("should delete with filters", async () => {
    await builder.eq("id", 456).delete();

    expect(fetch.lastMethod).toBe("DELETE");
    expect(fetch.lastUrl).toContain("id=eq.456");
  });

  it("should delete multiple rows", async () => {
    await builder.eq("spam", true).delete();

    expect(fetch.lastMethod).toBe("DELETE");
    expect(fetch.lastUrl).toContain("spam=eq.true");
  });
});

describe("QueryBuilder - Single Row", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "profiles");
  });

  it("should fetch single row", async () => {
    await builder.eq("user_id", "abc-123").single().execute();

    expect(fetch.lastUrl).toContain("user_id=eq.abc-123");
    expect(fetch.lastUrl).toContain("limit=1");
  });
});

describe("QueryBuilder - Complex Queries", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "orders");
  });

  it("should build complex query with multiple operations", async () => {
    await builder
      .select("id, total, status, user(name, email)")
      .eq("status", "pending")
      .gte("total", 100)
      .order("created_at", { ascending: false })
      .limit(20)
      .offset(0)
      .execute();

    const url = fetch.lastUrl;
    expect(url).toContain("select=");
    expect(url).toContain("status=eq.pending");
    expect(url).toContain("total=gte.100");
    expect(url).toContain("order=created_at.desc");
    expect(url).toContain("limit=20");
    expect(url).toContain("offset=0");
  });

  it("should support filtering with JSON operators", async () => {
    await builder.eq("metadata->theme", "dark").execute();

    expect(fetch.lastUrl).toContain("metadata");
    expect(fetch.lastUrl).toContain("theme");
  });
});

describe("QueryBuilder - Group By", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "sales");
  });

  it("should group by single column", async () => {
    await builder.select("category, count").groupBy("category").execute();

    expect(fetch.lastUrl).toContain("group_by=category");
  });

  it("should group by multiple columns", async () => {
    await builder
      .select("category, status, count")
      .groupBy("category,status")
      .execute();

    expect(fetch.lastUrl).toContain("group_by=");
    expect(fetch.lastUrl).toContain("category");
  });
});

describe("QueryBuilder - Text Search", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "documents");
  });

  it("should perform full-text search", async () => {
    await builder.textSearch("content", "search terms").execute();

    expect(fetch.lastUrl).toContain("content=fts");
  });
});

describe("QueryBuilder - Batch Operations", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "tasks");
  });

  it("should batch insert", async () => {
    const tasks = [
      { title: "Task 1", completed: false },
      { title: "Task 2", completed: false },
      { title: "Task 3", completed: false },
    ];

    await builder.insert(tasks);

    expect(fetch.lastBody).toEqual(tasks);
    expect(Array.isArray(fetch.lastBody)).toBe(true);
  });

  it("should batch update with filters", async () => {
    await builder.eq("status", "pending").update({ status: "completed" });

    expect(fetch.lastUrl).toContain("status=eq.pending");
    expect(fetch.lastBody).toEqual({ status: "completed" });
  });

  it("should batch delete with filters", async () => {
    await builder.lt("created_at", "2023-01-01").delete();

    expect(fetch.lastUrl).toContain("created_at=lt.2023-01-01");
  });
});

describe("QueryBuilder - Error Handling", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "items");
  });

  it("should handle empty filters gracefully", async () => {
    await builder.execute();

    expect(fetch.lastUrl).toBe("/api/v1/tables/items");
  });

  it("should handle undefined values", async () => {
    await builder.eq("name", undefined as any).execute();

    // Should still build URL even with undefined
    expect(fetch.lastUrl).toContain("name=eq");
  });

  it("should handle null values", async () => {
    await builder.is("deleted_at", null).execute();

    expect(fetch.lastUrl).toContain("deleted_at=is.null");
  });
});

describe("QueryBuilder - Advanced Features", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "analytics");
  });

  it("should support NOT operator", async () => {
    await builder.not("status", "eq", "deleted").execute();

    expect(fetch.lastUrl).toContain("status=not.eq.deleted");
  });

  it("should support OR operator", async () => {
    await builder.or("status.eq.active,status.eq.pending").execute();

    expect(fetch.lastUrl).toContain("or=");
    expect(fetch.lastUrl).toContain("status");
  });

  it("should chain complex filters", async () => {
    await builder
      .select("*")
      .or("status.eq.active,priority.eq.high")
      .gte("score", 80)
      .order("created_at", { ascending: false })
      .limit(50)
      .execute();

    const url = fetch.lastUrl;
    expect(url).toContain("or=");
    expect(url).toContain("score=gte.80");
    expect(url).toContain("limit=50");
  });

  it("should support match() for multiple exact matches", async () => {
    await builder.match({ id: 1, status: "active", role: "admin" }).execute();

    const url = fetch.lastUrl;
    expect(url).toContain("id=eq.1");
    expect(url).toContain("status=eq.active");
    expect(url).toContain("role=eq.admin");
  });

  it("should support filter() generic method", async () => {
    await builder.filter("age", "gte", "18").execute();

    expect(fetch.lastUrl).toContain("age=gte.18");
  });

  it("should support containedBy() for arrays", async () => {
    await builder.containedBy("tags", '["news","update"]').execute();

    expect(fetch.lastUrl).toContain("tags=cd.");
  });

  it("should support overlaps() for arrays", async () => {
    await builder.overlaps("tags", '["news","sports"]').execute();

    expect(fetch.lastUrl).toContain("tags=ov.");
  });

  it("should support and() operator for grouped conditions", async () => {
    await builder.and("status.eq.active,verified.eq.true").execute();

    const url = fetch.lastUrl;
    expect(url).toContain("and=");
    expect(url).toContain("status.eq.active");
    expect(url).toContain("verified.eq.true");
  });

  it("should support maybeSingle() returning null for no results", async () => {
    fetch.mockResponse = [];

    const { data, error } = await builder.eq("id", 999).maybeSingle().execute();

    expect(data).toBeNull();
    expect(error).toBeNull();
  });

  it("should support maybeSingle() returning single row", async () => {
    const mockUser = { id: 1, name: "Alice" };
    fetch.mockResponse = [mockUser];

    const { data, error } = await builder.eq("id", 1).maybeSingle().execute();

    expect(data).toEqual(mockUser);
    expect(error).toBeNull();
  });

  it("should support throwOnError() returning data directly", async () => {
    const mockUsers = [
      { id: 1, name: "Alice" },
      { id: 2, name: "Bob" },
    ];
    fetch.mockResponse = mockUsers;

    const data = await builder.throwOnError();

    expect(data).toEqual(mockUsers);
  });

  it("should support throwOnError() throwing on error", async () => {
    fetch.mockError = new Error("Network error");

    await expect(builder.throwOnError()).rejects.toThrow("Network error");
  });

  it("should support upsert() with onConflict option", async () => {
    await builder.upsert(
      { id: 1, email: "alice@example.com" },
      { onConflict: "email" },
    );

    const url = fetch.lastUrl;
    expect(url).toContain("on_conflict=email");
    expect(fetch.lastHeaders?.Prefer).toContain("resolution=merge-duplicates");
  });

  it("should support upsert() with ignoreDuplicates option", async () => {
    await builder.upsert(
      { id: 1, email: "alice@example.com" },
      { ignoreDuplicates: true },
    );

    expect(fetch.lastHeaders?.Prefer).toContain("resolution=ignore-duplicates");
  });

  it("should support upsert() with defaultToNull option", async () => {
    await builder.upsert({ id: 1, name: "Alice" }, { defaultToNull: true });

    expect(fetch.lastHeaders?.Prefer).toContain("missing=default");
  });
});

describe("QueryBuilder - RPC (Remote Procedure Call)", () => {
  let fetch: MockFetch;

  beforeEach(() => {
    fetch = new MockFetch();
  });

  it("should call RPC function", async () => {
    await fetch.post("/api/v1/rpc/calculate_total", { order_id: 123 });

    expect(fetch.lastUrl).toContain("/api/v1/rpc/calculate_total");
    expect(fetch.lastBody).toEqual({ order_id: 123 });
  });

  it("should call RPC with no parameters", async () => {
    await fetch.post("/api/v1/rpc/get_stats");

    expect(fetch.lastUrl).toContain("/api/v1/rpc/get_stats");
  });
});

describe("QueryBuilder - Method Chaining (New)", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "users");
  });

  describe("Update Chaining", () => {
    it("should allow chaining .eq() after .update()", async () => {
      await builder.update({ name: "Updated Name" }).eq("id", 1);

      expect(fetch.lastMethod).toBe("PATCH");
      expect(fetch.lastUrl).toContain("id=eq.1");
      expect(fetch.lastBody).toEqual({ name: "Updated Name" });
    });

    it("should allow chaining multiple filters after .update()", async () => {
      await builder
        .update({ status: "active" })
        .eq("role", "admin")
        .gte("created_at", "2024-01-01");

      expect(fetch.lastMethod).toBe("PATCH");
      expect(fetch.lastUrl).toContain("role=eq.admin");
      expect(fetch.lastUrl).toContain("created_at=gte.2024-01-01");
      expect(fetch.lastBody).toEqual({ status: "active" });
    });

    it("should allow filters before .update() (backwards compatible)", async () => {
      await builder.eq("id", 1).update({ name: "Updated Name" });

      expect(fetch.lastMethod).toBe("PATCH");
      expect(fetch.lastUrl).toContain("id=eq.1");
      expect(fetch.lastBody).toEqual({ name: "Updated Name" });
    });

    it("should combine filters before and after .update()", async () => {
      await builder
        .eq("status", "pending")
        .update({ status: "completed" })
        .gte("created_at", "2024-01-01");

      expect(fetch.lastMethod).toBe("PATCH");
      expect(fetch.lastUrl).toContain("status=eq.pending");
      expect(fetch.lastUrl).toContain("created_at=gte.2024-01-01");
      expect(fetch.lastBody).toEqual({ status: "completed" });
    });
  });

  describe("Delete Chaining", () => {
    it("should allow chaining .eq() after .delete()", async () => {
      await builder.delete().eq("id", 5);

      expect(fetch.lastMethod).toBe("DELETE");
      expect(fetch.lastUrl).toContain("id=eq.5");
    });

    it("should allow chaining multiple filters after .delete()", async () => {
      await builder.delete().eq("spam", true).lt("created_at", "2023-01-01");

      expect(fetch.lastMethod).toBe("DELETE");
      expect(fetch.lastUrl).toContain("spam=eq.true");
      expect(fetch.lastUrl).toContain("created_at=lt.2023-01-01");
    });

    it("should allow filters before .delete() (backwards compatible)", async () => {
      await builder.eq("id", 5).delete();

      expect(fetch.lastMethod).toBe("DELETE");
      expect(fetch.lastUrl).toContain("id=eq.5");
    });

    it("should combine filters before and after .delete()", async () => {
      await builder
        .eq("status", "deleted")
        .delete()
        .lt("created_at", "2023-01-01");

      expect(fetch.lastMethod).toBe("DELETE");
      expect(fetch.lastUrl).toContain("status=eq.deleted");
      expect(fetch.lastUrl).toContain("created_at=lt.2023-01-01");
    });
  });

  describe("Insert Chaining", () => {
    it("should allow chaining .select() after .insert()", async () => {
      await builder
        .insert({ name: "John", email: "john@example.com" })
        .select("id, name");

      expect(fetch.lastMethod).toBe("POST");
      expect(fetch.lastBody).toEqual({
        name: "John",
        email: "john@example.com",
      });
    });

    it("should allow inserting multiple rows with chaining", async () => {
      const users = [
        { name: "Alice", email: "alice@example.com" },
        { name: "Bob", email: "bob@example.com" },
      ];
      await builder.insert(users).select("*");

      expect(fetch.lastMethod).toBe("POST");
      expect(fetch.lastBody).toEqual(users);
    });
  });

  describe("Complex Chaining Patterns", () => {
    it("should support update with order and limit chaining", async () => {
      await builder
        .update({ featured: true })
        .eq("status", "published")
        .order("created_at", { ascending: false })
        .limit(10);

      expect(fetch.lastMethod).toBe("PATCH");
      expect(fetch.lastUrl).toContain("status=eq.published");
      expect(fetch.lastUrl).toContain("order=created_at.desc");
      expect(fetch.lastUrl).toContain("limit=10");
      expect(fetch.lastBody).toEqual({ featured: true });
    });

    it("should support delete with complex filters", async () => {
      await builder
        .delete()
        .or("status.eq.spam,status.eq.deleted")
        .lt("created_at", "2023-01-01")
        .limit(100);

      expect(fetch.lastMethod).toBe("DELETE");
      expect(fetch.lastUrl).toContain("or=");
      expect(fetch.lastUrl).toContain("created_at=lt.2023-01-01");
      expect(fetch.lastUrl).toContain("limit=100");
    });

    it("should support explicit .execute() call after chaining", async () => {
      await builder
        .update({ status: "archived" })
        .eq("active", false)
        .lt("last_login", "2022-01-01")
        .execute();

      expect(fetch.lastMethod).toBe("PATCH");
      expect(fetch.lastUrl).toContain("active=eq.false");
      expect(fetch.lastUrl).toContain("last_login=lt.2022-01-01");
      expect(fetch.lastBody).toEqual({ status: "archived" });
    });
  });

  describe("Batch Operations with Chaining", () => {
    it("should support insertMany with select", async () => {
      const users = [
        { name: "Alice", email: "alice@example.com" },
        { name: "Bob", email: "bob@example.com" },
      ];
      await builder.insertMany(users);

      expect(fetch.lastMethod).toBe("POST");
      expect(fetch.lastBody).toEqual(users);
    });

    it("should support updateMany with filters before", async () => {
      await builder.eq("status", "pending").updateMany({ status: "completed" });

      expect(fetch.lastMethod).toBe("PATCH");
      expect(fetch.lastUrl).toContain("status=eq.pending");
      expect(fetch.lastBody).toEqual({ status: "completed" });
    });

    it("should support deleteMany with filters before", async () => {
      await builder.eq("spam", true).deleteMany();

      expect(fetch.lastMethod).toBe("DELETE");
      expect(fetch.lastUrl).toContain("spam=eq.true");
    });
  });
});

describe("QueryBuilder - Count Queries (Supabase-compatible)", () => {
  let fetch: MockFetch;
  let builder: QueryBuilder;

  beforeEach(() => {
    fetch = new MockFetch();
    builder = new QueryBuilder(fetch, "tracker_data");
  });

  describe("select() with count option", () => {
    it("should add count=exact parameter when count option is set", async () => {
      fetch.mockContentRangeHeader = "0-999/50000";
      fetch.mockResponse = [{ id: 1 }, { id: 2 }];

      await builder.select("*", { count: "exact" }).execute();

      expect(fetch.lastUrl).toContain("count=exact");
    });

    it("should add count=planned parameter when count option is planned", async () => {
      fetch.mockContentRangeHeader = "0-999/50000";
      fetch.mockResponse = [{ id: 1 }];

      await builder.select("*", { count: "planned" }).execute();

      expect(fetch.lastUrl).toContain("count=planned");
    });

    it("should add count=estimated parameter when count option is estimated", async () => {
      fetch.mockContentRangeHeader = "0-999/50000";
      fetch.mockResponse = [{ id: 1 }];

      await builder.select("*", { count: "estimated" }).execute();

      expect(fetch.lastUrl).toContain("count=estimated");
    });

    it("should not add count parameter when count option is not set", async () => {
      fetch.mockResponse = [{ id: 1 }];

      await builder.select("*").execute();

      expect(fetch.lastUrl).not.toContain("count=");
    });
  });

  describe("Content-Range header parsing", () => {
    it("should return count from Content-Range header", async () => {
      fetch.mockContentRangeHeader = "0-999/50000";
      fetch.mockResponse = [{ id: 1 }, { id: 2 }];

      const { count, data } = await builder
        .select("*", { count: "exact" })
        .execute();

      expect(count).toBe(50000);
      expect(data).toEqual([{ id: 1 }, { id: 2 }]);
    });

    it("should return count even when data array has fewer rows", async () => {
      // This is the key test: server returns 1000 rows due to limit, but count is 50000
      fetch.mockContentRangeHeader = "0-999/50000";
      const mockData = Array.from({ length: 1000 }, (_, i) => ({ id: i }));
      fetch.mockResponse = mockData;

      const { count, data } = await builder
        .select("*", { count: "exact" })
        .eq("user_id", "abc-123")
        .execute();

      expect(count).toBe(50000);
      expect((data as any[]).length).toBe(1000);
    });

    it("should handle Content-Range with different formats", async () => {
      fetch.mockContentRangeHeader = "0-49/100";
      fetch.mockResponse = Array.from({ length: 50 }, (_, i) => ({ id: i }));

      const { count } = await builder.select("*", { count: "exact" }).execute();

      expect(count).toBe(100);
    });

    it("should return null count when Content-Range header is missing", async () => {
      fetch.mockContentRangeHeader = null;
      fetch.mockResponse = [{ id: 1 }, { id: 2 }];

      const { count, data } = await builder
        .select("*", { count: "exact" })
        .execute();

      // Falls back to array length when header missing
      expect(count).toBe(2);
      expect(data).toEqual([{ id: 1 }, { id: 2 }]);
    });
  });

  describe("head option (count only, no data)", () => {
    it("should return count with null data when head is true", async () => {
      fetch.mockContentRangeHeader = "0-999/50000";
      fetch.mockResponse = [{ id: 1 }];

      const { count, data } = await builder
        .select("*", { count: "exact", head: true })
        .eq("user_id", "abc-123")
        .execute();

      expect(count).toBe(50000);
      expect(data).toBeNull();
    });

    it("should still include count parameter in URL with head option", async () => {
      fetch.mockContentRangeHeader = "0-0/12345";
      fetch.mockResponse = [];

      await builder.select("*", { count: "exact", head: true }).execute();

      expect(fetch.lastUrl).toContain("count=exact");
    });
  });

  describe("count with filters", () => {
    it("should work with eq filter", async () => {
      fetch.mockContentRangeHeader = "0-99/500";
      fetch.mockResponse = Array.from({ length: 100 }, (_, i) => ({ id: i }));

      const { count } = await builder
        .select("*", { count: "exact" })
        .eq("status", "active")
        .execute();

      expect(fetch.lastUrl).toContain("status=eq.active");
      expect(fetch.lastUrl).toContain("count=exact");
      expect(count).toBe(500);
    });

    it("should work with not.is.null filter (original bug scenario)", async () => {
      fetch.mockContentRangeHeader = "0-999/50000";
      fetch.mockResponse = Array.from({ length: 1000 }, (_, i) => ({ id: i }));

      const { count } = await builder
        .select("*", { count: "exact", head: true })
        .eq("user_id", "target-user-id")
        .not("location", "is", null)
        .execute();

      expect(fetch.lastUrl).toContain("user_id=eq.target-user-id");
      expect(fetch.lastUrl).toContain("location=not.is.null");
      expect(fetch.lastUrl).toContain("count=exact");
      expect(count).toBe(50000);
    });

    it("should work with limit and offset", async () => {
      fetch.mockContentRangeHeader = "100-199/5000";
      fetch.mockResponse = Array.from({ length: 100 }, (_, i) => ({ id: i + 100 }));

      const { count, data } = await builder
        .select("*", { count: "exact" })
        .limit(100)
        .offset(100)
        .execute();

      expect(fetch.lastUrl).toContain("limit=100");
      expect(fetch.lastUrl).toContain("offset=100");
      expect(fetch.lastUrl).toContain("count=exact");
      expect(count).toBe(5000);
      expect((data as any[]).length).toBe(100);
    });
  });

  describe("count with single/maybeSingle", () => {
    it("should return server count with single() modifier", async () => {
      fetch.mockContentRangeHeader = "0-0/1";
      fetch.mockResponse = [{ id: 1, name: "Test" }];

      const { count, data } = await builder
        .select("*", { count: "exact" })
        .eq("id", 1)
        .single()
        .execute();

      expect(count).toBe(1);
      expect(data).toEqual({ id: 1, name: "Test" });
    });

    it("should return server count with maybeSingle() for no results", async () => {
      fetch.mockContentRangeHeader = "*/0";
      fetch.mockResponse = [];

      const { count, data, error } = await builder
        .select("*", { count: "exact" })
        .eq("id", 999)
        .maybeSingle()
        .execute();

      // When no rows found, count should be 0
      expect(count).toBe(0);
      expect(data).toBeNull();
      expect(error).toBeNull();
    });
  });
});
