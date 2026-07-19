import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  canLoadMoreDocuments,
  listDocuments,
  mergeDocumentPage,
  type DocumentPaginationState,
} from './documents';

const api = vi.hoisted(() => ({ authenticatedRequest: vi.fn() }));

vi.mock('./api', () => ({ authenticatedRequest: api.authenticatedRequest }));

const firstPage: DocumentPaginationState = {
  items: [{ id: 'document-1', file_name: 'one.md' }],
  total: 3,
  page: 1,
  pageSize: 20,
};

describe('knowledge document API and pagination', () => {
  beforeEach(() => {
    api.authenticatedRequest.mockReset();
  });

  it('maps the documented page, page_size and encoded kb_id query through authenticated request', async () => {
    const response = { list: [], total: 0, page: 2, page_size: 20 };
    api.authenticatedRequest.mockResolvedValue(response);

    await expect(listDocuments('knowledge / 1', 2, 20)).resolves.toBe(response);
    expect(api.authenticatedRequest).toHaveBeenCalledWith(
      '/api/document?page=2&page_size=20&kb_id=knowledge+%2F+1',
    );
  });

  it('merges subsequent pages without duplicate document ids and preserves order', () => {
    expect(mergeDocumentPage(firstPage, {
      list: [
        { id: 'document-1', file_name: 'duplicate.md' },
        { id: 'document-2', file_name: 'two.md' },
      ],
      total: 3,
      page: 2,
      page_size: 20,
    })).toEqual({
      items: [
        { id: 'document-1', file_name: 'one.md' },
        { id: 'document-2', file_name: 'two.md' },
      ],
      total: 3,
      page: 2,
      pageSize: 20,
    });
  });

  it('resets pagination to the refreshed first page', () => {
    expect(mergeDocumentPage(firstPage, {
      list: [{ id: 'document-new', file_name: 'new.md' }],
      total: 1,
      page: 1,
      page_size: 20,
    }, true)).toEqual({
      items: [{ id: 'document-new', file_name: 'new.md' }],
      total: 1,
      page: 1,
      pageSize: 20,
    });
  });

  it('guards load more while busy, empty or fully loaded', () => {
    expect(canLoadMoreDocuments(firstPage, false)).toBe(true);
    expect(canLoadMoreDocuments(firstPage, true)).toBe(false);
    expect(canLoadMoreDocuments({ ...firstPage, items: [] }, false)).toBe(false);
    expect(canLoadMoreDocuments({ ...firstPage, total: 1 }, false)).toBe(false);
  });
});
