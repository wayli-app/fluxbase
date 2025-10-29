import z from 'zod'
import { createFileRoute } from '@tanstack/react-router'
import { Tables } from '@/features/tables'

const tableSearchSchema = z.object({
  table: z.string().optional(),
  page: z.number().optional().catch(1),
  pageSize: z.number().optional().catch(10),
  filter: z.string().optional().catch(''),
})

export const Route = createFileRoute('/_authenticated/tables/')({
  validateSearch: tableSearchSchema,
  component: Tables,
})
