/**
 * Helper functions for webhook processing
 */

import { WebhookPayload, ProcessedResult } from './types.ts';

export function validateWebhook(payload: WebhookPayload): boolean {
  return !!(payload.event && payload.data && payload.timestamp);
}

export function processWebhook(payload: WebhookPayload): ProcessedResult {
  // Process different event types
  let message = '';

  switch (payload.event) {
    case 'user.created':
      message = `User created: ${payload.data.email || 'unknown'}`;
      break;
    case 'user.updated':
      message = `User updated: ${payload.data.id || 'unknown'}`;
      break;
    case 'user.deleted':
      message = `User deleted: ${payload.data.id || 'unknown'}`;
      break;
    default:
      message = `Unknown event: ${payload.event}`;
  }

  return {
    success: true,
    event: payload.event,
    message,
    processedAt: new Date().toISOString(),
  };
}
