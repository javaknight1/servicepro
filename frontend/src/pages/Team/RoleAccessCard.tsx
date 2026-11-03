import { CircleCheck, EyeOff } from 'lucide-react';
import { Card, CardContent } from '@components/shared';

type Access = 'yes' | 'no' | 'own';

/** "What each role can see" (mockup 13). Mirrors the API's role grants. */
const ROWS: { area: string; owner: Access; office: Access; tech: Access }[] = [
  { area: 'Schedule & dispatch', owner: 'yes', office: 'yes', tech: 'own' },
  { area: 'Customers & sites', owner: 'yes', office: 'yes', tech: 'own' },
  {
    area: 'Prices, quotes & agreement amounts',
    owner: 'yes',
    office: 'yes',
    tech: 'no',
  },
  { area: 'Invoices & payments', owner: 'yes', office: 'yes', tech: 'no' },
  {
    area: 'Reports (A/R aging, Revenue)',
    owner: 'yes',
    office: 'yes',
    tech: 'no',
  },
  {
    area: 'Billing, plan & integrations',
    owner: 'yes',
    office: 'no',
    tech: 'no',
  },
];

function Cell({ access }: { access: Access }) {
  if (access === 'yes')
    return (
      <CircleCheck
        className="mx-auto h-5 w-5 text-secondary-600"
        aria-label="Yes"
      />
    );
  if (access === 'own')
    return (
      <span className="whitespace-nowrap rounded-full border border-primary-200 bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700">
        Own visits
      </span>
    );
  return (
    <span className="text-neutral-400" aria-label="No">
      —
    </span>
  );
}

/** The fixed role matrix and the technicians-never-see-money note. */
export function RoleAccessCard() {
  return (
    <div className="space-y-6">
      <Card variant="elevated" padding="lg">
        <CardContent>
          <h2 className="text-lg font-semibold text-neutral-900">
            What each role can see
          </h2>
          <p className="mt-1 text-sm text-neutral-600">
            Three fixed roles. No setup needed.
          </p>
          <table className="mt-4 w-full text-sm">
            <thead>
              <tr className="text-neutral-600">
                <th className="py-2 text-left font-normal">
                  <span className="sr-only">Area</span>
                </th>
                <th className="px-2 py-2 font-medium">Owner</th>
                <th className="px-2 py-2 font-medium">Office</th>
                <th className="px-2 py-2 font-medium">Technician</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-200">
              {ROWS.map((row) => (
                <tr key={row.area}>
                  <th
                    scope="row"
                    className="py-3 pr-2 text-left font-normal text-neutral-800"
                  >
                    {row.area}
                  </th>
                  <td className="py-3 text-center">
                    <Cell access={row.owner} />
                  </td>
                  <td className="py-3 text-center">
                    <Cell access={row.office} />
                  </td>
                  <td className="py-3 text-center">
                    <Cell access={row.tech} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </CardContent>
      </Card>

      <div className="rounded-2xl border border-secondary-200 bg-secondary-50 p-5">
        <p className="flex items-center gap-2 font-semibold text-secondary-800">
          <EyeOff className="h-5 w-5" />
          Technicians never see money
        </p>
        <p className="mt-2 text-sm text-secondary-800">
          No prices, agreement amounts, invoices or customer balances. They see
          the jobs assigned to them, site notes and sign-off.
        </p>
      </div>
    </div>
  );
}
