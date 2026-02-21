import type { Stage } from '@/lib/api/types'
import type { WorkflowDefinition } from '@/lib/serializer'

const MODEL = 'anthropic/claude-sonnet-4-20250514'

export type PipelineTemplate = {
  name: string
  description: string
  emoji: string
  stages: Omit<Stage, 'id'>[]
  workflows: WorkflowDefinition[]
}

export const PIPELINE_TEMPLATES: PipelineTemplate[] = [
  // ── 1. 콘텐츠 생성 & 검토 ──────────────────────────────────────────────────
  {
    name: '콘텐츠 생성 & 검토',
    description: 'AI가 콘텐츠를 생성하고, 사람이 검토한 뒤 발행하는 파이프라인',
    emoji: '✍️',
    stages: [
      { name: '콘텐츠 생성', type: 'workflow', config: { workflow_name: 'content-generator' } },
      { name: '담당자 검토', type: 'approval',  config: { message: '생성된 콘텐츠를 검토하고 승인 또는 거절해주세요.' } },
      { name: '콘텐츠 발행', type: 'workflow', config: { workflow_name: 'content-publisher' } },
    ],
    workflows: [
      {
        name: 'content-generator',
        version: 1,
        nodes: [
          { id: 'topic',  type: 'input', config: { label: '주제 또는 요청사항', value: '' } },
          { id: 'writer', type: 'agent', config: {
            label: '콘텐츠 작성',
            model: MODEL,
            system_prompt: '당신은 전문 콘텐츠 작성자입니다. 명확하고 매력적인 콘텐츠를 작성합니다.',
            prompt: '다음 주제 또는 요청에 맞는 고품질 콘텐츠를 작성해주세요:\n\n{{topic}}',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'topic',  to: 'writer' },
          { from: 'writer', to: 'output' },
        ],
      },
      {
        name: 'content-publisher',
        version: 1,
        nodes: [
          { id: 'content',   type: 'input', config: { label: '발행할 콘텐츠', value: '' } },
          { id: 'publisher', type: 'agent', config: {
            label: '발행 정리',
            model: MODEL,
            system_prompt: '당신은 콘텐츠 발행 전문가입니다.',
            prompt: '다음 콘텐츠를 발행 준비합니다. 제목, 요약, 본문을 포함한 최종 형식으로 정리해주세요:\n\n{{content}}',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'content',   to: 'publisher' },
          { from: 'publisher', to: 'output' },
        ],
      },
    ],
  },

  // ── 2. 일일 자동 리포트 ─────────────────────────────────────────────────────
  {
    name: '일일 자동 리포트',
    description: '매일 정해진 시간에 리포트를 생성하고 전달하는 파이프라인',
    emoji: '📊',
    stages: [
      { name: '매일 오전 9시', type: 'schedule', config: { cron: '0 9 * * *', timezone: 'Asia/Seoul' } },
      { name: '리포트 생성',   type: 'workflow', config: { workflow_name: 'report-generator' } },
      { name: '리포트 전달',   type: 'workflow', config: { workflow_name: 'report-sender' } },
    ],
    workflows: [
      {
        name: 'report-generator',
        version: 1,
        nodes: [
          { id: 'report_agent', type: 'agent', config: {
            label: '리포트 생성',
            model: MODEL,
            system_prompt: '당신은 전문적인 비즈니스 리포트 작성자입니다.',
            prompt: '오늘의 일일 업무 리포트를 작성해주세요. 다음 섹션을 포함해주세요:\n1. 오늘의 핵심 요약\n2. 주요 지표 및 현황\n3. 완료된 작업\n4. 이슈 및 리스크\n5. 내일의 액션 아이템',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'report_agent', to: 'output' },
        ],
      },
      {
        name: 'report-sender',
        version: 1,
        nodes: [
          { id: 'report_content', type: 'input', config: { label: '전달할 리포트', value: '' } },
          { id: 'formatter', type: 'agent', config: {
            label: '이메일 형식 정리',
            model: MODEL,
            system_prompt: '당신은 이메일 전문가입니다. 비즈니스 리포트를 이메일 형식으로 변환합니다.',
            prompt: '다음 리포트를 이메일 형식으로 정리해주세요. 수신자, 제목, 본문을 포함해주세요:\n\n{{report_content}}',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'report_content', to: 'formatter' },
          { from: 'formatter',      to: 'output' },
        ],
      },
    ],
  },

  // ── 3. 멀티 에이전트 리서치 ─────────────────────────────────────────────────
  {
    name: '멀티 에이전트 리서치',
    description: '여러 에이전트가 순차적으로 리서치·분석·요약을 처리하는 파이프라인',
    emoji: '🔍',
    stages: [
      { name: '정보 수집', type: 'workflow', config: { workflow_name: 'researcher' } },
      { name: '데이터 분석', type: 'workflow', config: { workflow_name: 'analyzer' } },
      { name: '결과 요약',  type: 'workflow', config: { workflow_name: 'summarizer' } },
      { name: '최종 검토',  type: 'approval', config: { message: '리서치 결과를 확인하고 승인해주세요.' } },
    ],
    workflows: [
      {
        name: 'researcher',
        version: 1,
        nodes: [
          { id: 'topic', type: 'input', config: { label: '리서치 주제', value: '' } },
          { id: 'research_agent', type: 'agent', config: {
            label: '정보 수집',
            model: MODEL,
            system_prompt: '당신은 전문 리서처입니다. 주어진 주제에 대해 철저하고 체계적인 조사를 수행합니다.',
            prompt: '다음 주제에 대해 최대한 상세히 리서치하고 정리해주세요:\n\n{{topic}}\n\n배경, 현황, 주요 사실, 데이터, 관련 사례를 포함해주세요.',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'topic',          to: 'research_agent' },
          { from: 'research_agent', to: 'output' },
        ],
      },
      {
        name: 'analyzer',
        version: 1,
        nodes: [
          { id: 'research_data', type: 'input', config: { label: '리서치 데이터', value: '' } },
          { id: 'analysis_agent', type: 'agent', config: {
            label: '데이터 분석',
            model: MODEL,
            system_prompt: '당신은 데이터 분석 전문가입니다. 정보에서 의미 있는 인사이트를 도출합니다.',
            prompt: '다음 리서치 결과를 분석하고 핵심 인사이트를 도출해주세요:\n\n{{research_data}}\n\n주요 패턴, 트렌드, 시사점을 포함해주세요.',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'research_data',  to: 'analysis_agent' },
          { from: 'analysis_agent', to: 'output' },
        ],
      },
      {
        name: 'summarizer',
        version: 1,
        nodes: [
          { id: 'analysis_data', type: 'input', config: { label: '분석 결과', value: '' } },
          { id: 'summary_agent', type: 'agent', config: {
            label: '결과 요약',
            model: MODEL,
            system_prompt: '당신은 복잡한 정보를 간결하게 요약하는 전문가입니다.',
            prompt: '다음 분석 내용을 간결하고 명확하게 요약해주세요:\n\n{{analysis_data}}\n\n핵심 포인트 5개와 최종 결론을 포함해주세요.',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'analysis_data', to: 'summary_agent' },
          { from: 'summary_agent', to: 'output' },
        ],
      },
    ],
  },

  // ── 4. 데이터 수집 & 분석 ───────────────────────────────────────────────────
  {
    name: '데이터 수집 & 분석',
    description: '데이터를 수집하고 변환한 뒤 AI로 분석하는 파이프라인',
    emoji: '🗄️',
    stages: [
      { name: '데이터 수집', type: 'workflow',  config: { workflow_name: 'data-collector' } },
      { name: '데이터 변환', type: 'transform', config: { expression: 'input' } },
      { name: 'AI 분석',    type: 'workflow',  config: { workflow_name: 'data-analyzer' } },
      { name: '결과 검증',  type: 'approval',  config: { message: '분석 결과를 검토하고 승인해주세요.' } },
    ],
    workflows: [
      {
        name: 'data-collector',
        version: 1,
        nodes: [
          { id: 'source', type: 'input', config: { label: '데이터 소스 또는 요청', value: '' } },
          { id: 'collector_agent', type: 'agent', config: {
            label: '데이터 수집',
            model: MODEL,
            system_prompt: '당신은 데이터 수집 전문가입니다. 요청된 데이터를 구조적으로 정리합니다.',
            prompt: '다음 요청에 따라 데이터를 수집하고 구조화해주세요:\n\n{{source}}\n\n데이터를 명확한 카테고리와 형식으로 정리해주세요.',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'source',          to: 'collector_agent' },
          { from: 'collector_agent', to: 'output' },
        ],
      },
      {
        name: 'data-analyzer',
        version: 1,
        nodes: [
          { id: 'dataset', type: 'input', config: { label: '수집된 데이터', value: '' } },
          { id: 'analyzer_agent', type: 'agent', config: {
            label: 'AI 분석',
            model: MODEL,
            system_prompt: '당신은 데이터 분석 전문가입니다. 데이터에서 의미 있는 패턴과 인사이트를 찾습니다.',
            prompt: '다음 데이터를 분석해주세요:\n\n{{dataset}}\n\n주요 패턴, 트렌드, 이상치, 개선 포인트를 포함한 분석 보고서를 작성해주세요.',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'dataset',        to: 'analyzer_agent' },
          { from: 'analyzer_agent', to: 'output' },
        ],
      },
    ],
  },

  // ── 5. 고객 문의 자동화 ─────────────────────────────────────────────────────
  {
    name: '고객 문의 자동화',
    description: '웹훅으로 문의를 받아 분류하고 AI 답변을 검토 후 발송하는 파이프라인',
    emoji: '💬',
    stages: [
      { name: '문의 수신', type: 'trigger',  config: { trigger_id: 'webhook-trigger' } },
      { name: '문의 분류', type: 'workflow', config: { workflow_name: 'inquiry-classifier' } },
      { name: '답변 생성', type: 'workflow', config: { workflow_name: 'reply-generator' } },
      { name: '답변 검토', type: 'approval', config: { message: 'AI가 작성한 답변을 검토하고 승인해주세요.' } },
      { name: '답변 발송', type: 'workflow', config: { workflow_name: 'reply-sender' } },
    ],
    workflows: [
      {
        name: 'inquiry-classifier',
        version: 1,
        nodes: [
          { id: 'inquiry', type: 'input', config: { label: '고객 문의 내용', value: '' } },
          { id: 'classifier_agent', type: 'agent', config: {
            label: '문의 분류',
            model: MODEL,
            system_prompt: '당신은 고객 문의를 정확하게 분류하는 전문가입니다.',
            prompt: '다음 고객 문의를 분류해주세요:\n\n{{inquiry}}\n\n카테고리: 기술지원 / 환불요청 / 배송문의 / 계정문의 / 일반문의 / 기타\n\n분류 결과, 이유, 우선순위(높음/보통/낮음)를 명시해주세요.',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'inquiry',           to: 'classifier_agent' },
          { from: 'classifier_agent',  to: 'output' },
        ],
      },
      {
        name: 'reply-generator',
        version: 1,
        nodes: [
          { id: 'inquiry',  type: 'input', config: { label: '고객 문의', value: '' } },
          { id: 'category', type: 'input', config: { label: '문의 분류 결과', value: '' } },
          { id: 'reply_agent', type: 'agent', config: {
            label: '답변 생성',
            model: MODEL,
            system_prompt: '당신은 친절하고 전문적인 고객 서비스 담당자입니다. 고객의 문제 해결에 집중합니다.',
            prompt: '분류: {{category}}\n고객 문의: {{inquiry}}\n\n위 문의에 대한 친절하고 전문적인 답변을 작성해주세요. 공감, 해결책, 추가 도움 제안을 포함해주세요.',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'inquiry',     to: 'reply_agent' },
          { from: 'category',    to: 'reply_agent' },
          { from: 'reply_agent', to: 'output' },
        ],
      },
      {
        name: 'reply-sender',
        version: 1,
        nodes: [
          { id: 'reply_content', type: 'input', config: { label: '발송할 답변', value: '' } },
          { id: 'sender_agent', type: 'agent', config: {
            label: '발송 준비',
            model: MODEL,
            system_prompt: '당신은 고객 커뮤니케이션 전문가입니다.',
            prompt: '다음 답변을 이메일 발송 형식으로 최종 정리해주세요:\n\n{{reply_content}}\n\n수신자 정보, 제목, 본문, 서명을 포함한 완성된 이메일 형식으로 작성해주세요.',
          }},
          { id: 'output', type: 'output', config: {} },
        ],
        edges: [
          { from: 'reply_content', to: 'sender_agent' },
          { from: 'sender_agent',  to: 'output' },
        ],
      },
    ],
  },
]
