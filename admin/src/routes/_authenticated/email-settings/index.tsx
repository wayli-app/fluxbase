import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Mail, FileText, Send, RotateCcw, Loader2, AlertCircle } from 'lucide-react'
import { toast } from 'sonner'
import { apiClient } from '@/lib/api'
import { useState } from 'react'

export const Route = createFileRoute('/_authenticated/email-settings/')({
  component: EmailSettingsPage,
})

interface EmailSettings {
  enabled: boolean
  provider: string
}

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

function EmailSettingsPage() {
  const queryClient = useQueryClient()
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null)
  const [editingTemplate, setEditingTemplate] = useState<Partial<EmailTemplate> | null>(null)

  // Fetch email settings
  const { data: settings, isLoading: settingsLoading } = useQuery<EmailSettings>({
    queryKey: ['email-settings'],
    queryFn: async () => {
      const [enabled, provider] = await Promise.all([
        apiClient.get('/api/v1/admin/system/settings/app.email.enabled'),
        apiClient.get('/api/v1/admin/system/settings/app.email.provider'),
      ])
      return {
        enabled: enabled.data.value.value,
        provider: provider.data.value.value,
      }
    },
  })

  // Fetch email templates
  const { data: templates, isLoading: templatesLoading } = useQuery<EmailTemplate[]>({
    queryKey: ['email-templates'],
    queryFn: async () => {
      const response = await apiClient.get('/api/v1/admin/email/templates')
      return response.data
    },
  })

  // Update email settings mutation
  const updateSettingMutation = useMutation({
    mutationFn: async ({ key, value }: { key: string; value: boolean | string }) => {
      await apiClient.put(`/api/v1/admin/system/settings/${key}`, {
        value: { value },
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['email-settings'] })
      toast.success('Email settings updated')
    },
    onError: () => {
      toast.error('Failed to update email settings')
    },
  })

  // Update template mutation
  const updateTemplateMutation = useMutation({
    mutationFn: async ({ type, data }: { type: string; data: Partial<EmailTemplate> }) => {
      const response = await apiClient.put(`/api/v1/admin/email/templates/${type}`, data)
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
      const response = await apiClient.post(`/api/v1/admin/email/templates/${type}/reset`)
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
    updateSettingMutation.mutate({
      key: 'app.email.enabled',
      value: checked,
    })
  }

  const handleProviderChange = (provider: string) => {
    updateSettingMutation.mutate({
      key: 'app.email.provider',
      value: provider,
    })
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
    if (confirm('Are you sure you want to reset this template to default?')) {
      resetTemplateMutation.mutate(type)
    }
  }

  const handleTestTemplate = (type: string) => {
    const email = prompt('Enter email address to send test email:')
    if (email) {
      testTemplateMutation.mutate({ type, email })
    }
  }

  if (settingsLoading || templatesLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
          <Mail className="h-8 w-8" />
          Email Settings
        </h1>
        <p className="text-muted-foreground mt-2">
          Configure email service and customize email templates
        </p>
      </div>

      <Tabs defaultValue="configuration" className="space-y-4">
        <TabsList>
          <TabsTrigger value="configuration" className="flex items-center gap-2">
            <Mail className="h-4 w-4" />
            Configuration
          </TabsTrigger>
          <TabsTrigger value="templates" className="flex items-center gap-2">
            <FileText className="h-4 w-4" />
            Email Templates
          </TabsTrigger>
        </TabsList>

        <TabsContent value="configuration" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Email Service Configuration</CardTitle>
              <CardDescription>
                Configure your email service provider and settings
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label htmlFor="email-enabled">Enable Email Service</Label>
                  <p className="text-sm text-muted-foreground">
                    Enable or disable email functionality
                  </p>
                </div>
                <Switch
                  id="email-enabled"
                  checked={settings?.enabled || false}
                  onCheckedChange={handleToggleEnabled}
                  disabled={updateSettingMutation.isPending}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="email-provider">Email Provider</Label>
                <Select
                  value={settings?.provider || 'smtp'}
                  onValueChange={handleProviderChange}
                  disabled={updateSettingMutation.isPending}
                >
                  <SelectTrigger id="email-provider">
                    <SelectValue placeholder="Select provider" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="smtp">SMTP</SelectItem>
                    <SelectItem value="sendgrid">SendGrid</SelectItem>
                    <SelectItem value="ses">AWS SES</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-sm text-muted-foreground">
                  Configure provider settings via environment variables
                </p>
              </div>

              <div className="rounded-lg bg-muted p-4">
                <div className="flex gap-2">
                  <AlertCircle className="h-5 w-5 text-muted-foreground shrink-0 mt-0.5" />
                  <div className="text-sm space-y-1">
                    <p className="font-medium">Email Provider Configuration</p>
                    <p className="text-muted-foreground">
                      Email provider credentials should be configured via environment variables.
                      See documentation for specific provider requirements.
                    </p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="templates" className="space-y-4">
          {!selectedTemplate ? (
            <div className="grid gap-4 md:grid-cols-3">
              {templates?.map((template) => (
                <Card key={template.template_type} className="relative">
                  <CardHeader>
                    <CardTitle className="flex items-center justify-between">
                      <span className="capitalize">
                        {template.template_type.replace(/_/g, ' ')}
                      </span>
                      {template.is_custom && (
                        <Badge variant="secondary">Custom</Badge>
                      )}
                    </CardTitle>
                    <CardDescription className="line-clamp-2">
                      {template.subject}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <Button
                      variant="outline"
                      className="w-full"
                      onClick={() => handleEditTemplate(template)}
                    >
                      <FileText className="h-4 w-4 mr-2" />
                      Edit Template
                    </Button>
                    {template.is_custom && (
                      <Button
                        variant="outline"
                        className="w-full"
                        onClick={() => handleResetTemplate(template.template_type)}
                        disabled={resetTemplateMutation.isPending}
                      >
                        <RotateCcw className="h-4 w-4 mr-2" />
                        Reset to Default
                      </Button>
                    )}
                    <Button
                      variant="outline"
                      className="w-full"
                      onClick={() => handleTestTemplate(template.template_type)}
                      disabled={testTemplateMutation.isPending}
                    >
                      <Send className="h-4 w-4 mr-2" />
                      Send Test Email
                    </Button>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="capitalize">
                      Edit {selectedTemplate.replace(/_/g, ' ')} Template
                    </CardTitle>
                    <CardDescription>
                      Customize the email template with variables like {"{{.AppName}}"}, {"{{.MagicLink}}"}, etc.
                    </CardDescription>
                  </div>
                  <Button
                    variant="outline"
                    onClick={() => {
                      setSelectedTemplate(null)
                      setEditingTemplate(null)
                    }}
                  >
                    Back to Templates
                  </Button>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="subject">Subject</Label>
                  <Input
                    id="subject"
                    value={editingTemplate?.subject || ''}
                    onChange={(e) => setEditingTemplate({
                      ...editingTemplate,
                      subject: e.target.value,
                    })}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="html_body">HTML Body</Label>
                  <Textarea
                    id="html_body"
                    value={editingTemplate?.html_body || ''}
                    onChange={(e) => setEditingTemplate({
                      ...editingTemplate,
                      html_body: e.target.value,
                    })}
                    rows={15}
                    className="font-mono text-sm"
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="text_body">Text Body (Optional)</Label>
                  <Textarea
                    id="text_body"
                    value={editingTemplate?.text_body || ''}
                    onChange={(e) => setEditingTemplate({
                      ...editingTemplate,
                      text_body: e.target.value,
                    })}
                    rows={10}
                    className="font-mono text-sm"
                  />
                </div>

                <div className="flex gap-2">
                  <Button
                    onClick={handleSaveTemplate}
                    disabled={updateTemplateMutation.isPending}
                  >
                    {updateTemplateMutation.isPending && (
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    )}
                    Save Template
                  </Button>
                  <Button
                    variant="outline"
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
    </div>
  )
}
