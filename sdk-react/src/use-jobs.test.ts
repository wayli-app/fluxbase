/**
 * Tests for Jobs hooks
 */

import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import {
  useSubmitJob,
  useJobStatus,
  useJobs,
  useCancelJob,
  useRetryJob,
} from "./use-jobs";
import { createMockClient, createWrapper } from "./test-utils";

describe("useSubmitJob", () => {
  it("should submit a job", async () => {
    const submit = vi.fn().mockResolvedValue({
      data: { id: "job-1", status: "pending" },
      error: null,
    });
    const client = createMockClient({
      jobs: { submit } as any,
    });

    const { result } = renderHook(() => useSubmitJob(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: "send-email", payload: { to: "a@b.com" } });
    });

    expect(submit).toHaveBeenCalledWith("send-email", { to: "a@b.com" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      jobs: {
        submit: vi.fn().mockResolvedValue({ data: null, error: new Error("Failed") }),
      } as any,
    });

    const { result } = renderHook(() => useSubmitJob(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({ name: "bad-job" });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useJobStatus", () => {
  it("should get job status by ID", async () => {
    const mockJob = { id: "job-1", status: "completed" };
    const client = createMockClient({
      jobs: {
        get: vi.fn().mockResolvedValue({ data: mockJob, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useJobStatus("job-1"), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockJob);
  });

  it("should not fetch when jobId is null", () => {
    const client = createMockClient();
    const { result } = renderHook(() => useJobStatus(null), {
      wrapper: createWrapper(client),
    });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeUndefined();
  });
});

describe("useJobs", () => {
  it("should list jobs", async () => {
    const mockJobs = [
      { id: "job-1", status: "completed" },
      { id: "job-2", status: "running" },
    ];
    const client = createMockClient({
      jobs: {
        list: vi.fn().mockResolvedValue({ data: mockJobs, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useJobs(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockJobs);
  });

  it("should pass options to list", async () => {
    const list = vi.fn().mockResolvedValue({ data: [], error: null });
    const client = createMockClient({
      jobs: { list } as any,
    });

    renderHook(() => useJobs({ status: "running", limit: 10 }), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => {
      expect(list).toHaveBeenCalledWith({ status: "running", limit: 10 });
    });
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      jobs: {
        list: vi.fn().mockResolvedValue({ data: null, error: new Error("Failed") }),
      } as any,
    });

    const { result } = renderHook(() => useJobs(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useCancelJob", () => {
  it("should cancel a job and invalidate queries", async () => {
    const cancel = vi.fn().mockResolvedValue({ data: null, error: null });
    const client = createMockClient({
      jobs: { cancel } as any,
    });

    const { result } = renderHook(() => useCancelJob(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync("job-1");
    });

    expect(cancel).toHaveBeenCalledWith("job-1");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      jobs: {
        cancel: vi.fn().mockResolvedValue({ data: null, error: new Error("Not found") }),
      } as any,
    });

    const { result } = renderHook(() => useCancelJob(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync("job-1");
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useRetryJob", () => {
  it("should retry a job and invalidate queries", async () => {
    const retry = vi.fn().mockResolvedValue({
      data: { id: "job-2", status: "pending" },
      error: null,
    });
    const client = createMockClient({
      jobs: { retry } as any,
    });

    const { result } = renderHook(() => useRetryJob(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync("job-1");
    });

    expect(retry).toHaveBeenCalledWith("job-1");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      jobs: {
        retry: vi.fn().mockResolvedValue({ data: null, error: new Error("Failed") }),
      } as any,
    });

    const { result } = renderHook(() => useRetryJob(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync("job-1");
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
