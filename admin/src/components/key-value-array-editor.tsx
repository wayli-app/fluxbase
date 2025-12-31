import { useState } from 'react'
import { X } from 'lucide-react'
import { Button } from './ui/button'
import { Input } from './ui/input'

interface KeyValueArrayEditorProps {
  value: Record<string, string[]>
  onChange: (value: Record<string, string[]>) => void
  keyPlaceholder?: string
  valuePlaceholder?: string
  addButtonText?: string
}

export function KeyValueArrayEditor({
  value,
  onChange,
  keyPlaceholder = 'Key',
  valuePlaceholder = 'Value',
  addButtonText = 'Add Item',
}: KeyValueArrayEditorProps) {
  const [key, setKey] = useState('')
  const [itemValue, setItemValue] = useState('')

  const handleAdd = () => {
    if (key.trim() && itemValue.trim()) {
      const newValue = { ...value }
      if (newValue[key]) {
        if (!newValue[key].includes(itemValue.trim())) {
          newValue[key] = [...newValue[key], itemValue.trim()]
        }
      } else {
        newValue[key] = [itemValue.trim()]
      }
      onChange(newValue)
      setItemValue('')
    }
  }

  const handleRemove = (claimKey: string, valueIndex: number) => {
    const newValue = { ...value }
    newValue[claimKey] = newValue[claimKey].filter((_, i) => i !== valueIndex)
    if (newValue[claimKey].length === 0) {
      delete newValue[claimKey]
    }
    onChange(newValue)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      handleAdd()
    }
  }

  return (
    <div className='space-y-2'>
      <div className='grid grid-cols-[1fr_1fr_auto] gap-2'>
        <Input
          value={key}
          onChange={(e) => setKey(e.target.value)}
          placeholder={keyPlaceholder}
        />
        <Input
          value={itemValue}
          onChange={(e) => setItemValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={valuePlaceholder}
        />
        <Button type='button' onClick={handleAdd} variant='outline'>
          {addButtonText}
        </Button>
      </div>
      {Object.keys(value).length > 0 && (
        <div className='space-y-2'>
          {Object.entries(value).map(([claimKey, values]) => (
            <div key={claimKey} className='space-y-1 rounded-md border p-3'>
              <div className='text-sm font-semibold'>{claimKey}</div>
              <div className='space-y-1'>
                {values.map((val, idx) => (
                  <div
                    key={idx}
                    className='bg-muted flex items-center gap-2 rounded-md px-3 py-1'
                  >
                    <span className='flex-1 font-mono text-sm'>{val}</span>
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      onClick={() => handleRemove(claimKey, idx)}
                    >
                      <X className='h-4 w-4' />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
