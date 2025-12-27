import { useState } from 'react'
import z from 'zod'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, getRouteApi } from '@tanstack/react-router'
import type { EmailProviderSettings } from '@fluxbase/sdk'
import {
  Mail,
  FileText,
  Send,
  RotateCcw,
  Loader2,
  Settings2,
  Eye,
  EyeOff,
  CheckCircle2,
} from 'lucide-react'
import { toast } from 'sonner'
import { apiClient } from '@/lib/api'
import { fluxbaseClient } from '@/lib/fluxbase-client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  OverridableSelect,
  SelectItem,
} from '@/components/admin/overridable-select'
import { OverridableSwitch } from '@/components/admin/overridable-switch'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { PromptDialog } from '@/components/prompt-dialog'

const emailSettingsSearchSchema = z.object({
  tab: z.string().optional().catch('configuration'),
})

export const Route = createFileRoute('/_authenticated/email-settings/')({
  validateSearch: emailSettingsSearchSchema,
  component: EmailSettingsPage,
})

const route = getRouteApi('/_authenticated/email-settings/')

// EmailProviderSettings type is imported from @fluxbase/sdk

interface EmailTemplate {
  id: string
  template_type: string
  subject: string
  html_body: string
  text_body?: string
  is_custom: boolean
  created_at: string
  updated_at: string
}

// Form state for editing provider settings
interface ProviderFormState {
  from_address: string
  from_name: string
  // SMTP
  smtp_host: string
  smtp_port: string
  smtp_username: string
  smtp_password: string
  smtp_tls: boolean
  // SendGrid
  sendgrid_api_key: string
  // Mailgun
  mailgun_api_key: string
  mailgun_domain: string
  // AWS SES
  ses_access_key: string
  ses_secret_key: string
  ses_region: string
}

function EmailSettingsPage() {
  const queryClient = useQueryClient()
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null)
  const [editingTemplate, setEditingTemplate] =
    useState<Partial<EmailTemplate> | null>(null)
  const [showResetConfirm, setShowResetConfirm] = useState(false)
  const [resetTemplateType, setResetTemplateType] = useState<string | null>(
    null
  )
  const [showTestEmailPrompt, setShowTestEmailPrompt] = useState(false)
  const [testTemplateType, setTestTemplateType] = useState<string | null>(null)

  // Provider settings form state
  const [formState, setFormState] = useState<ProviderFormState>({
    from_address: '',
    from_name: '',
    smtp_host: '',
    smtp_port: '587',
    smtp_username: '',
    smtp_password: '',
    smtp_tls: true,
    sendgrid_api_key: '',
    mailgun_api_key: '',
    mailgun_domain: '',
    ses_access_key: '',
    ses_secret_key: '',
    ses_region: 'us-east-1',
  })
  const [showPassword, setShowPassword] = useState(false)
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false)

  // Track which settings version the form was initialized from
  const [initializedFromDataUpdatedAt, setInitializedFromDataUpdatedAt] =
    useState<number | null>(null)

  // Fetch email settings using SDK
  const {
    data: settings,
    isLoading: settingsLoading,
    dataUpdatedAt,
  } = useQuery<EmailProviderSettings>({
    queryKey: ['email-provider-settings'],
    queryFn: () => fluxbaseClient.admin.settings.email.get(),
  })

  // Initialize form state when settings are first loaded or refetched
  if (settings && dataUpdatedAt !== initializedFromDataUpdatedAt) {
    setInitializedFromDataUpdatedAt(dataUpdatedAt)
    setFormState({
      from_address: settings.from_address || '',
      from_name: settings.from_name || '',
      smtp_host: settings.smtp_host || '',
      smtp_port: String(settings.smtp_port || 587),
      smtp_username: settings.smtp_username || '',
      smtp_password: '', // Never populate password from server
      smtp_tls: settings.smtp_tls ?? true,
      sendgrid_api_key: '',
      mailgun_api_key: '',
      mailgun_domain: settings.mailgun_domain || '',
      ses_access_key: '',
      ses_secret_key: '',
      ses_region: settings.ses_region || 'us-east-1',
    })
    setHasUnsavedChanges(false)
  }

  // Fetch email templates
  const { data: templates, isLoading: templatesLoading } = useQuery<
    EmailTemplate[]
  >({
    queryKey: ['email-templates'],
    queryFn: async () => {
      const response = await apiClient.get('/api/v1/admin/email/templates')
      return response.data
    },
  })

  // Update email settings mutation using SDK
  const updateSettingsMutation = useMutation({
    mutationFn: (
      data: Parameters<typeof fluxbaseClient.admin.settings.email.update>[0]
    ) => fluxbaseClient.admin.settings.email.update(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['email-provider-settings'] })
      setHasUnsavedChanges(false)
      toast.success('Email settings updated')
    },
    onError: (error: unknown) => {
      if (error && typeof error === 'object' && 'response' in error) {
        const err = error as {
          response?: {
            status?: number
            data?: { code?: string; error?: string }
          }
        }
        if (
          err.response?.status === 409 &&
          err.response?.data?.code === 'ENV_OVERRIDE'
        ) {
          toast.error(
            'This setting is controlled by an environment variable and cannot be changed'
          )
          return
        }
        if (err.response?.data?.error) {
          toast.error(err.response.data.error)
          return
        }
      }
      toast.error('Failed to update email settings')
    },
  })

  // Test email settings mutation using SDK
  const testSettingsMutation = useMutation({
    mutationFn: (email: string) =>
      fluxbaseClient.admin.settings.email.test(email),
    onSuccess: () => {
      toast.success('Test email sent successfully')
    },
    onError: (error: unknown) => {
      if (error && typeof error === 'object' && 'response' in error) {
        const err = error as {
          response?: { data?: { error?: string; details?: string } }
        }
        if (err.response?.data?.details) {
          toast.error(`Failed to send test email: ${err.response.data.details}`)
          return
        }
      }
      toast.error('Failed to send test email')
    },
  })

  // Update template mutation
  const updateTemplateMutation = useMutation({
    mutationFn: async ({
      type,
      data,
    }: {
      type: string
      data: Partial<EmailTemplate>
    }) => {
      const response = await apiClient.put(
        `/api/v1/admin/email/templates/${type}`,
        data
      )
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['email-templates'] })
      setEditingTemplate(null)
      toast.success('Template updated successfully')
    },
    onError: () => {
      toast.error('Failed to update template')
    },
  })

  // Reset template mutation
  const resetTemplateMutation = useMutation({
    mutationFn: async (type: string) => {
      const response = await apiClient.post(
        `/api/v1/admin/email/templates/${type}/reset`
      )
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['email-templates'] })
      setEditingTemplate(null)
      toast.success('Template reset to default')
    },
    onError: () => {
      toast.error('Failed to reset template')
    },
  })

  // Test template mutation
  const testTemplateMutation = useMutation({
    mutationFn: async ({ type, email }: { type: string; email: string }) => {
      await apiClient.post(`/api/v1/admin/email/templates/${type}/test`, {
        recipient_email: email,
      })
    },
    onSuccess: () => {
      toast.success('Test email sent (when email service is configured)')
    },
    onError: () => {
      toast.error('Failed to send test email')
    },
  })

  const handleToggleEnabled = (checked: boolean) => {
    updateSettingsMutation.mutate({ enabled: checked })
  }

  const handleProviderChange = (provider: string) => {
    updateSettingsMutation.mutate({
      provider: provider as 'smtp' | 'sendgrid' | 'mailgun' | 'ses',
    })
  }

  const handleFormChange = (
    field: keyof ProviderFormState,
    value: string | boolean
  ) => {
    setFormState((prev) => ({ ...prev, [field]: value }))
    setHasUnsavedChanges(true)
  }

  const handleSaveProviderSettings = () => {
    const provider = settings?.provider || 'smtp'
    const data: Record<string, unknown> = {
      from_address: formState.from_address || undefined,
      from_name: formState.from_name || undefined,
    }

    if (provider === 'smtp') {
      data.smtp_host = formState.smtp_host || undefined
      data.smtp_port = formState.smtp_port
        ? parseInt(formState.smtp_port)
        : undefined
      data.smtp_username = formState.smtp_username || undefined
      if (formState.smtp_password) {
        data.smtp_password = formState.smtp_password
      }
      data.smtp_tls = formState.smtp_tls
    } else if (provider === 'sendgrid') {
      if (formState.sendgrid_api_key) {
        data.sendgrid_api_key = formState.sendgrid_api_key
      }
    } else if (provider === 'mailgun') {
      if (formState.mailgun_api_key) {
        data.mailgun_api_key = formState.mailgun_api_key
      }
      data.mailgun_domain = formState.mailgun_domain || undefined
    } else if (provider === 'ses') {
      if (formState.ses_access_key) {
        data.ses_access_key = formState.ses_access_key
      }
      if (formState.ses_secret_key) {
        data.ses_secret_key = formState.ses_secret_key
      }
      data.ses_region = formState.ses_region || undefined
    }

    updateSettingsMutation.mutate(data)
  }

  const handleTestConfiguration = () => {
    setTestTemplateType('config')
    setShowTestEmailPrompt(true)
  }

  const handleEditTemplate = (template: EmailTemplate) => {
    setSelectedTemplate(template.template_type)
    setEditingTemplate({
      subject: template.subject,
      html_body: template.html_body,
      text_body: template.text_body,
    })
  }

  const handleSaveTemplate = () => {
    if (!selectedTemplate || !editingTemplate) return
    updateTemplateMutation.mutate({
      type: selectedTemplate,
      data: editingTemplate,
    })
  }

  const handleResetTemplate = (type: string) => {
    setResetTemplateType(type)
    setShowResetConfirm(true)
  }

  const handleTestTemplate = (type: string) => {
    setTestTemplateType(type)
    setShowTestEmailPrompt(true)
  }

  const isOverridden = (field: string) => {
    return settings?._overrides?.[field]?.is_overridden ?? false
  }

  const getEnvVar = (field: string) => {
    return settings?._overrides?.[field]?.env_var || ''
  }

  if (settingsLoading || templatesLoading) {
    return (
      <div className='flex h-full items-center justify-center'>
        <Loader2 className='text-muted-foreground h-8 w-8 animate-spin' />
      </div>
    )
  }

  const currentProvider = settings?.provider || 'smtp'

  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div>
        <h1 className='flex items-center gap-2 text-3xl font-bold tracking-tight'>
          <Mail className='h-8 w-8' />
          Email Settings
        </h1>
        <p className='text-muted-foreground mt-2'>
          Configure email service and customize email templates
        </p>
      </div>

      <Tabs
        value={search.tab || 'configuration'}
        onValueChange={(tab) => navigate({ search: { tab } })}
        className='space-y-4'
      >
        <TabsList>
          <TabsTrigger
            value='configuration'
            className='flex items-center gap-2'
          >
            <Mail className='h-4 w-4' />
            Configuration
          </TabsTrigger>
          <TabsTrigger value='templates' className='flex items-center gap-2'>
            <FileText className='h-4 w-4' />
            Email Templates
          </TabsTrigger>
        </TabsList>

        <TabsContent value='configuration' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle>Email Service Configuration</CardTitle>
              <CardDescription>
                Configure your email service provider and settings
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-6'>
              <OverridableSwitch
                id='email-enabled'
                label='Enable Email Service'
                description='Enable or disable email functionality'
                checked={settings?.enabled || false}
                onCheckedChange={handleToggleEnabled}
                override={settings?._overrides?.enabled}
                disabled={updateSettingsMutation.isPending}
              />

              <OverridableSelect
                id='email-provider'
                label='Email Provider'
                description='Select your email service provider'
                value={currentProvider}
                onValueChange={handleProviderChange}
                override={settings?._overrides?.provider}
                disabled={updateSettingsMutation.isPending}
              >
                <SelectItem value='smtp'>SMTP</SelectItem>
                <SelectItem value='sendgrid'>SendGrid</SelectItem>
                <SelectItem value='mailgun'>Mailgun</SelectItem>
                <SelectItem value='ses'>AWS SES</SelectItem>
              </OverridableSelect>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Settings2 className='h-5 w-5' />
                {currentProvider === 'smtp' && 'SMTP Settings'}
                {currentProvider === 'sendgrid' && 'SendGrid Settings'}
                {currentProvider === 'mailgun' && 'Mailgun Settings'}
                {currentProvider === 'ses' && 'AWS SES Settings'}
              </CardTitle>
              <CardDescription>
                Configure your {currentProvider.toUpperCase()} provider settings
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              {/* Common Fields */}
              <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
                <div className='space-y-2'>
                  <Label htmlFor='from_address'>
                    From Email
                    {isOverridden('from_address') && (
                      <Badge variant='outline' className='ml-2 text-xs'>
                        ENV: {getEnvVar('from_address')}
                      </Badge>
                    )}
                  </Label>
                  <Input
                    id='from_address'
                    placeholder='noreply@example.com'
                    value={formState.from_address}
                    onChange={(e) =>
                      handleFormChange('from_address', e.target.value)
                    }
                    disabled={isOverridden('from_address')}
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='from_name'>
                    From Name
                    {isOverridden('from_name') && (
                      <Badge variant='outline' className='ml-2 text-xs'>
                        ENV: {getEnvVar('from_name')}
                      </Badge>
                    )}
                  </Label>
                  <Input
                    id='from_name'
                    placeholder='My App'
                    value={formState.from_name}
                    onChange={(e) =>
                      handleFormChange('from_name', e.target.value)
                    }
                    disabled={isOverridden('from_name')}
                  />
                </div>
              </div>

              {/* SMTP Settings */}
              {currentProvider === 'smtp' && (
                <>
                  <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
                    <div className='space-y-2'>
                      <Label htmlFor='smtp_host'>
                        SMTP Host
                        {isOverridden('smtp_host') && (
                          <Badge variant='outline' className='ml-2 text-xs'>
                            ENV: {getEnvVar('smtp_host')}
                          </Badge>
                        )}
                      </Label>
                      <Input
                        id='smtp_host'
                        placeholder='smtp.example.com'
                        value={formState.smtp_host}
                        onChange={(e) =>
                          handleFormChange('smtp_host', e.target.value)
                        }
                        disabled={isOverridden('smtp_host')}
                      />
                    </div>
                    <div className='space-y-2'>
                      <Label htmlFor='smtp_port'>
                        SMTP Port
                        {isOverridden('smtp_port') && (
                          <Badge variant='outline' className='ml-2 text-xs'>
                            ENV: {getEnvVar('smtp_port')}
                          </Badge>
                        )}
                      </Label>
                      <Input
                        id='smtp_port'
                        type='number'
                        placeholder='587'
                        value={formState.smtp_port}
                        onChange={(e) =>
                          handleFormChange('smtp_port', e.target.value)
                        }
                        disabled={isOverridden('smtp_port')}
                      />
                    </div>
                    <div className='space-y-2'>
                      <Label htmlFor='smtp_username'>
                        Username
                        {isOverridden('smtp_username') && (
                          <Badge variant='outline' className='ml-2 text-xs'>
                            ENV: {getEnvVar('smtp_username')}
                          </Badge>
                        )}
                      </Label>
                      <Input
                        id='smtp_username'
                        placeholder='username'
                        value={formState.smtp_username}
                        onChange={(e) =>
                          handleFormChange('smtp_username', e.target.value)
                        }
                        disabled={isOverridden('smtp_username')}
                      />
                    </div>
                    <div className='space-y-2'>
                      <Label htmlFor='smtp_password'>
                        Password
                        {isOverridden('smtp_password') && (
                          <Badge variant='outline' className='ml-2 text-xs'>
                            ENV: {getEnvVar('smtp_password')}
                          </Badge>
                        )}
                        {settings?.smtp_password_set &&
                          !isOverridden('smtp_password') && (
                            <Badge variant='secondary' className='ml-2 text-xs'>
                              <CheckCircle2 className='mr-1 h-3 w-3' />
                              Set
                            </Badge>
                          )}
                      </Label>
                      <div className='relative'>
                        <Input
                          id='smtp_password'
                          type={showPassword ? 'text' : 'password'}
                          placeholder={
                            settings?.smtp_password_set
                              ? '••••••••'
                              : 'Enter password'
                          }
                          value={formState.smtp_password}
                          onChange={(e) =>
                            handleFormChange('smtp_password', e.target.value)
                          }
                          disabled={isOverridden('smtp_password')}
                        />
                        <Button
                          type='button'
                          variant='ghost'
                          size='sm'
                          className='absolute top-0 right-0 h-full px-3 py-2 hover:bg-transparent'
                          onClick={() => setShowPassword(!showPassword)}
                        >
                          {showPassword ? (
                            <EyeOff className='h-4 w-4' />
                          ) : (
                            <Eye className='h-4 w-4' />
                          )}
                        </Button>
                      </div>
                    </div>
                  </div>
                  <div className='flex items-center space-x-2'>
                    <Switch
                      id='smtp_tls'
                      checked={formState.smtp_tls}
                      onCheckedChange={(checked) =>
                        handleFormChange('smtp_tls', checked)
                      }
                      disabled={isOverridden('smtp_tls')}
                    />
                    <Label htmlFor='smtp_tls'>
                      Enable TLS
                      {isOverridden('smtp_tls') && (
                        <Badge variant='outline' className='ml-2 text-xs'>
                          ENV: {getEnvVar('smtp_tls')}
                        </Badge>
                      )}
                    </Label>
                  </div>
                </>
              )}

              {/* SendGrid Settings */}
              {currentProvider === 'sendgrid' && (
                <div className='space-y-2'>
                  <Label htmlFor='sendgrid_api_key'>
                    API Key
                    {isOverridden('sendgrid_api_key') && (
                      <Badge variant='outline' className='ml-2 text-xs'>
                        ENV: {getEnvVar('sendgrid_api_key')}
                      </Badge>
                    )}
                    {settings?.sendgrid_api_key_set &&
                      !isOverridden('sendgrid_api_key') && (
                        <Badge variant='secondary' className='ml-2 text-xs'>
                          <CheckCircle2 className='mr-1 h-3 w-3' />
                          Set
                        </Badge>
                      )}
                  </Label>
                  <div className='relative'>
                    <Input
                      id='sendgrid_api_key'
                      type={showPassword ? 'text' : 'password'}
                      placeholder={
                        settings?.sendgrid_api_key_set ? '••••••••' : 'SG.xxxxx'
                      }
                      value={formState.sendgrid_api_key}
                      onChange={(e) =>
                        handleFormChange('sendgrid_api_key', e.target.value)
                      }
                      disabled={isOverridden('sendgrid_api_key')}
                    />
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      className='absolute top-0 right-0 h-full px-3 py-2 hover:bg-transparent'
                      onClick={() => setShowPassword(!showPassword)}
                    >
                      {showPassword ? (
                        <EyeOff className='h-4 w-4' />
                      ) : (
                        <Eye className='h-4 w-4' />
                      )}
                    </Button>
                  </div>
                </div>
              )}

              {/* Mailgun Settings */}
              {currentProvider === 'mailgun' && (
                <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
                  <div className='space-y-2'>
                    <Label htmlFor='mailgun_api_key'>
                      API Key
                      {isOverridden('mailgun_api_key') && (
                        <Badge variant='outline' className='ml-2 text-xs'>
                          ENV: {getEnvVar('mailgun_api_key')}
                        </Badge>
                      )}
                      {settings?.mailgun_api_key_set &&
                        !isOverridden('mailgun_api_key') && (
                          <Badge variant='secondary' className='ml-2 text-xs'>
                            <CheckCircle2 className='mr-1 h-3 w-3' />
                            Set
                          </Badge>
                        )}
                    </Label>
                    <div className='relative'>
                      <Input
                        id='mailgun_api_key'
                        type={showPassword ? 'text' : 'password'}
                        placeholder={
                          settings?.mailgun_api_key_set
                            ? '••••••••'
                            : 'key-xxxxx'
                        }
                        value={formState.mailgun_api_key}
                        onChange={(e) =>
                          handleFormChange('mailgun_api_key', e.target.value)
                        }
                        disabled={isOverridden('mailgun_api_key')}
                      />
                      <Button
                        type='button'
                        variant='ghost'
                        size='sm'
                        className='absolute top-0 right-0 h-full px-3 py-2 hover:bg-transparent'
                        onClick={() => setShowPassword(!showPassword)}
                      >
                        {showPassword ? (
                          <EyeOff className='h-4 w-4' />
                        ) : (
                          <Eye className='h-4 w-4' />
                        )}
                      </Button>
                    </div>
                  </div>
                  <div className='space-y-2'>
                    <Label htmlFor='mailgun_domain'>
                      Domain
                      {isOverridden('mailgun_domain') && (
                        <Badge variant='outline' className='ml-2 text-xs'>
                          ENV: {getEnvVar('mailgun_domain')}
                        </Badge>
                      )}
                    </Label>
                    <Input
                      id='mailgun_domain'
                      placeholder='mg.example.com'
                      value={formState.mailgun_domain}
                      onChange={(e) =>
                        handleFormChange('mailgun_domain', e.target.value)
                      }
                      disabled={isOverridden('mailgun_domain')}
                    />
                  </div>
                </div>
              )}

              {/* AWS SES Settings */}
              {currentProvider === 'ses' && (
                <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
                  <div className='space-y-2'>
                    <Label htmlFor='ses_access_key'>
                      Access Key
                      {isOverridden('ses_access_key') && (
                        <Badge variant='outline' className='ml-2 text-xs'>
                          ENV: {getEnvVar('ses_access_key')}
                        </Badge>
                      )}
                      {settings?.ses_access_key_set &&
                        !isOverridden('ses_access_key') && (
                          <Badge variant='secondary' className='ml-2 text-xs'>
                            <CheckCircle2 className='mr-1 h-3 w-3' />
                            Set
                          </Badge>
                        )}
                    </Label>
                    <div className='relative'>
                      <Input
                        id='ses_access_key'
                        type={showPassword ? 'text' : 'password'}
                        placeholder={
                          settings?.ses_access_key_set
                            ? '••••••••'
                            : 'AKIAXXXXX'
                        }
                        value={formState.ses_access_key}
                        onChange={(e) =>
                          handleFormChange('ses_access_key', e.target.value)
                        }
                        disabled={isOverridden('ses_access_key')}
                      />
                      <Button
                        type='button'
                        variant='ghost'
                        size='sm'
                        className='absolute top-0 right-0 h-full px-3 py-2 hover:bg-transparent'
                        onClick={() => setShowPassword(!showPassword)}
                      >
                        {showPassword ? (
                          <EyeOff className='h-4 w-4' />
                        ) : (
                          <Eye className='h-4 w-4' />
                        )}
                      </Button>
                    </div>
                  </div>
                  <div className='space-y-2'>
                    <Label htmlFor='ses_secret_key'>
                      Secret Key
                      {isOverridden('ses_secret_key') && (
                        <Badge variant='outline' className='ml-2 text-xs'>
                          ENV: {getEnvVar('ses_secret_key')}
                        </Badge>
                      )}
                      {settings?.ses_secret_key_set &&
                        !isOverridden('ses_secret_key') && (
                          <Badge variant='secondary' className='ml-2 text-xs'>
                            <CheckCircle2 className='mr-1 h-3 w-3' />
                            Set
                          </Badge>
                        )}
                    </Label>
                    <div className='relative'>
                      <Input
                        id='ses_secret_key'
                        type={showPassword ? 'text' : 'password'}
                        placeholder={
                          settings?.ses_secret_key_set
                            ? '••••••••'
                            : 'Secret key'
                        }
                        value={formState.ses_secret_key}
                        onChange={(e) =>
                          handleFormChange('ses_secret_key', e.target.value)
                        }
                        disabled={isOverridden('ses_secret_key')}
                      />
                      <Button
                        type='button'
                        variant='ghost'
                        size='sm'
                        className='absolute top-0 right-0 h-full px-3 py-2 hover:bg-transparent'
                        onClick={() => setShowPassword(!showPassword)}
                      >
                        {showPassword ? (
                          <EyeOff className='h-4 w-4' />
                        ) : (
                          <Eye className='h-4 w-4' />
                        )}
                      </Button>
                    </div>
                  </div>
                  <div className='space-y-2'>
                    <Label htmlFor='ses_region'>
                      Region
                      {isOverridden('ses_region') && (
                        <Badge variant='outline' className='ml-2 text-xs'>
                          ENV: {getEnvVar('ses_region')}
                        </Badge>
                      )}
                    </Label>
                    <Input
                      id='ses_region'
                      placeholder='us-east-1'
                      value={formState.ses_region}
                      onChange={(e) =>
                        handleFormChange('ses_region', e.target.value)
                      }
                      disabled={isOverridden('ses_region')}
                    />
                  </div>
                </div>
              )}

              <div className='flex gap-2 pt-4'>
                <Button
                  onClick={handleSaveProviderSettings}
                  disabled={
                    updateSettingsMutation.isPending || !hasUnsavedChanges
                  }
                >
                  {updateSettingsMutation.isPending && (
                    <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                  )}
                  Save Settings
                </Button>
                <Button
                  variant='outline'
                  onClick={handleTestConfiguration}
                  disabled={testSettingsMutation.isPending}
                >
                  {testSettingsMutation.isPending && (
                    <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                  )}
                  <Send className='mr-2 h-4 w-4' />
                  Test Configuration
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value='templates' className='space-y-4'>
          {!selectedTemplate ? (
            <div className='grid gap-4 md:grid-cols-3'>
              {templates?.map((template) => (
                <Card key={template.template_type} className='relative'>
                  <CardHeader>
                    <CardTitle className='flex items-center justify-between'>
                      <span className='capitalize'>
                        {template.template_type.replace(/_/g, ' ')}
                      </span>
                      {template.is_custom && (
                        <Badge variant='secondary'>Custom</Badge>
                      )}
                    </CardTitle>
                    <CardDescription className='line-clamp-2'>
                      {template.subject}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className='space-y-2'>
                    <Button
                      variant='outline'
                      className='w-full'
                      onClick={() => handleEditTemplate(template)}
                    >
                      <FileText className='mr-2 h-4 w-4' />
                      Edit Template
                    </Button>
                    {template.is_custom && (
                      <Button
                        variant='outline'
                        className='w-full'
                        onClick={() =>
                          handleResetTemplate(template.template_type)
                        }
                        disabled={resetTemplateMutation.isPending}
                      >
                        <RotateCcw className='mr-2 h-4 w-4' />
                        Reset to Default
                      </Button>
                    )}
                    <Button
                      variant='outline'
                      className='w-full'
                      onClick={() => handleTestTemplate(template.template_type)}
                      disabled={testTemplateMutation.isPending}
                    >
                      <Send className='mr-2 h-4 w-4' />
                      Send Test Email
                    </Button>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <Card>
              <CardHeader>
                <div className='flex items-center justify-between'>
                  <div>
                    <CardTitle className='capitalize'>
                      Edit {selectedTemplate.replace(/_/g, ' ')} Template
                    </CardTitle>
                    <CardDescription>
                      Customize the email template with variables like{' '}
                      {'{{.AppName}}'}, {'{{.MagicLink}}'}, etc.
                    </CardDescription>
                  </div>
                  <Button
                    variant='outline'
                    onClick={() => {
                      setSelectedTemplate(null)
                      setEditingTemplate(null)
                    }}
                  >
                    Back to Templates
                  </Button>
                </div>
              </CardHeader>
              <CardContent className='space-y-4'>
                <div className='space-y-2'>
                  <Label htmlFor='subject'>Subject</Label>
                  <Input
                    id='subject'
                    value={editingTemplate?.subject || ''}
                    onChange={(e) =>
                      setEditingTemplate({
                        ...editingTemplate,
                        subject: e.target.value,
                      })
                    }
                  />
                </div>

                <div className='space-y-2'>
                  <Label htmlFor='html_body'>HTML Body</Label>
                  <Textarea
                    id='html_body'
                    value={editingTemplate?.html_body || ''}
                    onChange={(e) =>
                      setEditingTemplate({
                        ...editingTemplate,
                        html_body: e.target.value,
                      })
                    }
                    rows={15}
                    className='font-mono text-sm'
                  />
                </div>

                <div className='space-y-2'>
                  <Label htmlFor='text_body'>Text Body (Optional)</Label>
                  <Textarea
                    id='text_body'
                    value={editingTemplate?.text_body || ''}
                    onChange={(e) =>
                      setEditingTemplate({
                        ...editingTemplate,
                        text_body: e.target.value,
                      })
                    }
                    rows={10}
                    className='font-mono text-sm'
                  />
                </div>

                <div className='flex gap-2'>
                  <Button
                    onClick={handleSaveTemplate}
                    disabled={updateTemplateMutation.isPending}
                  >
                    {updateTemplateMutation.isPending && (
                      <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                    )}
                    Save Template
                  </Button>
                  <Button
                    variant='outline'
                    onClick={() => {
                      setSelectedTemplate(null)
                      setEditingTemplate(null)
                    }}
                  >
                    Cancel
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>

      {/* Reset Template Confirmation */}
      <ConfirmDialog
        open={showResetConfirm}
        onOpenChange={setShowResetConfirm}
        title='Reset Template'
        desc='Are you sure you want to reset this template to default? Any customizations will be lost.'
        confirmText='Reset'
        destructive
        isLoading={resetTemplateMutation.isPending}
        handleConfirm={() => {
          if (resetTemplateType) {
            resetTemplateMutation.mutate(resetTemplateType, {
              onSuccess: () => {
                setShowResetConfirm(false)
                setResetTemplateType(null)
              },
            })
          }
        }}
      />

      {/* Test Email Prompt */}
      <PromptDialog
        open={showTestEmailPrompt}
        onOpenChange={setShowTestEmailPrompt}
        title='Send Test Email'
        description='Enter an email address to send a test email.'
        placeholder='email@example.com'
        inputType='email'
        confirmText='Send Test'
        isLoading={
          testTemplateMutation.isPending || testSettingsMutation.isPending
        }
        validation={(value) => {
          if (!value) return 'Email is required'
          if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value))
            return 'Invalid email address'
          return null
        }}
        onConfirm={(email) => {
          if (testTemplateType === 'config') {
            testSettingsMutation.mutate(email, {
              onSuccess: () => {
                setShowTestEmailPrompt(false)
                setTestTemplateType(null)
              },
            })
          } else if (testTemplateType) {
            testTemplateMutation.mutate(
              { type: testTemplateType, email },
              {
                onSuccess: () => {
                  setShowTestEmailPrompt(false)
                  setTestTemplateType(null)
                },
              }
            )
          }
        }}
      />
    </div>
  )
}
