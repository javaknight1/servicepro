/**
 * A job's photos and files (T174; mockups 09 and 16): a thumbnail grid with
 * the time each was taken and its caption, tap to view, caption and delete.
 * The office variant sits on the job page; the phone variant on the
 * technician's visit page has a big "Add photo" that opens the camera.
 *
 * Technicians can change or delete only what they added; office staff
 * (jobs.view_all) anything. The API enforces both.
 */
import {
  useId,
  useRef,
  useState,
  type ChangeEvent,
  type RefObject,
} from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Camera, FileText, ImageIcon, Plus, Trash2 } from 'lucide-react';
import { Modal } from '@components/shared';
import {
  ATTACHMENT_ACCEPT,
  MAX_ATTACHMENT_BYTES,
  jobAttachmentService,
  jobAttachmentsQueryKey,
  type JobAttachment,
} from '@services/jobAttachmentService';
import { cn } from '@utils/cn';
import { getErrorMessage } from '@/utils/error';

export interface JobPhotosSectionProps {
  jobId: string;
  /** 'phone' for the technician's visit page (camera, big targets). */
  variant?: 'office' | 'phone';
  /** The signed-in user, to know which attachments are theirs. */
  currentUserId?: string;
  /** Office staff may change or delete anyone's attachments. */
  canManageAll: boolean;
  /** Lets a button elsewhere (the phone's bottom bar) open the picker. */
  inputRef?: RefObject<HTMLInputElement | null>;
  /** IANA zone for the times shown (the tenant's). */
  timeZone?: string;
}

/** "5:41p", as on the mockups' thumbnails. */
function shortTime(iso: string, timeZone?: string): string {
  const text = new Date(iso).toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    timeZone,
  });
  return text.replace(/\s?([AP])M$/i, (_, m: string) => m.toLowerCase());
}

function formatSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(bytes / 1024))} KB`;
}

interface Progress {
  index: number;
  count: number;
  percent: number;
}

function AttachmentViewer({
  attachment,
  jobId,
  canChange,
  timeZone,
  onClose,
}: {
  attachment: JobAttachment;
  jobId: string;
  canChange: boolean;
  timeZone?: string;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [caption, setCaption] = useState(attachment.caption ?? '');
  const [confirming, setConfirming] = useState(false);
  const captionId = useId();
  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: jobAttachmentsQueryKey(jobId) });

  const save = useMutation({
    mutationFn: () =>
      jobAttachmentService.updateCaption(jobId, attachment.id, caption),
    onSuccess: refresh,
  });
  const remove = useMutation({
    mutationFn: () => jobAttachmentService.remove(jobId, attachment.id),
    onSuccess: () => {
      refresh();
      onClose();
    },
  });
  const isPhoto = attachment.kind === 'photo';
  const label = attachment.caption || attachment.filename || 'Photo';
  const meta = [
    attachment.uploaded_by_name,
    new Date(attachment.created_at).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
      timeZone,
    }),
  ].filter(Boolean);

  return (
    <Modal isOpen onClose={onClose} title={label} size="xl">
      <div className="space-y-4">
        {isPhoto ? (
          <a href={attachment.url} target="_blank" rel="noopener noreferrer">
            <img
              src={attachment.url}
              alt={label}
              className="max-h-[70vh] w-full rounded-xl bg-neutral-100 object-contain"
            />
          </a>
        ) : (
          <a
            href={attachment.url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-3 rounded-xl border border-neutral-200 p-4 text-primary-700 hover:bg-primary-50"
          >
            <FileText className="h-6 w-6" aria-hidden="true" />
            <span>
              Open {attachment.filename || 'file'} (
              {formatSize(attachment.size_bytes)})
            </span>
          </a>
        )}
        <p className="text-sm text-neutral-500">{meta.join(' · ')}</p>

        {canChange && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              save.mutate();
            }}
            className="space-y-2"
          >
            <label
              htmlFor={captionId}
              className="block text-sm font-medium text-neutral-700"
            >
              Caption
            </label>
            <div className="flex gap-2">
              <input
                id={captionId}
                value={caption}
                maxLength={500}
                onChange={(e) => setCaption(e.target.value)}
                placeholder="e.g. Lobby floor"
                className="min-h-11 flex-1 rounded-xl border border-neutral-300 px-3 text-base focus:border-primary-500 focus:ring-2 focus:ring-primary-200 focus:outline-hidden"
              />
              <button
                type="submit"
                disabled={
                  save.isPending || caption === (attachment.caption ?? '')
                }
                className="min-h-11 rounded-xl border-[1.5px] border-primary-600 px-4 font-semibold text-primary-700 hover:bg-primary-50 disabled:opacity-50"
              >
                {save.isPending ? 'Saving…' : 'Save'}
              </button>
            </div>
            {save.isError && (
              <p role="alert" className="text-sm text-error-600">
                {getErrorMessage(save.error, "Couldn't save the caption")}
              </p>
            )}
          </form>
        )}

        {canChange && (
          <div className="flex items-center justify-end gap-2 border-t border-neutral-100 pt-3">
            {confirming && (
              <button
                type="button"
                onClick={() => setConfirming(false)}
                className="min-h-11 rounded-xl px-4 font-medium text-neutral-700 hover:bg-neutral-100"
              >
                Keep it
              </button>
            )}
            <button
              type="button"
              onClick={() =>
                confirming ? remove.mutate() : setConfirming(true)
              }
              disabled={remove.isPending}
              className="inline-flex min-h-11 items-center gap-2 rounded-xl border border-error-200 px-4 font-semibold text-error-700 hover:bg-error-50 disabled:opacity-50"
            >
              <Trash2 className="h-4 w-4" aria-hidden="true" />
              {remove.isPending
                ? 'Deleting…'
                : confirming
                  ? `Delete this ${isPhoto ? 'photo' : 'file'}`
                  : 'Delete'}
            </button>
          </div>
        )}
        {remove.isError && (
          <p role="alert" className="text-sm text-error-600">
            {getErrorMessage(remove.error, "Couldn't delete it")}
          </p>
        )}
      </div>
    </Modal>
  );
}

export function JobPhotosSection({
  jobId,
  variant = 'office',
  currentUserId,
  canManageAll,
  inputRef,
  timeZone,
}: JobPhotosSectionProps) {
  const phone = variant === 'phone';
  const queryClient = useQueryClient();
  const ownInput = useRef<HTMLInputElement>(null);
  const fileInput = inputRef ?? ownInput;
  const [progress, setProgress] = useState<Progress | null>(null);
  const [tooBig, setTooBig] = useState<string | null>(null);
  const [viewing, setViewing] = useState<JobAttachment | null>(null);

  const {
    data: attachments = [],
    isLoading,
    error,
  } = useQuery({
    queryKey: jobAttachmentsQueryKey(jobId),
    queryFn: () => jobAttachmentService.list(jobId),
    // Links are presigned for an hour: refresh well inside that.
    staleTime: 10 * 60 * 1000,
    refetchInterval: 30 * 60 * 1000,
  });

  const upload = useMutation({
    mutationFn: async (files: File[]) => {
      for (let i = 0; i < files.length; i++) {
        setProgress({ index: i + 1, count: files.length, percent: 0 });
        await jobAttachmentService.upload(jobId, files[i], undefined, (p) =>
          setProgress({ index: i + 1, count: files.length, percent: p })
        );
      }
    },
    onSettled: () => {
      setProgress(null);
      queryClient.invalidateQueries({
        queryKey: jobAttachmentsQueryKey(jobId),
      });
    },
  });

  const onPick = (e: ChangeEvent<HTMLInputElement>) => {
    const picked = Array.from(e.target.files ?? []);
    e.target.value = '';
    const ok = picked.filter((f) => f.size <= MAX_ATTACHMENT_BYTES);
    const skipped = picked.filter((f) => f.size > MAX_ATTACHMENT_BYTES);
    setTooBig(
      skipped.length > 0
        ? `${skipped.map((f) => f.name).join(', ')} ${skipped.length === 1 ? 'is' : 'are'} larger than 15 MB.`
        : null
    );
    if (ok.length > 0) upload.mutate(ok);
  };

  const canChange = (a: JobAttachment) =>
    canManageAll || (!!currentUserId && a.uploaded_by === currentUserId);
  const photos = attachments.filter((a) => a.kind === 'photo').length;
  const files = attachments.length - photos;
  const summary = [
    photos > 0 ? `${photos} ${photos === 1 ? 'photo' : 'photos'}` : null,
    files > 0 ? `${files} ${files === 1 ? 'file' : 'files'}` : null,
  ]
    .filter(Boolean)
    .join(' · ');
  const busy = upload.isPending;
  const addLabel = phone ? 'Add photo' : 'Add photos';

  return (
    <section
      aria-labelledby={`photos-${jobId}`}
      className={cn(
        'border border-neutral-200 bg-white',
        phone ? 'rounded-2xl p-4' : 'rounded-2xl p-6'
      )}
    >
      <div className="flex items-center justify-between gap-3">
        <h2
          id={`photos-${jobId}`}
          className="text-lg font-semibold text-neutral-900"
        >
          Photos
        </h2>
        <div className="flex items-center gap-3">
          {summary && (
            <span className="text-sm text-neutral-500">{summary}</span>
          )}
          {!phone && (
            <button
              type="button"
              onClick={() => fileInput.current?.click()}
              disabled={busy}
              className="inline-flex min-h-10 items-center gap-1.5 rounded-xl border-[1.5px] border-primary-600 px-3 text-sm font-semibold text-primary-700 hover:bg-primary-50 disabled:opacity-50"
            >
              <Plus className="h-4 w-4" aria-hidden="true" />
              {addLabel}
            </button>
          )}
        </div>
      </div>

      <input
        ref={fileInput}
        type="file"
        className="sr-only"
        tabIndex={-1}
        aria-label={addLabel}
        data-testid="job-photo-input"
        accept={phone ? 'image/*' : ATTACHMENT_ACCEPT}
        capture={phone ? 'environment' : undefined}
        multiple={!phone}
        onChange={onPick}
      />

      {progress && (
        <p role="status" className="mt-3 text-sm text-neutral-600">
          Uploading
          {progress.count > 1
            ? ` ${progress.index} of ${progress.count}`
            : ''}… {progress.percent}%
        </p>
      )}
      {(tooBig || upload.isError) && (
        <p role="alert" className="mt-3 text-sm text-error-600">
          {tooBig ?? getErrorMessage(upload.error, "Couldn't upload the photo")}
          {tooBig && upload.isError
            ? ` ${getErrorMessage(upload.error, "Couldn't upload the photo")}`
            : ''}
        </p>
      )}

      {isLoading ? (
        <p className="mt-3 text-neutral-500">Loading photos…</p>
      ) : error ? (
        <p role="alert" className="mt-3 text-sm text-error-600">
          {getErrorMessage(error, "Couldn't load the photos")}
        </p>
      ) : attachments.length === 0 ? (
        <p className="mt-3 text-neutral-500">
          {phone
            ? 'No photos yet. Take before and after shots as proof of service.'
            : 'No photos yet.'}
        </p>
      ) : (
        <ul
          aria-label="Job photos"
          className={cn(
            'mt-4 grid gap-3',
            phone ? 'grid-cols-3' : 'grid-cols-2 sm:grid-cols-4'
          )}
        >
          {attachments.map((a) => {
            const label = a.caption || a.filename || 'Photo';
            return (
              <li key={a.id} className="min-w-0">
                <button
                  type="button"
                  onClick={() => setViewing(a)}
                  aria-label={`View ${label}`}
                  className="group relative block aspect-[4/3] w-full overflow-hidden rounded-xl border border-neutral-200 bg-neutral-100 focus:ring-2 focus:ring-primary-300 focus:outline-hidden"
                >
                  {a.kind === 'photo' && a.thumb_url ? (
                    <img
                      src={a.thumb_url}
                      alt=""
                      loading="lazy"
                      className="h-full w-full object-cover transition group-hover:opacity-90"
                    />
                  ) : (
                    <span className="flex h-full w-full flex-col items-center justify-center gap-1 px-2 text-neutral-500">
                      {a.kind === 'photo' ? (
                        <ImageIcon className="h-6 w-6" aria-hidden="true" />
                      ) : (
                        <FileText className="h-6 w-6" aria-hidden="true" />
                      )}
                      <span className="w-full truncate text-xs">
                        {a.filename}
                      </span>
                    </span>
                  )}
                  <span className="absolute right-1.5 bottom-1.5 rounded-md bg-white/90 px-1.5 py-0.5 text-xs font-medium text-neutral-700">
                    {shortTime(a.created_at, timeZone)}
                  </span>
                </button>
                {a.caption && (
                  <p className="mt-1 truncate text-sm text-neutral-700">
                    {a.caption}
                  </p>
                )}
              </li>
            );
          })}
        </ul>
      )}

      {phone && !inputRef && (
        <button
          type="button"
          onClick={() => fileInput.current?.click()}
          disabled={busy}
          className="mt-4 flex min-h-14 w-full items-center justify-center gap-2 rounded-xl border-2 border-primary-600 bg-white text-lg font-semibold text-primary-600 hover:bg-primary-50 disabled:opacity-60"
        >
          <Camera className="h-6 w-6" aria-hidden="true" />
          {busy ? 'Uploading…' : addLabel}
        </button>
      )}

      {viewing && (
        <AttachmentViewer
          key={viewing.id}
          attachment={viewing}
          jobId={jobId}
          canChange={canChange(viewing)}
          timeZone={timeZone}
          onClose={() => setViewing(null)}
        />
      )}
    </section>
  );
}
