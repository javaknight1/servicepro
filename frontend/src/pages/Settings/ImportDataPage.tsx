import { useEffect, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router';
import {
  AlertTriangle,
  CheckCircle2,
  FileSpreadsheet,
  Upload,
} from 'lucide-react';
import { DashboardLayout } from '@components/layout';
import { Button, PageHeader } from '@components/shared';
import {
  IMPORT_ACCEPT,
  IMPORT_ENTITIES,
  MAX_IMPORT_BYTES,
  importService,
  isImportFinished,
} from '@services/importService';
import type {
  ImportEntity,
  ImportJob,
  ImportPreview,
  ImportSource,
} from '@services/importService';
import { getErrorMessage } from '@/utils/error';
import { cn } from '@utils/cn';

const SOURCES: { key: ImportSource; label: string; icon: string | null }[] = [
  { key: 'jobber', label: 'Jobber', icon: 'J' },
  { key: 'housecall_pro', label: 'Housecall Pro', icon: 'H' },
  { key: 'generic', label: 'CSV or Excel', icon: null },
];

const ENTITY_LABELS: Record<ImportEntity, string> = {
  customers: 'Customers',
  sites: 'Sites',
  jobs: 'Jobs',
  invoices: 'Open invoices',
};

/** Which export to upload, per source and entity. */
const HELP: Record<ImportSource, Record<ImportEntity, string>> = {
  jobber: {
    customers:
      'Export your client list from Jobber as CSV (Clients → Export). Each client’s service property becomes its first site. Jobber splits big lists into several files: import each one.',
    sites:
      'Export your properties report from Jobber as CSV. Properties are matched to clients you already imported by client email or name.',
    jobs: 'Export your jobs report from Jobber as CSV. Jobs keep their Jobber job number.',
    invoices:
      'Export your invoices report from Jobber as CSV. Only invoices with a balance due are imported, for that balance.',
  },
  housecall_pro: {
    customers:
      'In Housecall Pro: Customers → More → Download customer list. The first address of each customer becomes its first site.',
    sites:
      'Housecall Pro addresses come in with the customer list; use this for any extra service addresses you keep in a spreadsheet.',
    jobs: 'Upload the jobs data report (date range: from when you started with Housecall Pro until today).',
    invoices:
      'Upload the same jobs data report: jobs whose invoice was sent and still has an amount due become open invoices.',
  },
  generic: {
    customers:
      'Any CSV or Excel file with a header row. Match its columns to ServicePro’s fields below.',
    sites: 'One row per site, naming its customer by email or name.',
    jobs: 'One row per job, naming its customer by email or name.',
    invoices:
      'One row per unpaid invoice: number, customer, balance due and dates.',
  },
};

/** Columns shown in the preview table, per entity. */
const PREVIEW_FIELDS: Record<ImportEntity, string[]> = {
  customers: [
    'company_name',
    'first_name',
    'last_name',
    'email',
    'phone',
    'billing_city',
  ],
  sites: ['customer_name', 'customer_email', 'name', 'street', 'city'],
  jobs: ['external_id', 'customer_name', 'title', 'status', 'scheduled_start'],
  invoices: [
    'invoice_number',
    'customer_name',
    'issue_date',
    'due_date',
    'balance',
  ],
};

const RESULT_LINKS: Record<ImportEntity, { to: string; label: string }> = {
  customers: { to: '/customers', label: 'View customers' },
  sites: { to: '/customers', label: 'View customers' },
  jobs: { to: '/jobs', label: 'View jobs' },
  invoices: { to: '/invoices', label: 'View invoices' },
};

function isSource(v: string | null): v is ImportSource {
  return v === 'jobber' || v === 'housecall_pro' || v === 'generic';
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

/**
 * Import data (T193): bring customers, sites, jobs and open invoices over
 * from Jobber, Housecall Pro or any spreadsheet. Upload → preview the
 * mapped rows and their problems (fix the column mapping if needed) →
 * import; the import runs in chunks the page keeps asking for until done.
 */
export function ImportDataPage() {
  const [params, setParams] = useSearchParams();
  const sourceParam = params.get('source');
  const source: ImportSource = isSource(sourceParam) ? sourceParam : 'jobber';
  const [entity, setEntity] = useState<ImportEntity>('customers');
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [busy, setBusy] = useState<'preview' | 'import' | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [job, setJob] = useState<ImportJob | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const alive = useRef(true);

  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);

  const reset = () => {
    setFile(null);
    setPreview(null);
    setJob(null);
    setError(null);
    if (inputRef.current) inputRef.current.value = '';
  };

  const chooseSource = (s: ImportSource) => {
    reset();
    setParams({ source: s }, { replace: true });
  };

  const chooseEntity = (e: ImportEntity) => {
    reset();
    setEntity(e);
  };

  const runPreview = async (f: File, mapping?: Record<string, string>) => {
    setBusy('preview');
    setError(null);
    try {
      const p = await importService.preview(entity, f, source, mapping);
      if (alive.current) setPreview(p);
    } catch (err: unknown) {
      if (alive.current) {
        setPreview(null);
        setError(getErrorMessage(err, 'The file could not be read'));
      }
    } finally {
      if (alive.current) setBusy(null);
    }
  };

  const onFile = (f: File | undefined) => {
    setPreview(null);
    setJob(null);
    if (!f) return;
    if (f.size > MAX_IMPORT_BYTES) {
      setFile(null);
      setError('The file is larger than 10 MB. Split it and import each part.');
      return;
    }
    setFile(f);
    void runPreview(f);
  };

  const changeMapping = (field: string, header: string) => {
    if (!file || !preview) return;
    const next = { ...preview.mapping };
    if (header) next[field] = header;
    else delete next[field];
    void runPreview(file, next);
  };

  const startImport = async () => {
    if (!file || !preview) return;
    setBusy('import');
    setError(null);
    try {
      let current = await importService.start(
        entity,
        file,
        source,
        preview.mapping
      );
      if (alive.current) setJob(current);
      // Keep the import moving while the page is open; the scheduled job
      // finishes it otherwise.
      while (alive.current && !isImportFinished(current)) {
        await sleep(current.status === 'pending' ? 300 : 2000);
        if (!alive.current) break;
        current =
          current.status === 'pending'
            ? await importService.run(entity, current.job_id)
            : await importService.get(entity, current.job_id);
        if (alive.current) setJob(current);
      }
    } catch (err: unknown) {
      if (alive.current)
        setError(getErrorMessage(err, 'The import could not be started'));
    } finally {
      if (alive.current) setBusy(null);
    }
  };

  const fields = preview?.fields ?? [];
  const shownFields = PREVIEW_FIELDS[entity].filter((k) =>
    preview?.rows.some((r) => r.values[k])
  );
  const label = (key: string) =>
    fields.find((f) => f.key === key)?.label ?? key;
  const canImport =
    !!preview &&
    preview.missing.length === 0 &&
    preview.valid_rows > 0 &&
    !busy;
  const showMapping =
    !!preview && (source === 'generic' || preview.missing.length > 0);

  return (
    <DashboardLayout>
      <div className="mx-auto max-w-5xl">
        <PageHeader
          title="Import data"
          subtitle="Bring your customers, sites, jobs and open invoices with you."
        />

        <section
          aria-label="Import from"
          className="grid grid-cols-1 gap-3 sm:grid-cols-3"
        >
          {SOURCES.map((s) => (
            <button
              key={s.key}
              type="button"
              aria-pressed={source === s.key}
              onClick={() => chooseSource(s.key)}
              className={cn(
                'flex items-center gap-3 rounded-xl border p-4 text-left',
                source === s.key
                  ? 'border-2 border-primary-600 bg-white'
                  : 'border-neutral-200 bg-white hover:border-neutral-300'
              )}
            >
              <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-neutral-100 font-semibold text-neutral-700">
                {s.icon ?? <FileSpreadsheet className="h-4 w-4" />}
              </span>
              <span className="font-medium text-neutral-900">{s.label}</span>
            </button>
          ))}
        </section>

        <div
          role="tablist"
          aria-label="What to import"
          className="mt-6 flex flex-wrap gap-2"
        >
          {IMPORT_ENTITIES.map((e, i) => (
            <button
              key={e}
              role="tab"
              type="button"
              aria-selected={entity === e}
              onClick={() => chooseEntity(e)}
              className={cn(
                'rounded-full px-4 py-1.5 text-sm font-medium',
                entity === e
                  ? 'bg-primary-600 text-white'
                  : 'bg-neutral-100 text-neutral-700 hover:bg-neutral-200'
              )}
            >
              {i + 1}. {ENTITY_LABELS[e]}
            </button>
          ))}
        </div>
        <p className="mt-2 text-sm text-neutral-500">
          Import customers first: sites, jobs and invoices are matched to them.
          Importing the same file again won’t create duplicates.
        </p>

        <section className="mt-4 rounded-2xl border border-neutral-200 bg-white p-5">
          <p className="text-neutral-700">{HELP[source][entity]}</p>
          <div className="mt-4 flex flex-wrap items-center gap-3">
            <input
              ref={inputRef}
              type="file"
              accept={IMPORT_ACCEPT}
              aria-label="Choose file"
              className="hidden"
              onChange={(e) => onFile(e.target.files?.[0])}
            />
            <Button
              variant="outline"
              className="gap-2"
              disabled={busy !== null}
              onClick={() => inputRef.current?.click()}
            >
              <Upload className="h-4 w-4" />
              {file ? 'Choose another file' : 'Choose file'}
            </Button>
            {file && (
              <span className="text-sm text-neutral-600">{file.name}</span>
            )}
            {busy === 'preview' && (
              <span className="text-sm text-neutral-500">
                Reading the file…
              </span>
            )}
          </div>
          <p className="mt-2 text-xs text-neutral-500">
            CSV or Excel (.xlsx), up to 10 MB and 50,000 rows.
          </p>
        </section>

        {error && (
          <div
            role="alert"
            className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800"
          >
            {error}
          </div>
        )}

        {preview && !job && (
          <>
            {showMapping && (
              <section className="mt-4 rounded-2xl border border-neutral-200 bg-white p-5">
                <h2 className="text-lg font-semibold text-neutral-900">
                  Match your columns
                </h2>
                {preview.missing.length > 0 && (
                  <p className="mt-1 text-sm text-amber-800">
                    Still needed: {preview.missing.join(', ')}
                  </p>
                )}
                <div className="mt-3 grid grid-cols-1 gap-x-6 gap-y-2 sm:grid-cols-2">
                  {fields.map((f) => (
                    <label
                      key={f.key}
                      className="flex items-center justify-between gap-3 text-sm"
                    >
                      <span className="text-neutral-700">
                        {f.label}
                        {f.required && <span className="text-red-600"> *</span>}
                      </span>
                      <select
                        aria-label={`Column for ${f.label}`}
                        value={preview.mapping[f.key] ?? ''}
                        disabled={busy !== null}
                        onChange={(e) => changeMapping(f.key, e.target.value)}
                        className="w-48 rounded-md border border-neutral-300 px-2 py-1 text-sm"
                      >
                        <option value="">— not imported —</option>
                        {preview.headers.map((h) => (
                          <option key={h} value={h}>
                            {h}
                          </option>
                        ))}
                      </select>
                    </label>
                  ))}
                </div>
              </section>
            )}

            <section className="mt-4 rounded-2xl border border-neutral-200 bg-white p-5">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h2 className="text-lg font-semibold text-neutral-900">
                    Preview
                  </h2>
                  <p
                    className="text-sm text-neutral-600"
                    data-testid="import-summary"
                  >
                    {preview.total_rows} rows: {preview.valid_rows} ready
                    {preview.skip_rows > 0 && `, ${preview.skip_rows} skipped`}
                    {preview.error_rows > 0 &&
                      `, ${preview.error_rows} with problems`}
                  </p>
                </div>
                <Button
                  onClick={() => void startImport()}
                  disabled={!canImport}
                >
                  Import {preview.valid_rows}{' '}
                  {ENTITY_LABELS[entity].toLowerCase()}
                </Button>
              </div>

              {preview.rows.length > 0 && shownFields.length > 0 && (
                <div className="mt-4 overflow-x-auto">
                  <table className="min-w-full text-left text-sm">
                    <thead className="text-xs uppercase text-neutral-500">
                      <tr>
                        <th className="py-2 pr-3">Row</th>
                        {shownFields.map((k) => (
                          <th key={k} className="py-2 pr-3">
                            {label(k)}
                          </th>
                        ))}
                        <th className="py-2">Status</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-neutral-100">
                      {preview.rows.map((r) => (
                        <tr key={r.row}>
                          <td className="py-2 pr-3 text-neutral-500">
                            {r.row}
                          </td>
                          {shownFields.map((k) => (
                            <td key={k} className="py-2 pr-3 text-neutral-800">
                              {r.values[k] ?? ''}
                            </td>
                          ))}
                          <td className="py-2">
                            {r.errors?.length ? (
                              <span className="text-red-700">
                                {r.errors[0].message}
                              </span>
                            ) : r.skip ? (
                              <span className="text-neutral-500">
                                Skipped: {r.skip}
                              </span>
                            ) : (
                              <span className="text-green-700">Ready</span>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {preview.errors.length > 0 && (
                <div className="mt-4">
                  <h3 className="flex items-center gap-2 text-sm font-semibold text-neutral-900">
                    <AlertTriangle className="h-4 w-4 text-amber-600" />
                    Rows that won’t be imported
                  </h3>
                  <ul className="mt-2 max-h-60 space-y-1 overflow-y-auto text-sm text-neutral-700">
                    {preview.errors.map((e, i) => (
                      <li key={`${e.row}-${i}`}>
                        Row {e.row}: {e.message}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </section>
          </>
        )}

        {job && <ImportResult job={job} onAgain={reset} />}
      </div>
    </DashboardLayout>
  );
}

function ImportResult({
  job,
  onAgain,
}: {
  job: ImportJob;
  onAgain: () => void;
}) {
  const done = isImportFinished(job);
  const pct =
    job.total_rows > 0
      ? Math.round((job.processed_rows / job.total_rows) * 100)
      : 0;
  const link = RESULT_LINKS[job.entity];
  return (
    <section
      className="mt-4 rounded-2xl border border-neutral-200 bg-white p-5"
      aria-live="polite"
    >
      <h2 className="flex items-center gap-2 text-lg font-semibold text-neutral-900">
        {done ? (
          <>
            <CheckCircle2 className="h-5 w-5 text-green-600" />
            {job.status === 'completed' ? 'Import finished' : 'Import failed'}
          </>
        ) : (
          `Importing… ${pct}%`
        )}
      </h2>
      <div className="mt-3 h-2 w-full rounded-full bg-neutral-100">
        <div
          className="h-2 rounded-full bg-primary-600"
          style={{ width: `${done ? 100 : pct}%` }}
        />
      </div>
      <p className="mt-3 text-sm text-neutral-700" data-testid="import-result">
        {job.successful_rows} imported, {job.skipped_rows} already there or
        skipped, {job.failed_rows} failed (of {job.total_rows}).
      </p>
      {!done && (
        <p className="mt-1 text-xs text-neutral-500">
          You can leave this page: the import keeps going in the background.
        </p>
      )}
      {job.error_summary && done && (
        <p className="mt-1 text-sm text-red-700">{job.error_summary}</p>
      )}
      {job.errors && job.errors.length > 0 && (
        <ul className="mt-3 max-h-60 space-y-1 overflow-y-auto text-sm text-neutral-700">
          {job.errors.map((e, i) => (
            <li key={`${e.row_number}-${i}`}>
              Row {e.row_number}: {e.error}
            </li>
          ))}
        </ul>
      )}
      {done && (
        <div className="mt-4 flex flex-wrap gap-3">
          <Link
            to={link.to}
            className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700"
          >
            {link.label}
          </Link>
          <Button variant="outline" onClick={onAgain}>
            Import another file
          </Button>
        </div>
      )}
    </section>
  );
}
