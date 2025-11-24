/**
 * Type definitions for the webhook processor
 */

export interface WebhookPayload {
  event: string;
  data: any;
  timestamp: string;
}

export interface ProcessedResult {
  success: boolean;
  event: string;
  message: string;
  processedAt: string;
}
