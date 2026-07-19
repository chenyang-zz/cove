import { authenticatedRequest } from './api';

export type KnowledgeDocument = {
  id: string;
  kb_id?: string | null;
  file_name: string;
  file_ext?: string | null;
  file_size?: number | null;
  source_type?: string | null;
  source_url?: string | null;
  status?: string | null;
  progress?: number | null;
  chunk_num?: number | null;
  error_msg?: string | null;
  tags?: string[] | null;
  created_at?: string | null;
  updated_at?: string | null;
};

export type DocumentListResponse = {
  list: KnowledgeDocument[];
  total: number;
  page: number;
  page_size: number;
};

export type DocumentPaginationState = {
  items: KnowledgeDocument[];
  total: number;
  page: number;
  pageSize: number;
};

export function listDocuments(
  knowledgeBaseId: string,
  page = 1,
  pageSize = 20,
): Promise<DocumentListResponse> {
  const query = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
    kb_id: knowledgeBaseId,
  });
  return authenticatedRequest<DocumentListResponse>(`/api/document?${query}`);
}

export function mergeDocumentPage(
  current: DocumentPaginationState,
  response: DocumentListResponse,
  reset = false,
): DocumentPaginationState {
  const merged = reset ? [] : [...current.items];
  const known = new Set(merged.map((item) => item.id));
  for (const item of response.list) {
    if (!item.id || known.has(item.id)) {
      continue;
    }
    known.add(item.id);
    merged.push(item);
  }
  return {
    items: merged,
    total: Math.max(0, response.total),
    page: Math.max(1, response.page),
    pageSize: Math.max(1, response.page_size),
  };
}

export function canLoadMoreDocuments(
  state: DocumentPaginationState,
  busy: boolean,
): boolean {
  return !busy && state.items.length > 0 && state.items.length < state.total;
}
