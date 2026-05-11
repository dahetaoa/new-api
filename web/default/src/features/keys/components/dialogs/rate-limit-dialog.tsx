/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useEffect, useMemo, useState } from 'react'
import { Gauge, Layers, Plus, Trash2, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ComboboxInput } from '@/components/ui/combobox-input'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { getTokenRateLimits, updateTokenRateLimits } from '../../api'
import type {
  ApiKey,
  ModelRateLimitRule,
  RateLimitRule,
  TokenRateLimitResponse,
} from '../../types'

type RateLimitDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  apiKey: ApiKey | null
}

const EMPTY_RULE: RateLimitRule = { rpm: 0, rph: 0, rpd: 0 }

const normalizeRule = (rule?: Partial<RateLimitRule>): RateLimitRule => ({
  rpm: Math.max(0, Number(rule?.rpm) || 0),
  rph: Math.max(0, Number(rule?.rph) || 0),
  rpd: Math.max(0, Number(rule?.rpd) || 0),
})

const hasLimit = (rule: RateLimitRule): boolean =>
  rule.rpm > 0 || rule.rph > 0 || rule.rpd > 0

type LimitInputsProps = {
  value: RateLimitRule
  onChange: (next: RateLimitRule) => void
  idPrefix: string
}

function LimitInputs(props: LimitInputsProps) {
  const { t } = useTranslation()
  const update = (field: keyof RateLimitRule, raw: string) => {
    const parsed = parseInt(raw, 10)
    props.onChange({
      ...props.value,
      [field]: Number.isFinite(parsed) && parsed > 0 ? parsed : 0,
    })
  }
  const fields: Array<{
    key: keyof RateLimitRule
    label: string
    placeholder: string
  }> = [
    { key: 'rpm', label: 'RPM', placeholder: t('Per minute') },
    { key: 'rph', label: 'RPH', placeholder: t('Per hour') },
    { key: 'rpd', label: 'RPD', placeholder: t('Per day') },
  ]
  return (
    <div className='grid grid-cols-1 gap-2 sm:grid-cols-3'>
      {fields.map((field) => {
        const inputId = `${props.idPrefix}-${field.key}`
        return (
          <div key={field.key} className='space-y-1'>
            <Label
              htmlFor={inputId}
              className='text-muted-foreground text-xs font-medium'
            >
              {field.label}
            </Label>
            <Input
              id={inputId}
              type='number'
              min={0}
              step={1}
              value={props.value[field.key] || ''}
              placeholder={field.placeholder}
              onChange={(e) => update(field.key, e.target.value)}
            />
          </div>
        )
      })}
    </div>
  )
}

export function RateLimitDialog(props: RateLimitDialogProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [totalRule, setTotalRule] = useState<RateLimitRule>(EMPTY_RULE)
  const [modelRules, setModelRules] = useState<ModelRateLimitRule[]>([])
  const [availableModels, setAvailableModels] = useState<string[]>([])
  const [selectedModel, setSelectedModel] = useState('')

  const tokenId = props.apiKey?.id

  const selectedModelNames = useMemo(
    () => new Set(modelRules.map((rule) => rule.model_name)),
    [modelRules]
  )

  const availableModelOptions = useMemo(
    () =>
      availableModels
        .filter((name) => !selectedModelNames.has(name))
        .map((name) => ({ value: name, label: name })),
    [availableModels, selectedModelNames]
  )

  useEffect(() => {
    if (!props.open || !tokenId) {
      return
    }
    let cancelled = false
    setLoading(true)
    setSelectedModel('')
    setTotalRule(EMPTY_RULE)
    setModelRules([])
    setAvailableModels([])
    getTokenRateLimits(tokenId)
      .then((res) => {
        if (cancelled) return
        if (!res.success || !res.data) {
          toast.error(res.message || t('Failed to load rate limits'))
          return
        }
        const data = res.data as TokenRateLimitResponse
        setTotalRule(normalizeRule(data.total))
        setModelRules(
          (data.models || []).map((rule) => ({
            model_name: rule.model_name,
            ...normalizeRule(rule),
          }))
        )
        setAvailableModels(
          Array.isArray(data.available_models) ? data.available_models : []
        )
      })
      .catch(() => {
        if (!cancelled) toast.error(t('Failed to load rate limits'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [props.open, tokenId, t])

  const addModelRule = (value: string) => {
    const name = value.trim()
    if (!name) return
    if (selectedModelNames.has(name)) {
      toast.error(t('This model is already in the rate-limit list'))
      setSelectedModel('')
      return
    }
    setModelRules((prev) => [...prev, { model_name: name, ...EMPTY_RULE }])
    setSelectedModel('')
  }

  const updateModelRule = (name: string, rule: RateLimitRule) => {
    setModelRules((prev) =>
      prev.map((item) =>
        item.model_name === name ? { ...item, ...normalizeRule(rule) } : item
      )
    )
  }

  const removeModelRule = (name: string) => {
    setModelRules((prev) => prev.filter((item) => item.model_name !== name))
  }

  const handleSave = async () => {
    if (!tokenId) return
    setSaving(true)
    try {
      const payload = {
        total: normalizeRule(totalRule),
        models: modelRules
          .map((rule) => ({
            model_name: rule.model_name.trim(),
            ...normalizeRule(rule),
          }))
          .filter((rule) => rule.model_name && hasLimit(rule)),
      }
      const res = await updateTokenRateLimits(tokenId, payload)
      if (res.success) {
        toast.success(t('Rate limits updated'))
        props.onOpenChange(false)
      } else {
        toast.error(res.message || t('Failed to update rate limits'))
      }
    } catch {
      toast.error(t('Failed to update rate limits'))
    } finally {
      setSaving(false)
    }
  }

  const totalEnabled = hasLimit(totalRule)

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent
        side='right'
        className='bg-background flex !h-dvh !w-screen max-w-none gap-0 overflow-hidden p-0 sm:!w-full sm:!max-w-[620px]'
      >
        <SheetHeader className='bg-background border-b px-4 py-3 text-start sm:px-5 sm:py-4'>
          <SheetTitle className='text-base sm:text-lg'>
            {t('Rate Limit Management')}
          </SheetTitle>
          <SheetDescription className='pr-6 text-xs sm:text-sm'>
            {props.apiKey?.name
              ? t('Configure rate limits for "{{name}}"', {
                  name: props.apiKey.name,
                })
              : t(
                  'Configure per-token and per-model request rate limits (RPM / RPH / RPD).'
                )}
          </SheetDescription>
        </SheetHeader>

        <div className='min-h-0 flex-1 space-y-4 overflow-y-auto overscroll-contain px-4 py-4 sm:space-y-5 sm:px-5 sm:py-5'>
          {loading ? (
            <div className='flex items-center justify-center py-10'>
              <Loader2 className='text-muted-foreground size-5 animate-spin' />
            </div>
          ) : (
            <>
              <section className='space-y-3 border-b pb-4 sm:space-y-4 sm:pb-5'>
                <div className='flex items-center justify-between gap-2'>
                  <div className='flex items-center gap-2.5'>
                    <span className='bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-lg'>
                      <Gauge className='size-4' />
                    </span>
                    <h3 className='text-sm font-semibold tracking-tight'>
                      {t('Overall token rate limit')}
                    </h3>
                  </div>
                  <Badge variant={totalEnabled ? 'default' : 'secondary'}>
                    {totalEnabled ? t('Enabled') : t('No limit')}
                  </Badge>
                </div>
                <LimitInputs
                  value={totalRule}
                  onChange={setTotalRule}
                  idPrefix='token-total'
                />
                <p className='text-muted-foreground text-xs'>
                  {t(
                    '0 means unlimited; the overall limit counts all model requests for this token.'
                  )}
                </p>
              </section>

              <section className='space-y-3 sm:space-y-4'>
                <div className='flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                  <div className='flex items-center gap-2.5'>
                    <span className='bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-lg'>
                      <Layers className='size-4' />
                    </span>
                    <h3 className='text-sm font-semibold tracking-tight'>
                      {t('Per-model rate limits')}
                    </h3>
                  </div>
                  <div className='flex w-full gap-2 sm:w-96'>
                    <div className='min-w-0 flex-1'>
                      <ComboboxInput
                        options={availableModelOptions}
                        value={selectedModel}
                        onValueChange={setSelectedModel}
                        placeholder={t('Select or enter a model')}
                        emptyText={t('No available models')}
                        allowCustomValue
                      />
                    </div>
                    <Button
                      type='button'
                      variant='outline'
                      size='icon'
                      onClick={() => addModelRule(selectedModel)}
                      disabled={!selectedModel.trim()}
                      aria-label={t('Add')}
                    >
                      <Plus className='size-4' />
                    </Button>
                  </div>
                </div>

                {modelRules.length === 0 ? (
                  <div className='rounded-md border border-dashed py-8 text-center'>
                    <p className='text-muted-foreground text-sm'>
                      {t(
                        'No model-level rate limits. All models are unrestricted.'
                      )}
                    </p>
                  </div>
                ) : (
                  <div className='space-y-2'>
                    {modelRules.map((rule) => (
                      <div
                        key={rule.model_name}
                        className='space-y-3 rounded-md border p-3'
                      >
                        <div className='flex items-center justify-between gap-2'>
                          <span
                            className='truncate text-sm font-medium'
                            title={rule.model_name}
                          >
                            {rule.model_name}
                          </span>
                          <Button
                            type='button'
                            variant='ghost'
                            size='icon-sm'
                            onClick={() => removeModelRule(rule.model_name)}
                            aria-label={t('Remove')}
                            className='text-destructive hover:text-destructive'
                          >
                            <Trash2 className='size-4' />
                          </Button>
                        </div>
                        <LimitInputs
                          value={rule}
                          onChange={(next) =>
                            updateModelRule(rule.model_name, next)
                          }
                          idPrefix={`model-${rule.model_name}`}
                        />
                      </div>
                    ))}
                  </div>
                )}
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'Only models in this list with non-zero limits are rate-limited; other models are unrestricted.'
                  )}
                </p>
              </section>
            </>
          )}
        </div>

        <SheetFooter className='bg-background grid grid-cols-2 gap-2 border-t px-3 py-3 sm:flex sm:flex-row sm:justify-end sm:px-5 sm:py-4'>
          <SheetClose
            render={
              <Button
                variant='outline'
                className='w-full sm:w-auto'
                disabled={saving}
              />
            }
          >
            {t('Cancel')}
          </SheetClose>
          <Button
            onClick={handleSave}
            disabled={saving || loading}
            className='w-full sm:w-auto'
          >
            {saving ? (
              <>
                <Loader2 className='size-4 animate-spin' />
                {t('Saving...')}
              </>
            ) : (
              t('Save')
            )}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
