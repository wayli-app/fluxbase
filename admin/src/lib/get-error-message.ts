export function getErrorMessage(error: unknown): string {
  const err = error as { response?: { data?: { error?: string; message?: string; detail?: string; title?: string } }; message?: string }
  return err?.response?.data?.error
    || err?.response?.data?.message
    || err?.response?.data?.detail
    || err?.response?.data?.title
    || err?.message
    || "Something went wrong"
}
