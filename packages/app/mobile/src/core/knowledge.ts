import { authenticatedRequest } from './api';

export type KnowledgeBase = {
  id: string;
  name: string;
  description?: string | null;
  icon?: string | null;
  color?: string | null;
  doc_count?: number | null;
  image_count?: number | null;
  chat_enabled?: boolean | null;
  is_default?: boolean | null;
  created_at?: string | null;
  updated_at?: string | null;
};

export type CreateKnowledgeBaseInput = {
  name: string;
  description?: string;
};

export type KnowledgeBaseInputErrors = Partial<Record<'name' | 'description', string>>;

type KnowledgeBaseListResponse = {
  list: KnowledgeBase[];
};

export function listKnowledgeBases(): Promise<KnowledgeBaseListResponse> {
  return authenticatedRequest<KnowledgeBaseListResponse>('/api/knowledge-base/');
}

export function createKnowledgeBase(input: CreateKnowledgeBaseInput): Promise<KnowledgeBase> {
  const name = input.name.trim();
  const description = input.description?.trim();
  return authenticatedRequest<KnowledgeBase>('/api/knowledge-base', {
    method: 'POST',
    body: JSON.stringify(description ? { name, description } : { name }),
  });
}

export function getKnowledgeBase(knowledgeBaseId: string): Promise<KnowledgeBase> {
  return authenticatedRequest<KnowledgeBase>(
    `/api/knowledge-base/${encodeURIComponent(knowledgeBaseId)}`,
  );
}

export function setDefaultKnowledgeBase(knowledgeBaseId: string): Promise<KnowledgeBase> {
  return authenticatedRequest<KnowledgeBase>(
    `/api/knowledge-base/${encodeURIComponent(knowledgeBaseId)}/default`,
    { method: 'POST' },
  );
}

export function validateKnowledgeBaseInput(input: CreateKnowledgeBaseInput): KnowledgeBaseInputErrors {
  const errors: KnowledgeBaseInputErrors = {};
  const name = input.name.trim();
  if (!name) {
    errors.name = '请输入知识库名称。';
  } else if (name.length > 128) {
    errors.name = '知识库名称不能超过 128 个字符。';
  }
  if ((input.description?.trim().length ?? 0) > 512) {
    errors.description = '描述不能超过 512 个字符。';
  }
  return errors;
}
