/**
 * Shared validation utilities for edge functions
 */

export function validateEmail(email: string): boolean {
  const regex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return regex.test(email);
}

export function validateRequired(data: any, fields: string[]): string[] {
  const missing: string[] = [];
  for (const field of fields) {
    if (!data[field]) {
      missing.push(field);
    }
  }
  return missing;
}

export function sanitizeString(str: string): string {
  return str.trim().replace(/[<>]/g, '');
}
