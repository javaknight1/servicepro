/**
 * Full-screen signature capture for the customer's sign-off (T176, mockup
 * 16): signer name (title and email optional), a canvas drawn with pointer
 * events (finger, stylus or mouse), Clear and Submit. The signature leaves
 * as a PNG data URL; the API sniffs and checks it again.
 */
import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type FormEvent,
  type PointerEvent as ReactPointerEvent,
} from 'react';
import { Eraser, X } from 'lucide-react';
import { cn } from '@utils/cn';

export interface SignatureSubmission {
  signer_name: string;
  signer_title: string;
  signer_email: string;
  /** PNG data URL. */
  signature: string;
}

export interface SignaturePadProps {
  /** Shown under the title, e.g. the site's name. */
  subtitle?: string;
  /** Prefills the signer (e.g. the site contact). */
  defaultSignerName?: string;
  pending?: boolean;
  error?: string | null;
  onSubmit: (value: SignatureSubmission) => void;
  onCancel: () => void;
}

/** Stroke length (CSS px) that counts as a signature, not a stray tap. */
export const MIN_SIGNATURE_LENGTH = 40;
/** Cap the backing store: sharp enough, and well under the API's limits. */
const MAX_PIXEL_RATIO = 2;

const inputClass =
  'mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-3 text-base text-neutral-900 placeholder:text-neutral-400 focus:border-primary-500 focus:ring-2 focus:ring-primary-200 focus:outline-hidden';

interface Point {
  x: number;
  y: number;
}

export function SignaturePad({
  subtitle,
  defaultSignerName = '',
  pending = false,
  error,
  onSubmit,
  onCancel,
}: SignaturePadProps) {
  const titleId = useId();
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const last = useRef<Point | null>(null);
  const drawing = useRef(false);
  const size = useRef('');
  const [length, setLength] = useState(0);
  const [name, setName] = useState(defaultSignerName);
  const [title, setTitle] = useState('');
  const [email, setEmail] = useState('');

  const context = () => canvasRef.current?.getContext('2d') ?? null;

  // Size the backing store to the box (and the screen's pixel ratio); that
  // clears it, so the drawn length resets too. The box has a fixed height,
  // so the phone keyboard opening (a resize) doesn't wipe the signature;
  // only a real size change (rotation) does.
  const fit = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();
    const key = `${rect.width}x${rect.height}`;
    if (key === size.current) return;
    size.current = key;
    const ratio = Math.min(window.devicePixelRatio || 1, MAX_PIXEL_RATIO);
    canvas.width = Math.max(1, Math.round(rect.width * ratio));
    canvas.height = Math.max(1, Math.round(rect.height * ratio));
    const ctx = context();
    if (ctx) {
      ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
      ctx.lineCap = 'round';
      ctx.lineJoin = 'round';
      ctx.lineWidth = 2.5;
      ctx.strokeStyle = '#171717';
    }
    setLength(0);
  }, []);

  useEffect(() => {
    fit();
    window.addEventListener('resize', fit);
    // Nothing behind the sheet scrolls while signing.
    const overflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      window.removeEventListener('resize', fit);
      document.body.style.overflow = overflow;
    };
  }, [fit]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !pending) onCancel();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onCancel, pending]);

  const point = (e: ReactPointerEvent<HTMLCanvasElement>): Point => {
    const rect = e.currentTarget.getBoundingClientRect();
    return { x: e.clientX - rect.left, y: e.clientY - rect.top };
  };

  const onPointerDown = (e: ReactPointerEvent<HTMLCanvasElement>) => {
    if (pending) return;
    e.preventDefault();
    e.currentTarget.setPointerCapture?.(e.pointerId);
    drawing.current = true;
    const p = point(e);
    last.current = p;
    const ctx = context();
    if (ctx) {
      ctx.beginPath();
      ctx.arc(p.x, p.y, 1.2, 0, Math.PI * 2);
      ctx.fillStyle = '#171717';
      ctx.fill();
    }
  };

  const onPointerMove = (e: ReactPointerEvent<HTMLCanvasElement>) => {
    if (!drawing.current || !last.current) return;
    e.preventDefault();
    const p = point(e);
    const from = last.current;
    const ctx = context();
    if (ctx) {
      ctx.beginPath();
      ctx.moveTo(from.x, from.y);
      ctx.lineTo(p.x, p.y);
      ctx.stroke();
    }
    setLength((n) => n + Math.hypot(p.x - from.x, p.y - from.y));
    last.current = p;
  };

  const endStroke = () => {
    drawing.current = false;
    last.current = null;
  };

  const clear = () => {
    const canvas = canvasRef.current;
    const ctx = context();
    if (canvas && ctx) {
      ctx.save();
      ctx.setTransform(1, 0, 0, 1, 0, 0);
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      ctx.restore();
    }
    setLength(0);
  };

  const signed = length >= MIN_SIGNATURE_LENGTH;
  const ready = signed && name.trim() !== '' && !pending;

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const canvas = canvasRef.current;
    if (!ready || !canvas) return;
    onSubmit({
      signer_name: name.trim(),
      signer_title: title.trim(),
      signer_email: email.trim(),
      signature: canvas.toDataURL('image/png'),
    });
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      className="fixed inset-0 z-50 flex flex-col bg-white"
    >
      <form
        onSubmit={submit}
        className="mx-auto flex h-full w-full max-w-xl flex-col overflow-y-auto px-4 pt-4 pb-4"
      >
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2
              id={titleId}
              className="text-2xl font-bold tracking-tight text-neutral-900"
            >
              Customer sign-off
            </h2>
            {subtitle && <p className="mt-0.5 text-neutral-600">{subtitle}</p>}
          </div>
          <button
            type="button"
            onClick={onCancel}
            disabled={pending}
            aria-label="Cancel signature"
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-neutral-600 hover:bg-neutral-100 disabled:opacity-50"
          >
            <X className="h-6 w-6" aria-hidden="true" />
          </button>
        </div>

        <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
          <label className="block sm:col-span-2">
            <span className="text-sm font-medium text-neutral-700">
              Signer&apos;s name
            </span>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={100}
              autoComplete="off"
              placeholder="e.g. Ray Burke"
              className={inputClass}
            />
          </label>
          <label className="block">
            <span className="text-sm font-medium text-neutral-700">
              Title (optional)
            </span>
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              maxLength={100}
              placeholder="e.g. Site manager"
              className={inputClass}
            />
          </label>
          <label className="block">
            <span className="text-sm font-medium text-neutral-700">
              Email for the report (optional)
            </span>
            <input
              type="email"
              inputMode="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              maxLength={255}
              autoComplete="off"
              className={inputClass}
            />
          </label>
        </div>

        <div className="relative mt-3 h-56 shrink-0 rounded-2xl border-2 border-dashed border-neutral-300 bg-neutral-50">
          <canvas
            ref={canvasRef}
            aria-label="Signature area: sign with your finger"
            data-testid="signature-canvas"
            className="absolute inset-0 h-full w-full cursor-crosshair touch-none rounded-2xl"
            onPointerDown={onPointerDown}
            onPointerMove={onPointerMove}
            onPointerUp={endStroke}
            onPointerCancel={endStroke}
            onPointerLeave={endStroke}
          />
          <div
            aria-hidden="true"
            className="pointer-events-none absolute right-6 bottom-10 left-6 flex items-end gap-2 border-b border-neutral-400 pb-1 text-neutral-400"
          >
            <span className="text-xl leading-none">×</span>
            {!signed && <span className="text-sm">Sign here</span>}
          </div>
        </div>

        {error && (
          <p role="alert" className="mt-2 text-sm text-error-600">
            {error}
          </p>
        )}

        <div className="mt-3 flex gap-3">
          <button
            type="button"
            onClick={clear}
            disabled={pending || length === 0}
            className="flex min-h-14 flex-1 items-center justify-center gap-2 rounded-xl border border-neutral-300 bg-white text-lg font-semibold text-neutral-800 hover:bg-neutral-50 disabled:opacity-50"
          >
            <Eraser className="h-5 w-5" aria-hidden="true" />
            Clear
          </button>
          <button
            type="submit"
            disabled={!ready}
            className={cn(
              'flex min-h-14 flex-[2] items-center justify-center rounded-xl text-lg font-semibold text-white',
              ready
                ? 'bg-primary-600 hover:bg-primary-700 active:bg-primary-800'
                : 'bg-primary-300'
            )}
          >
            {pending ? 'Saving…' : 'Submit signature'}
          </button>
        </div>
        <p className="mt-2 text-center text-sm text-neutral-500">
          {name.trim() === ''
            ? "Enter the signer's name, then sign above."
            : signed
              ? 'The time and this device are recorded with the signature.'
              : 'Sign above with your finger.'}
        </p>
      </form>
    </div>
  );
}
