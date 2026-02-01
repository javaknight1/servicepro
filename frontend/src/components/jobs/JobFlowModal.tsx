import { Modal, Button } from '@components/shared';
import { ArrowRight } from 'lucide-react';

interface JobFlowModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const statusInfo = [
  {
    status: 'new',
    label: 'New',
    description: 'Job has been created but not yet scheduled',
    color: 'bg-gray-100 text-gray-800 border-gray-300',
  },
  {
    status: 'scheduled',
    label: 'Scheduled',
    description: 'Job is scheduled for a specific date/time',
    color: 'bg-blue-100 text-blue-800 border-blue-300',
  },
  {
    status: 'en_route',
    label: 'En Route',
    description: 'Technician is traveling to the job site (optional)',
    color: 'bg-indigo-100 text-indigo-800 border-indigo-300',
  },
  {
    status: 'in_progress',
    label: 'In Progress',
    description: 'Work is actively being performed',
    color: 'bg-yellow-100 text-yellow-800 border-yellow-300',
  },
  {
    status: 'on_hold',
    label: 'On Hold',
    description:
      'Work is temporarily paused (waiting for parts, customer, etc.)',
    color: 'bg-orange-100 text-orange-800 border-orange-300',
  },
  {
    status: 'completed',
    label: 'Completed',
    description: 'Work has been finished',
    color: 'bg-green-100 text-green-800 border-green-300',
  },
  {
    status: 'invoiced',
    label: 'Invoiced',
    description: 'Invoice has been sent to customer',
    color: 'bg-purple-100 text-purple-800 border-purple-300',
  },
  {
    status: 'paid',
    label: 'Paid',
    description: 'Payment has been received',
    color: 'bg-emerald-100 text-emerald-800 border-emerald-300',
  },
  {
    status: 'cancelled',
    label: 'Cancelled',
    description: 'Job has been cancelled',
    color: 'bg-red-100 text-red-800 border-red-300',
  },
];

export function JobFlowModal({ isOpen, onClose }: JobFlowModalProps) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Job Status Flow" size="lg">
      <div className="space-y-6">
        <p className="text-neutral-600">
          Jobs progress through the following statuses. Most transitions happen
          automatically or through the &quot;Set [Status]&quot; button.
        </p>

        {/* Main Flow */}
        <div className="bg-neutral-50 rounded-lg p-4">
          <h3 className="text-sm font-semibold text-neutral-700 mb-3">
            Standard Flow
          </h3>
          <div className="flex flex-wrap items-center gap-2">
            {[
              'new',
              'scheduled',
              'in_progress',
              'completed',
              'invoiced',
              'paid',
            ].map((status, index, arr) => {
              const info = statusInfo.find((s) => s.status === status);
              return (
                <div key={status} className="flex items-center gap-2">
                  <span
                    className={`px-3 py-1 rounded-full text-xs font-medium border ${info?.color}`}
                  >
                    {info?.label}
                  </span>
                  {index < arr.length - 1 && (
                    <ArrowRight className="h-4 w-4 text-neutral-400" />
                  )}
                </div>
              );
            })}
          </div>
        </div>

        {/* Optional En Route */}
        <div className="bg-neutral-50 rounded-lg p-4">
          <h3 className="text-sm font-semibold text-neutral-700 mb-3">
            Optional: En Route Status
          </h3>
          <div className="flex items-center gap-2">
            <span className="px-3 py-1 rounded-full text-xs font-medium border bg-blue-100 text-blue-800 border-blue-300">
              Scheduled
            </span>
            <ArrowRight className="h-4 w-4 text-neutral-400" />
            <span className="px-3 py-1 rounded-full text-xs font-medium border bg-indigo-100 text-indigo-800 border-indigo-300">
              En Route
            </span>
            <ArrowRight className="h-4 w-4 text-neutral-400" />
            <span className="px-3 py-1 rounded-full text-xs font-medium border bg-yellow-100 text-yellow-800 border-yellow-300">
              In Progress
            </span>
          </div>
        </div>

        {/* On Hold */}
        <div className="bg-neutral-50 rounded-lg p-4">
          <h3 className="text-sm font-semibold text-neutral-700 mb-3">
            On Hold Flow
          </h3>
          <div className="flex items-center gap-2">
            <span className="px-3 py-1 rounded-full text-xs font-medium border bg-yellow-100 text-yellow-800 border-yellow-300">
              In Progress
            </span>
            <ArrowRight className="h-4 w-4 text-neutral-400" />
            <span className="px-3 py-1 rounded-full text-xs font-medium border bg-orange-100 text-orange-800 border-orange-300">
              On Hold
            </span>
            <ArrowRight className="h-4 w-4 text-neutral-400" />
            <span className="px-3 py-1 rounded-full text-xs font-medium border bg-yellow-100 text-yellow-800 border-yellow-300">
              In Progress
            </span>
          </div>
        </div>

        {/* Status Descriptions */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-neutral-700">
            Status Descriptions
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {statusInfo.map((info) => (
              <div
                key={info.status}
                className="flex items-start gap-3 p-3 bg-white rounded-lg border border-neutral-200"
              >
                <span
                  className={`px-2 py-0.5 rounded text-xs font-medium border ${info.color}`}
                >
                  {info.label}
                </span>
                <p className="text-sm text-neutral-600">{info.description}</p>
              </div>
            ))}
          </div>
        </div>

        <div className="flex justify-end">
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </Modal>
  );
}
