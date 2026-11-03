import { useEffect, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Button, Input, Modal } from '@components/shared';
import { siteService, type Site, type SiteInput } from '@services/siteService';
import { getErrorMessage, getFieldErrors } from '@/utils/error';

interface SiteFormModalProps {
  customerId: string;
  /** The site to edit; omitted to add a new one. */
  site?: Site | null;
  isOpen: boolean;
  onClose: () => void;
}

type SiteForm = Required<Omit<SiteInput, 'is_active'>>;

const emptyForm: SiteForm = {
  name: '',
  address_street: '',
  address_city: '',
  address_state: '',
  address_zip: '',
  access_notes: '',
  contact_name: '',
  contact_phone: '',
  contact_email: '',
};

function toForm(site: Site | null | undefined): SiteForm {
  if (!site) return emptyForm;
  return {
    name: site.name,
    address_street: site.address_street,
    address_city: site.address_city,
    address_state: site.address_state,
    address_zip: site.address_zip,
    access_notes: site.access_notes ?? '',
    contact_name: site.contact_name ?? '',
    contact_phone: site.contact_phone ?? '',
    contact_email: site.contact_email ?? '',
  };
}

/** Add or edit one of a customer's sites (T186, mockup 06). */
export function SiteFormModal({
  customerId,
  site,
  isOpen,
  onClose,
}: SiteFormModalProps) {
  const queryClient = useQueryClient();
  const isEdit = !!site;
  const [form, setForm] = useState<SiteForm>(() => toForm(site));

  const save = useMutation({
    mutationFn: (data: SiteForm) => {
      const body: SiteInput = {
        ...data,
        name: data.name.trim(),
        address_state: data.address_state.trim().toUpperCase(),
      };
      if (site) return siteService.update(customerId, site.id, body);
      // New sites: leave empty optional fields out.
      const create: SiteInput = {
        name: body.name,
        address_street: body.address_street,
        address_city: body.address_city,
        address_state: body.address_state,
        address_zip: body.address_zip,
        access_notes: body.access_notes || undefined,
        contact_name: body.contact_name || undefined,
        contact_phone: body.contact_phone || undefined,
        contact_email: body.contact_email || undefined,
      };
      return siteService.create(customerId, create);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['customer-sites', customerId],
      });
      onClose();
    },
  });

  const resetSave = save.reset;
  useEffect(() => {
    if (!isOpen) return;
    setForm(toForm(site));
    resetSave();
  }, [isOpen, site, resetSave]);

  const fieldErrors = save.error ? getFieldErrors(save.error) : {};
  const set =
    (name: keyof SiteForm) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>): void => {
      const { value } = e.target;
      setForm((prev) => ({ ...prev, [name]: value }));
    };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEdit ? `Edit ${site?.name}` : 'Add site'}
      description="A building or property you do work at: its address, how to get in and who to ask on site."
      size="xl"
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          e.stopPropagation();
          save.mutate(form);
        }}
        className="space-y-4"
      >
        {save.error && Object.keys(fieldErrors).length === 0 && (
          <p
            role="alert"
            className="rounded-lg border border-error-200 bg-error-50 p-3 text-sm text-error-700"
          >
            {getErrorMessage(save.error, 'Could not save the site')}
          </p>
        )}
        <Input
          id="site-name"
          label="Site name *"
          value={form.name}
          onChange={set('name')}
          required
          maxLength={200}
          placeholder="Harborview Lofts"
          error={fieldErrors.name}
          fullWidth
        />
        <Input
          id="site-street"
          label="Street address *"
          value={form.address_street}
          onChange={set('address_street')}
          required
          maxLength={100}
          error={fieldErrors.address_street}
          fullWidth
        />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-[1fr_5rem_7rem]">
          <Input
            id="site-city"
            label="City *"
            value={form.address_city}
            onChange={set('address_city')}
            required
            maxLength={50}
            error={fieldErrors.address_city}
            fullWidth
          />
          <Input
            id="site-state"
            label="State *"
            value={form.address_state}
            onChange={set('address_state')}
            required
            minLength={2}
            maxLength={2}
            placeholder="OH"
            className="uppercase"
            error={fieldErrors.address_state}
            fullWidth
          />
          <Input
            id="site-zip"
            label="ZIP *"
            value={form.address_zip}
            onChange={set('address_zip')}
            required
            maxLength={10}
            error={fieldErrors.address_zip}
            fullWidth
          />
        </div>
        <div className="flex flex-col">
          <label
            htmlFor="site-access-notes"
            className="mb-1.5 text-sm font-medium text-neutral-700"
          >
            Access notes
          </label>
          <textarea
            id="site-access-notes"
            value={form.access_notes}
            onChange={set('access_notes')}
            rows={3}
            maxLength={5000}
            placeholder="Alarm code, key box, loading dock, after-hours entry..."
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
          />
        </div>
        <fieldset className="space-y-4 rounded-xl border border-neutral-200 p-4">
          <legend className="px-1 text-sm font-medium text-neutral-700">
            Site contact
          </legend>
          <Input
            id="site-contact-name"
            label="Name"
            value={form.contact_name}
            onChange={set('contact_name')}
            maxLength={100}
            error={fieldErrors.contact_name}
            fullWidth
          />
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Input
              id="site-contact-phone"
              label="Phone"
              type="tel"
              value={form.contact_phone}
              onChange={set('contact_phone')}
              maxLength={20}
              error={fieldErrors.contact_phone}
              fullWidth
            />
            <Input
              id="site-contact-email"
              label="Email"
              type="email"
              value={form.contact_email}
              onChange={set('contact_email')}
              maxLength={255}
              error={fieldErrors.contact_email}
              fullWidth
            />
          </div>
        </fieldset>
        <div className="flex justify-end gap-3 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" isLoading={save.isPending}>
            {isEdit ? 'Save site' : 'Add site'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
