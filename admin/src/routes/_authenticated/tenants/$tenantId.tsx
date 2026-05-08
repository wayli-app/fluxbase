import { useQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import {
  ArrowLeft,
  Building2,
  Calendar,
  Database,
  Fingerprint,
  Hash,
  RefreshCw,
  ShieldCheck,
  Users,
} from 'lucide-react'
import { format } from 'date-fns'
import { tenantsApi } from '@/lib/api'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'

export const Route = createFileRoute('/_authenticated/tenants/$tenantId')({
  component: TenantDetailPage,
})

function TenantDetailPage() {
  const { tenantId } = Route.useParams()
  const navigate = useNavigate()

  const { data: tenant, isLoading: tenantLoading } = useQuery({
    queryKey: ['tenant', tenantId],
    queryFn: () => tenantsApi.get(tenantId),
  })

  if (tenantLoading) {
    return (
      <div className='flex h-96 items-center justify-center'>
        <RefreshCw className='text-muted-foreground h-8 w-8 animate-spin' />
      </div>
    )
  }

  if (!tenant) {
    return (
      <div className='flex h-96 flex-col items-center justify-center gap-4'>
        <p className='text-muted-foreground'>Tenant not found</p>
        <Button variant='outline' onClick={() => navigate({ to: '/tenants' })}>
          <ArrowLeft className='mr-2 h-4 w-4' />
          Back to Tenants
        </Button>
      </div>
    )
  }

  return (
    <div className='flex h-full flex-col'>
      <div className='bg-background flex items-center justify-between border-b px-6 py-4'>
        <div className='flex items-center gap-3'>
          <Button
            variant='ghost'
            size='sm'
            onClick={() => navigate({ to: '/tenants' })}
          >
            <ArrowLeft className='mr-2 h-4 w-4' />
            Back
          </Button>
          <div className='bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg'>
            <Building2 className='text-primary h-5 w-5' />
          </div>
          <div>
            <h1 className='text-xl font-semibold'>{tenant.name}</h1>
            <p className='text-muted-foreground text-sm'>
              <code className='text-xs'>{tenant.slug}</code>
              {tenant.is_default && (
                <Badge variant='default' className='ml-2'>
                  Default
                </Badge>
              )}
            </p>
          </div>
        </div>
      </div>

      <div className='flex-1 overflow-auto p-6'>
        <div className='grid gap-6 md:grid-cols-2'>
          <Card>
            <CardHeader>
              <CardTitle>Tenant Information</CardTitle>
              <CardDescription>Basic details about this tenant</CardDescription>
            </CardHeader>
            <CardContent>
              <div className='space-y-4'>
                <div className='flex items-center gap-3'>
                  <Hash className='text-muted-foreground h-4 w-4' />
                  <div>
                    <p className='text-muted-foreground text-xs'>Tenant ID</p>
                    <p className='font-mono text-sm'>{tenant.id}</p>
                  </div>
                </div>
                <Separator />
                <div className='flex items-center gap-3'>
                  <Fingerprint className='text-muted-foreground h-4 w-4' />
                  <div>
                    <p className='text-muted-foreground text-xs'>Slug</p>
                    <p className='font-mono text-sm'>{tenant.slug}</p>
                  </div>
                </div>
                <Separator />
                <div className='flex items-center gap-3'>
                  <Database className='text-muted-foreground h-4 w-4' />
                  <div>
                    <p className='text-muted-foreground text-xs'>Type</p>
                    <p className='text-sm'>
                      {tenant.is_default ? 'Default (shared database)' : 'Named tenant'}
                    </p>
                  </div>
                </div>
                <Separator />
                <div className='flex items-center gap-3'>
                  <Calendar className='text-muted-foreground h-4 w-4' />
                  <div>
                    <p className='text-muted-foreground text-xs'>Created</p>
                    <p className='text-sm'>
                      {format(new Date(tenant.created_at), 'PPPpp')}
                    </p>
                  </div>
                </div>
                {tenant.updated_at && (
                  <>
                    <Separator />
                    <div className='flex items-center gap-3'>
                      <Calendar className='text-muted-foreground h-4 w-4' />
                      <div>
                        <p className='text-muted-foreground text-xs'>Last Updated</p>
                        <p className='text-sm'>
                          {format(new Date(tenant.updated_at), 'PPPpp')}
                        </p>
                      </div>
                    </div>
                  </>
                )}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Quick Stats</CardTitle>
              <CardDescription>Summary of tenant resources</CardDescription>
            </CardHeader>
            <CardContent>
              <div className='space-y-4'>
                <div className='flex items-center gap-3'>
                  <Users className='text-muted-foreground h-4 w-4' />
                  <div>
                    <p className='text-muted-foreground text-xs'>Users</p>
                    <p className='text-sm'>{tenant.user_count ?? 0} user{tenant.user_count !== 1 ? 's' : ''}</p>
                  </div>
                </div>
                <Separator />
                <div className='flex items-center gap-3'>
                  <ShieldCheck className='text-muted-foreground h-4 w-4' />
                  <div>
                    <p className='text-muted-foreground text-xs'>Admins</p>
                    <p className='text-sm'>{tenant.admin_count ?? 0} admin{tenant.admin_count !== 1 ? 's' : ''}</p>
                  </div>
                </div>
                <Separator />
                <div className='flex items-center gap-3'>
                  <Building2 className='text-muted-foreground h-4 w-4' />
                  <div>
                    <p className='text-muted-foreground text-xs'>Status</p>
                    <Badge variant={tenant.deleted_at ? 'destructive' : 'default'}>
                      {tenant.deleted_at ? 'Deleted' : 'Active'}
                    </Badge>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
