import { Search, FileText, BarChart3, GitBranch, Globe, PenLine } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { WorkflowDefinition } from '@/entities/workflow'

export type TemplateDefinition = {
  id: string
  title: string
  description: string
  icon: LucideIcon
  color: string
  tags: string[]
  difficulty: 'Beginner' | 'Intermediate'
  workflow: WorkflowDefinition
}

export const templates: TemplateDefinition[] = [
  {
    id: 'basic-rag-agent',
    title: 'Basic RAG Agent',
    description: 'Retrieve relevant web content and generate contextual responses.',
    icon: Search,
    color: 'bg-teal-500/10 text-teal-500',
    tags: ['RAG', 'Web'],
    difficulty: 'Beginner',
    workflow: {
      name: 'Basic RAG Agent',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'User Query',
            prompt: 'Enter your question',
            description: 'Accepts the user question to research',
          },
        },
        {
          id: 'rag_agent',
          type: 'agent',
          config: {
            label: 'RAG Agent',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt: 'You are a helpful research assistant. Use the get_webpage tool to retrieve relevant information, then synthesize a clear, well-sourced answer.',
            prompt: '{{user_input}}',
            tools: ['get_webpage'],
            description: 'Fetches web content and generates a contextual answer',
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Response',
            description: 'Final answer with sourced information',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'rag_agent' },
        { from: 'rag_agent', to: 'output' },
      ],
    },
  },
  {
    id: 'content-summarizer',
    title: 'Content Summarizer',
    description: 'Extract key points from long articles or documents.',
    icon: FileText,
    color: 'bg-blue-500/10 text-blue-500',
    tags: ['NLP', 'Writing'],
    difficulty: 'Beginner',
    workflow: {
      name: 'Content Summarizer',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'Content Input',
            prompt: 'Paste article or document text',
            description: 'The text content to summarize',
          },
        },
        {
          id: 'summarizer',
          type: 'agent',
          config: {
            label: 'Summarizer',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt: 'You are an expert summarizer. Extract the key points, main arguments, and conclusions from the provided text. Format your summary with bullet points for key takeaways and a brief narrative overview.',
            prompt: 'Summarize the following content:\n\n{{user_input}}',
            tools: [],
            description: 'Extracts key points and generates a structured summary',
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Summary',
            description: 'Structured summary with key takeaways',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'summarizer' },
        { from: 'summarizer', to: 'output' },
      ],
    },
  },
  {
    id: 'sentiment-analyzer',
    title: 'Sentiment Analyzer',
    description: 'Classify tone and sentiment from text input.',
    icon: BarChart3,
    color: 'bg-rose-500/10 text-rose-500',
    tags: ['NLP', 'Analysis'],
    difficulty: 'Beginner',
    workflow: {
      name: 'Sentiment Analyzer',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'Text Input',
            prompt: 'Enter text to analyze',
            description: 'Text for sentiment analysis',
          },
        },
        {
          id: 'analyzer',
          type: 'agent',
          config: {
            label: 'Sentiment Analyzer',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt: 'You are a sentiment analysis expert. Analyze the given text and return a JSON object with: sentiment (positive/negative/neutral/mixed), confidence (0-1), key_phrases (array of influential phrases), and explanation (brief reasoning).',
            prompt: 'Analyze the sentiment of this text:\n\n{{user_input}}',
            tools: [],
            description: 'Classifies sentiment and extracts key emotional signals',
            output_extract: { mode: 'json', key: 'sentiment_result' },
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Analysis Result',
            description: 'Structured sentiment analysis result',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'analyzer' },
        { from: 'analyzer', to: 'output' },
      ],
    },
  },
  {
    id: 'data-pipeline',
    title: 'Data Pipeline',
    description: 'Process and classify incoming data streams.',
    icon: GitBranch,
    color: 'bg-indigo-500/10 text-indigo-500',
    tags: ['Data', 'Classification'],
    difficulty: 'Intermediate',
    workflow: {
      name: 'Data Pipeline',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'Raw Data',
            prompt: 'Paste raw data (JSON, CSV, or plain text)',
            description: 'Raw data to be processed and classified',
          },
        },
        {
          id: 'classifier',
          type: 'agent',
          config: {
            label: 'Data Classifier',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt: 'You are a data processing pipeline. Parse the input data, identify its format and structure, classify each record into categories, and output a clean structured JSON result with: format_detected, record_count, categories (array), and classified_records (array of objects with original data + assigned category + confidence).',
            prompt: 'Process and classify the following data:\n\n{{user_input}}',
            tools: [],
            description: 'Parses, classifies, and structures raw data',
            output_extract: { mode: 'json', key: 'pipeline_result' },
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Classified Data',
            description: 'Structured and classified data output',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'classifier' },
        { from: 'classifier', to: 'output' },
      ],
    },
  },
  {
    id: 'web-research-agent',
    title: 'Web Research Agent',
    description: 'Search the web, gather sources, and synthesize findings.',
    icon: Globe,
    color: 'bg-violet-500/10 text-violet-500',
    tags: ['Research', 'Multi-step'],
    difficulty: 'Intermediate',
    workflow: {
      name: 'Web Research Agent',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'Research Topic',
            prompt: 'Enter your research question or topic',
            description: 'The topic to research across the web',
          },
        },
        {
          id: 'researcher',
          type: 'agent',
          config: {
            label: 'Web Researcher',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt: 'You are a thorough web researcher. Use the available tools to find relevant sources on the given topic. Retrieve at least 2-3 different web pages. For each source, note the URL and extract the key relevant information.',
            prompt: 'Research the following topic and gather information from multiple sources:\n\n{{user_input}}',
            tools: ['get_webpage', 'http_request'],
            description: 'Searches and retrieves content from multiple web sources',
          },
        },
        {
          id: 'synthesizer',
          type: 'agent',
          config: {
            label: 'Synthesizer',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt: 'You are a research synthesizer. Take the raw research findings and create a well-organized research brief. Include: an executive summary, key findings organized by theme, source citations, and areas of consensus or disagreement between sources.',
            prompt: 'Synthesize these research findings into a comprehensive brief:\n\n{{researcher}}',
            tools: [],
            description: 'Synthesizes raw research into a structured brief',
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Research Brief',
            description: 'Comprehensive research brief with citations',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'researcher' },
        { from: 'researcher', to: 'synthesizer' },
        { from: 'synthesizer', to: 'output' },
      ],
    },
  },
  {
    id: 'multi-step-writer',
    title: 'Multi-Step Writer',
    description: 'Generate structured long-form content in stages.',
    icon: PenLine,
    color: 'bg-amber-500/10 text-amber-500',
    tags: ['Writing', 'Chain'],
    difficulty: 'Intermediate',
    workflow: {
      name: 'Multi-Step Writer',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'Writing Brief',
            prompt: 'Describe what you want written (topic, audience, tone, length)',
            description: 'The writing brief with topic and requirements',
          },
        },
        {
          id: 'outliner',
          type: 'agent',
          config: {
            label: 'Outliner',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt: 'You are a content strategist. Create a detailed outline for the requested piece. Include: a compelling title, section headers, key points for each section, suggested word counts per section, and the overall narrative arc.',
            prompt: 'Create a detailed outline for:\n\n{{user_input}}',
            tools: [],
            description: 'Creates a structured outline with sections and key points',
          },
        },
        {
          id: 'writer',
          type: 'agent',
          config: {
            label: 'Writer',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt: 'You are a skilled writer. Follow the provided outline exactly and write the full content. Match the requested tone and audience. Write engaging, clear prose with smooth transitions between sections.',
            prompt: 'Write the full content following this outline:\n\n{{outliner}}',
            tools: [],
            description: 'Writes the full draft following the outline',
          },
        },
        {
          id: 'editor_agent',
          type: 'agent',
          config: {
            label: 'Editor',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt: "You are a professional editor. Review the draft for clarity, grammar, flow, and engagement. Fix any issues and polish the text. Preserve the author's voice while improving quality. Output the final edited version.",
            prompt: 'Edit and polish this draft:\n\n{{writer}}',
            tools: [],
            description: 'Reviews, polishes, and finalizes the written content',
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Final Content',
            description: 'Polished, publication-ready content',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'outliner' },
        { from: 'outliner', to: 'writer' },
        { from: 'writer', to: 'editor_agent' },
        { from: 'editor_agent', to: 'output' },
      ],
    },
  },
]
