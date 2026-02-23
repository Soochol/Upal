import { useState } from 'react'
import { Loader2, Check, AlertCircle } from 'lucide-react'
import { Textarea } from '@/shared/ui/textarea'
import { KeywordTagInput } from '@/shared/ui/KeywordTagInput'
import type { PipelineContext } from '@/shared/types'

const LANGUAGE_OPTIONS = ['한국어', '영어', '일본어', '중국어']

const DEFAULT_CONTEXT: PipelineContext = {
  purpose: '',
  target_audience: '',
  tone_style: '',
  focus_keywords: [],
  exclude_keywords: [],
  language: '한국어',
}

type Props = {
  initialContext?: PipelineContext
  onSave: (context: PipelineContext) => Promise<void>
}

export function EditorialBriefForm({ initialContext, onSave }: Props) {
  const [draft, setDraft] = useState<PipelineContext>(initialContext ?? DEFAULT_CONTEXT)
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')

  const handleSave = async () => {
    setSaveStatus('saving')
    try {
      await onSave(draft)
      setSaveStatus('saved')
      setTimeout(() => setSaveStatus('idle'), 2000)
    } catch {
      setSaveStatus('error')
      setTimeout(() => setSaveStatus('idle'), 3000)
    }
  }

  const field = (key: keyof PipelineContext) => ({
    value: (draft[key] as string) ?? '',
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setDraft({ ...draft, [key]: e.target.value }),
  })

  return (
    <div className="space-y-5">
      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">주제 / 목적</label>
        <Textarea
          rows={3}
          placeholder="이 파이프라인이 다루는 주제와 목적을 설명해 주세요..."
          className="resize-none text-sm"
          {...field('purpose')}
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">타겟 독자</label>
        <input
          type="text"
          placeholder="예: IT 업계 종사자, AI에 관심 있는 일반인"
          className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none
            focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
          {...field('target_audience')}
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">톤 / 스타일</label>
        <input
          type="text"
          placeholder="예: 기술적이지만 접근 쉬운 한국어"
          className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none
            focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
          {...field('tone_style')}
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">
          포커스 키워드
          <span className="ml-1 font-normal text-muted-foreground/70">(이 키워드를 중심으로 선별)</span>
        </label>
        <KeywordTagInput
          keywords={draft.focus_keywords}
          onChange={(kws) => setDraft({ ...draft, focus_keywords: kws })}
          placeholder="LLM, AI 모델, 빅테크..."
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">
          제외 키워드
          <span className="ml-1 font-normal text-muted-foreground/70">(이 키워드가 포함된 아이템 제외)</span>
        </label>
        <KeywordTagInput
          keywords={draft.exclude_keywords}
          onChange={(kws) => setDraft({ ...draft, exclude_keywords: kws })}
          placeholder="게임, 스포츠..."
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">언어</label>
        <select
          value={draft.language}
          onChange={(e) => setDraft({ ...draft, language: e.target.value })}
          className="rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none
            focus:ring-1 focus:ring-ring"
        >
          {LANGUAGE_OPTIONS.map((lang) => (
            <option key={lang} value={lang}>{lang}</option>
          ))}
        </select>
      </div>

      <div className="flex items-center gap-3 pt-1">
        <button
          onClick={handleSave}
          disabled={saveStatus === 'saving'}
          className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
            bg-foreground text-background hover:opacity-90 transition-opacity
            disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
        >
          {saveStatus === 'saving' ? (
            <><Loader2 className="h-3.5 w-3.5 animate-spin" />저장 중…</>
          ) : saveStatus === 'saved' ? (
            <><Check className="h-3.5 w-3.5" />저장됨</>
          ) : saveStatus === 'error' ? (
            <><AlertCircle className="h-3.5 w-3.5" />저장 실패</>
          ) : '저장'}
        </button>
      </div>
    </div>
  )
}
