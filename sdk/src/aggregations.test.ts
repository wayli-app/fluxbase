/**
 * Tests for aggregation and batch operations
 */

import { describe, it, expect } from "vitest";
import { QueryBuilder } from "./query-builder";
import { FluxbaseFetch } from "./fetch";

// Mock FluxbaseFetch
class MockFetch extends FluxbaseFetch {
  constructor() {
    super("http://localhost:8080", {});
  }

  // Override to capture the URL being called
  lastUrl: string = "";

  async get<T>(path: string): Promise<T> {
    this.lastUrl = path;
    return [] as T;
  }
}

describe("QueryBuilder Aggregations", () => {
  it("should build count query", () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    builder.count("*").execute();

    expect(fetch.lastUrl).toContain("select=count");
    expect(fetch.lastUrl).toContain("*");
  });

  it("should build count with specific column", () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    builder.count("id").execute();

    expect(fetch.lastUrl).toContain("select=count");
    expect(fetch.lastUrl).toContain("id");
  });

  it("should build sum query", () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    builder.sum("price").execute();

    expect(fetch.lastUrl).toContain("select=sum");
    expect(fetch.lastUrl).toContain("price");
  });

  it("should build avg query", () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    builder.avg("price").execute();

    expect(fetch.lastUrl).toContain("select=avg");
    expect(fetch.lastUrl).toContain("price");
  });

  it("should build min query", () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    builder.min("price").execute();

    expect(fetch.lastUrl).toContain("select=min");
    expect(fetch.lastUrl).toContain("price");
  });

  it("should build max query", () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    builder.max("price").execute();

    expect(fetch.lastUrl).toContain("select=max");
    expect(fetch.lastUrl).toContain("price");
  });

  it("should build group by query", () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    builder.count("*").groupBy("category").execute();

    expect(fetch.lastUrl).toContain("select=count");
    expect(fetch.lastUrl).toContain("group_by=category");
  });

  it("should build group by with multiple columns", () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    builder.count("*").groupBy(["category", "status"]).execute();

    expect(fetch.lastUrl).toContain("group_by=category");
    expect(fetch.lastUrl).toContain("status");
  });

  it("should combine aggregation with filters", () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    builder.count("*").eq("active", true).groupBy("category").execute();

    expect(fetch.lastUrl).toContain("select=count");
    expect(fetch.lastUrl).toContain("active=eq.true");
    expect(fetch.lastUrl).toContain("group_by=category");
  });
});

describe("QueryBuilder Count Response Parsing", () => {
  it("should extract count value from response data", async () => {
    const fetch = new MockFetch();
    // Mock response with count result
    fetch.get = async () => [{ count: 8867 }] as any;

    const builder = new QueryBuilder(fetch, "tracker_data");
    const result = await builder.count("*").execute();

    expect(result.count).toBe(8867);
    expect(result.error).toBeNull();
    expect(result.data).toEqual([{ count: 8867 }]);
  });

  it("should extract count value with filters", async () => {
    const fetch = new MockFetch();
    fetch.get = async () => [{ count: 42 }] as any;

    const builder = new QueryBuilder(fetch, "users");
    const result = await builder.count("*").eq("active", true).execute();

    expect(result.count).toBe(42);
  });

  it("should return 0 for empty count result", async () => {
    const fetch = new MockFetch();
    fetch.get = async () => [] as any;

    const builder = new QueryBuilder(fetch, "users");
    const result = await builder.count("*").execute();

    expect(result.count).toBe(0);
  });

  it("should return full array for count with groupBy", async () => {
    const fetch = new MockFetch();
    const groupedData = [
      { category: "electronics", count: 45 },
      { category: "books", count: 23 },
    ];
    fetch.get = async () => groupedData as any;

    const builder = new QueryBuilder(fetch, "products");
    const result = await builder.count("*").groupBy("category").execute();

    // With groupBy, count should be array length, not extracted from data
    expect(result.count).toBe(2);
    expect(result.data).toEqual(groupedData);
  });

  it("should extract count with specific column", async () => {
    const fetch = new MockFetch();
    fetch.get = async () => [{ count: 500 }] as any;

    const builder = new QueryBuilder(fetch, "orders");
    const result = await builder.count("completed_at").execute();

    expect(result.count).toBe(500);
  });
});

describe("QueryBuilder Batch Operations", () => {
  it("should have insertMany alias", async () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    // Mock post method
    fetch.post = async (path: string, body: unknown) => {
      expect(path).toBe("/api/v1/tables/products");
      expect(Array.isArray(body)).toBe(true);
      return [] as any;
    };

    await builder.insertMany([{ name: "Product 1" }, { name: "Product 2" }]);
  });

  it("should have updateMany alias", async () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    // Mock patch method
    fetch.patch = async (path: string, body: unknown) => {
      expect(path).toContain("/api/v1/tables/products");
      expect(body).toEqual({ discount: 10 });
      return [] as any;
    };

    await builder.eq("category", "electronics").updateMany({ discount: 10 });
  });

  it("should have deleteMany alias", async () => {
    const fetch = new MockFetch();
    const builder = new QueryBuilder(fetch, "products");

    // Mock delete method
    fetch.delete = async (path: string) => {
      expect(path).toContain("/api/v1/tables/products");
      expect(path).toContain("active=eq.false");
      return undefined as any;
    };

    await builder.eq("active", false).deleteMany();
  });
});
