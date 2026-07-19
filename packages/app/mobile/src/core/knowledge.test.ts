import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  createKnowledgeBase,
  getKnowledgeBase,
  listKnowledgeBases,
  setDefaultKnowledgeBase,
  validateKnowledgeBaseInput,
} from './knowledge';

const api = vi.hoisted(() => ({ authenticatedRequest: vi.fn() }));

vi.mock('./api', () => ({ authenticatedRequest: api.authenticatedRequest }));

describe('knowledge base API', () => {
  beforeEach(() => {
    api.authenticatedRequest.mockReset();
  });

  it('loads the authenticated knowledge base list from the documented endpoint', async () => {
    const response = {
      list: [
        {
          id: 'knowledge-1',
          name: '产品资料',
          description: 'Cove 产品文档',
          doc_count: 12,
          chat_enabled: true,
        },
      ],
    };
    api.authenticatedRequest.mockResolvedValue(response);

    await expect(listKnowledgeBases()).resolves.toBe(response);
    expect(api.authenticatedRequest).toHaveBeenCalledWith('/api/knowledge-base/');
  });

  it('sets the selected knowledge base as default through the documented endpoint', async () => {
    const response = {
      id: 'knowledge-2',
      name: '团队资料',
      is_default: true,
    };
    api.authenticatedRequest.mockResolvedValue(response);

    await expect(setDefaultKnowledgeBase('knowledge-2')).resolves.toBe(response);
    expect(api.authenticatedRequest).toHaveBeenCalledWith(
      '/api/knowledge-base/knowledge-2/default',
      { method: 'POST' },
    );
  });

  it('creates a knowledge base with trimmed documented fields through authenticated request', async () => {
    const response = { id: 'knowledge-3', name: '产品资料', description: '说明' };
    api.authenticatedRequest.mockResolvedValue(response);

    await expect(createKnowledgeBase({ name: '  产品资料  ', description: '  说明  ' })).resolves.toBe(response);
    expect(api.authenticatedRequest).toHaveBeenCalledWith('/api/knowledge-base', {
      method: 'POST',
      body: JSON.stringify({ name: '产品资料', description: '说明' }),
    });
  });

  it('omits an empty optional description when creating a knowledge base', async () => {
    api.authenticatedRequest.mockResolvedValue({ id: 'knowledge-4', name: '空描述' });

    await createKnowledgeBase({ name: '空描述', description: '   ' });
    expect(api.authenticatedRequest).toHaveBeenCalledWith('/api/knowledge-base', {
      method: 'POST',
      body: JSON.stringify({ name: '空描述' }),
    });
  });

  it('loads an encoded knowledge base detail through authenticated request', async () => {
    const response = { id: 'knowledge / 5', name: '团队资料' };
    api.authenticatedRequest.mockResolvedValue(response);

    await expect(getKnowledgeBase('knowledge / 5')).resolves.toBe(response);
    expect(api.authenticatedRequest).toHaveBeenCalledWith('/api/knowledge-base/knowledge%20%2F%205');
  });

  it('validates required name and documented field limits', () => {
    expect(validateKnowledgeBaseInput({ name: '   ', description: 'a'.repeat(513) })).toEqual({
      name: '请输入知识库名称。',
      description: '描述不能超过 512 个字符。',
    });
    expect(validateKnowledgeBaseInput({ name: 'a'.repeat(129) })).toEqual({
      name: '知识库名称不能超过 128 个字符。',
    });
    expect(validateKnowledgeBaseInput({ name: '有效名称', description: '说明' })).toEqual({});
  });
});
