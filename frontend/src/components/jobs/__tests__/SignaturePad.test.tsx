import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SignaturePad } from '../SignaturePad';

/** The 2D context calls the pad makes; jsdom has no canvas. */
interface FakeContext {
  setTransform: ReturnType<typeof vi.fn>;
  beginPath: ReturnType<typeof vi.fn>;
  arc: ReturnType<typeof vi.fn>;
  fill: ReturnType<typeof vi.fn>;
  moveTo: ReturnType<typeof vi.fn>;
  lineTo: ReturnType<typeof vi.fn>;
  stroke: ReturnType<typeof vi.fn>;
  save: ReturnType<typeof vi.fn>;
  restore: ReturnType<typeof vi.fn>;
  clearRect: ReturnType<typeof vi.fn>;
}

const makeContext = (): FakeContext => ({
  setTransform: vi.fn(),
  beginPath: vi.fn(),
  arc: vi.fn(),
  fill: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  stroke: vi.fn(),
  save: vi.fn(),
  restore: vi.fn(),
  clearRect: vi.fn(),
});

let ctx: FakeContext;

beforeEach(() => {
  ctx = makeContext();
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
    ctx as unknown as CanvasRenderingContext2D
  );
  vi.spyOn(HTMLCanvasElement.prototype, 'toDataURL').mockReturnValue(
    'data:image/png;base64,SIG'
  );
});

afterEach(() => {
  vi.restoreAllMocks();
});

/** Draws a horizontal stroke of `length` CSS px across the canvas. */
function draw(canvas: HTMLElement, length: number) {
  fireEvent.pointerDown(canvas, { clientX: 10, clientY: 50, pointerId: 1 });
  for (let x = 20; x <= 10 + length; x += 10) {
    fireEvent.pointerMove(canvas, { clientX: x, clientY: 50, pointerId: 1 });
  }
  fireEvent.pointerUp(canvas, { pointerId: 1 });
}

describe('SignaturePad (T176)', () => {
  it('needs a name and a real signature before it submits', async () => {
    const onSubmit = vi.fn();
    render(
      <SignaturePad
        subtitle="Elm Street Plaza"
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    );
    const user = userEvent.setup();
    expect(
      screen.getByRole('dialog', { name: 'Customer sign-off' })
    ).toBeInTheDocument();
    expect(screen.getByText('Elm Street Plaza')).toBeInTheDocument();
    const submit = screen.getByRole('button', { name: 'Submit signature' });
    expect(submit).toBeDisabled();

    const canvas = screen.getByTestId('signature-canvas');
    expect(canvas).toHaveClass('touch-none');
    // A tap is not a signature.
    draw(canvas, 10);
    await user.type(screen.getByLabelText("Signer's name"), '  Ray Burke ');
    expect(submit).toBeDisabled();

    draw(canvas, 120);
    expect(ctx.lineTo).toHaveBeenCalled();
    expect(submit).toBeEnabled();

    await user.type(screen.getByLabelText('Title (optional)'), 'Site manager');
    await user.type(
      screen.getByLabelText('Email for the report (optional)'),
      'ray@example.com'
    );
    await user.click(submit);
    expect(onSubmit).toHaveBeenCalledWith({
      signer_name: 'Ray Burke',
      signer_title: 'Site manager',
      signer_email: 'ray@example.com',
      signature: 'data:image/png;base64,SIG',
    });
  });

  it('clears the canvas', async () => {
    render(
      <SignaturePad
        defaultSignerName="Ray Burke"
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    const user = userEvent.setup();
    const canvas = screen.getByTestId('signature-canvas');
    const clear = screen.getByRole('button', { name: 'Clear' });
    expect(clear).toBeDisabled();
    draw(canvas, 120);
    expect(
      screen.getByRole('button', { name: 'Submit signature' })
    ).toBeEnabled();

    await user.click(clear);
    expect(ctx.clearRect).toHaveBeenCalled();
    expect(
      screen.getByRole('button', { name: 'Submit signature' })
    ).toBeDisabled();
    expect(screen.getByText('Sign here')).toBeInTheDocument();
  });

  it('cancels with the close button and Escape, shows errors, locks while saving', async () => {
    const onCancel = vi.fn();
    const { rerender } = render(
      <SignaturePad
        onSubmit={vi.fn()}
        onCancel={onCancel}
        error="Ask the customer to sign before submitting"
      />
    );
    const user = userEvent.setup();
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Ask the customer to sign before submitting'
    );
    await user.click(screen.getByRole('button', { name: 'Cancel signature' }));
    await user.keyboard('{Escape}');
    expect(onCancel).toHaveBeenCalledTimes(2);

    rerender(<SignaturePad onSubmit={vi.fn()} onCancel={onCancel} pending />);
    expect(screen.getByRole('button', { name: 'Saving…' })).toBeDisabled();
    expect(
      screen.getByRole('button', { name: 'Cancel signature' })
    ).toBeDisabled();
  });
});
