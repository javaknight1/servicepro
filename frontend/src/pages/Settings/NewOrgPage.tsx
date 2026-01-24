import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import { Card, CardContent, Button, Input } from '@components/shared';
import { useTenantStore } from '@store';
import { Building2 } from 'lucide-react';
import type { CreateTenantRequest } from '@/types/tenant';

export function NewOrgPage() {
  const navigate = useNavigate();
  const { createTenant, setCurrentTenant, isLoading, error, clearError } =
    useTenantStore();

  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');

  // Auto-generate slug from name
  const handleNameChange = (value: string) => {
    setName(value);
    // Generate slug from name if slug hasn't been manually edited
    const generatedSlug = value
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, '')
      .replace(/\s+/g, '-')
      .replace(/-+/g, '-')
      .replace(/^-|-$/g, '');
    setSlug(generatedSlug);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    clearError();

    try {
      const data: CreateTenantRequest = {
        name: name.trim(),
        slug: slug.trim(),
        email: email.trim() || undefined,
        phone: phone.trim() || undefined,
      };

      const newTenant = await createTenant(data);
      setCurrentTenant(newTenant);
      // Navigate to dashboard after creating organization
      navigate('/dashboard');
    } catch (err) {
      console.error('Failed to create organization:', err);
    }
  };

  return (
    <DashboardLayout>
      <div className="max-w-2xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8 text-center">
          <div className="inline-flex items-center justify-center h-16 w-16 rounded-full bg-primary-100 mb-4">
            <Building2 className="h-8 w-8 text-primary-600" />
          </div>
          <h1 className="text-3xl font-bold text-neutral-900">
            Create Organization
          </h1>
          <p className="text-neutral-600 mt-2">
            Set up a new organization to manage your business
          </p>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
            {error}
          </div>
        )}

        <Card variant="elevated" padding="lg">
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-6">
              <Input
                label="Organization Name"
                type="text"
                value={name}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder="Acme Corporation"
                required
                fullWidth
                helperText="The display name for your organization"
              />

              <Input
                label="Organization Slug"
                type="text"
                value={slug}
                onChange={(e) =>
                  setSlug(
                    e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '')
                  )
                }
                placeholder="acme-corporation"
                required
                fullWidth
                helperText="URL-friendly identifier (lowercase letters, numbers, and hyphens only)"
              />

              <Input
                label="Contact Email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="contact@acme.com"
                fullWidth
                helperText="Optional - used for organization-level notifications"
              />

              <Input
                label="Phone Number"
                type="tel"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="+1 (555) 123-4567"
                fullWidth
                helperText="Optional - primary contact number"
              />

              <div className="flex justify-end space-x-3 pt-4">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => navigate(-1)}
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  disabled={isLoading || !name.trim() || !slug.trim()}
                >
                  {isLoading ? 'Creating...' : 'Create Organization'}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </DashboardLayout>
  );
}
