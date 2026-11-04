import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError, type AxiosResponse } from 'axios';

vi.mock('@services/jobAttachmentService', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@services/jobAttachmentService')>()),
  jobAttachmentService: {
    list: vi.fn(),
    upload: vi.fn(),
    updateCaption: vi.fn(),
    remove: vi.fn(),
  },
}));

import {
  jobAttachmentService,
  type JobAttachment,
} from '@services/jobAttachmentService';
import {
  JobPhotosSection,
  type JobPhotosSectionProps,
} from '../JobPhotosSection';

const attachment = (
  id: string,
  over: Partial<JobAttachment> = {}
): JobAttachment => ({
  id,
  job_id: 'job-1',
  kind: 'photo',
  filename: `${id}.jpg`,
  content_type: 'image/jpeg',
  size_bytes: 120_000,
  width: 2048,
  height: 1536,
  uploaded_by: 'tech-1',
  uploaded_by_name: 'Tasha Green',
  url: `https://store.test/${id}.jpg?sig=1`,
  thumb_url: `https://store.test/${id}_thumb.jpg?sig=1`,
  file_path: `/api/v1/jobs/job-1/attachments/${id}/file`,
  thumb_path: `/api/v1/jobs/job-1/attachments/${id}/thumb`,
  // 5:41 PM in Columbus
  created_at: '2026-11-02T22:41:00Z',
  ...over,
});

const renderSection = (props: Partial<JobPhotosSectionProps> = {}) =>
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <JobPhotosSection
        jobId="job-1"
        canManageAll={false}
        currentUserId="tech-1"
        timeZone="America/New_York"
        {...props}
      />
    </QueryClientProvider>
  );

describe('JobPhotosSection (T174)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows thumbnails with the time taken and captions', async () => {
    vi.mocked(jobAttachmentService.list).mockResolvedValue([
      attachment('p1', { caption: 'Lobby floor' }),
      attachment('p2'),
      attachment('f1', {
        kind: 'file',
        filename: 'work-order.pdf',
        content_type: 'application/pdf',
        thumb_url: undefined,
        thumb_path: undefined,
        width: undefined,
        height: undefined,
      }),
    ]);
    renderSection({ canManageAll: true });

    const grid = await screen.findByRole('list', { name: 'Job photos' });
    const tiles = within(grid).getAllByRole('listitem');
    expect(tiles).toHaveLength(3);
    expect(within(tiles[0]).getByText('Lobby floor')).toBeInTheDocument();
    expect(within(tiles[0]).getByText('5:41p')).toBeInTheDocument();
    expect(tiles[0].querySelector('img')).toHaveAttribute(
      'src',
      'https://store.test/p1_thumb.jpg?sig=1'
    );
    expect(within(tiles[2]).getByText('work-order.pdf')).toBeInTheDocument();
    expect(tiles[2].querySelector('img')).toBeNull();
    expect(screen.getByText('2 photos · 1 file')).toBeInTheDocument();
    // The office picker takes several photos or PDFs, no forced camera.
    const input = screen.getByTestId('job-photo-input');
    expect(input).toHaveAttribute('multiple');
    expect(input).not.toHaveAttribute('capture');
    expect(input.getAttribute('accept')).toContain('application/pdf');
  });

  it('says when there are no photos', async () => {
    vi.mocked(jobAttachmentService.list).mockResolvedValue([]);
    renderSection({ variant: 'phone' });
    expect(
      await screen.findByText(/No photos yet\. Take before and after shots/)
    ).toBeInTheDocument();
    // The phone variant opens the camera.
    const input = screen.getByTestId('job-photo-input');
    expect(input).toHaveAttribute('capture', 'environment');
    expect(input).toHaveAttribute('accept', 'image/*');
    expect(
      screen.getByRole('button', { name: 'Add photo' })
    ).toBeInTheDocument();
  });

  it('uploads each picked file and skips ones over 15 MB', async () => {
    vi.mocked(jobAttachmentService.list).mockResolvedValue([]);
    vi.mocked(jobAttachmentService.upload).mockResolvedValue(attachment('p9'));
    renderSection({ canManageAll: true });
    await screen.findByText('No photos yet.');

    const small = new File(['a'], 'small.jpg', { type: 'image/jpeg' });
    const huge = new File(['b'], 'huge.jpg', { type: 'image/jpeg' });
    Object.defineProperty(huge, 'size', { value: 16 * 1024 * 1024 });
    const user = userEvent.setup();
    await user.upload(
      screen.getByTestId('job-photo-input') as HTMLInputElement,
      [small, huge]
    );

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'huge.jpg is larger than 15 MB.'
    );
    await waitFor(() =>
      expect(jobAttachmentService.upload).toHaveBeenCalledTimes(1)
    );
    expect(jobAttachmentService.upload).toHaveBeenCalledWith(
      'job-1',
      small,
      undefined,
      expect.any(Function)
    );
    // The list is reloaded afterwards.
    await waitFor(() =>
      expect(jobAttachmentService.list).toHaveBeenCalledTimes(2)
    );
  });

  it('shows the server error when an upload is refused', async () => {
    vi.mocked(jobAttachmentService.list).mockResolvedValue([]);
    vi.mocked(jobAttachmentService.upload).mockRejectedValue(
      new AxiosError('Unsupported', '415', undefined, undefined, {
        status: 415,
        data: {
          error: 'unsupported_file_type',
          message:
            'Only JPEG, PNG and WebP photos and PDF files can be attached',
        },
      } as AxiosResponse)
    );
    renderSection({ variant: 'phone' });
    await screen.findByText(/No photos yet/);
    const user = userEvent.setup();
    await user.upload(
      screen.getByTestId('job-photo-input') as HTMLInputElement,
      new File(['x'], 'x.jpg', { type: 'image/jpeg' })
    );
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Only JPEG, PNG and WebP photos'
    );
  });

  it('lets a technician caption and delete their own photo', async () => {
    vi.mocked(jobAttachmentService.list).mockResolvedValue([
      attachment('p1', { caption: 'Lobby floor' }),
    ]);
    vi.mocked(jobAttachmentService.updateCaption).mockResolvedValue(
      attachment('p1', { caption: 'Restroom 2F' })
    );
    vi.mocked(jobAttachmentService.remove).mockResolvedValue(undefined);
    renderSection();
    const user = userEvent.setup();
    await user.click(
      await screen.findByRole('button', { name: 'View Lobby floor' })
    );

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByRole('img')).toHaveAttribute(
      'src',
      'https://store.test/p1.jpg?sig=1'
    );
    expect(within(dialog).getByText(/Tasha Green/)).toBeInTheDocument();

    const caption = within(dialog).getByLabelText('Caption');
    await user.clear(caption);
    await user.type(caption, 'Restroom 2F');
    await user.click(within(dialog).getByRole('button', { name: 'Save' }));
    expect(jobAttachmentService.updateCaption).toHaveBeenCalledWith(
      'job-1',
      'p1',
      'Restroom 2F'
    );

    await user.click(within(dialog).getByRole('button', { name: 'Delete' }));
    expect(jobAttachmentService.remove).not.toHaveBeenCalled();
    await user.click(
      within(dialog).getByRole('button', { name: 'Delete this photo' })
    );
    await waitFor(() =>
      expect(jobAttachmentService.remove).toHaveBeenCalledWith('job-1', 'p1')
    );
  });

  it("doesn't offer to change someone else's photo to a technician", async () => {
    vi.mocked(jobAttachmentService.list).mockResolvedValue([
      attachment('p1', { caption: 'Lobby floor', uploaded_by: 'office-1' }),
    ]);
    renderSection();
    const user = userEvent.setup();
    await user.click(
      await screen.findByRole('button', { name: 'View Lobby floor' })
    );
    const dialog = await screen.findByRole('dialog');
    expect(
      within(dialog).queryByRole('button', { name: 'Delete' })
    ).not.toBeInTheDocument();
    expect(within(dialog).queryByLabelText('Caption')).not.toBeInTheDocument();
  });
});
