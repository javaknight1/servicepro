import { useState, useEffect } from 'react';
import { DashboardLayout } from '@components/layout';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Button,
  Input,
} from '@components/shared';
import { ProfilePictureSection } from '@components/settings/ProfilePictureSection';
import { useAuthStore } from '@store';
import { userService } from '@services/userService';
import { User } from 'lucide-react';

export function SettingsPage() {
  const user = useAuthStore((state) => state.user);
  const setUser = useAuthStore((state) => state.setUser);
  const fetchCurrentUser = useAuthStore((state) => state.fetchCurrentUser);
  // Password, 2FA and notification preferences are hidden until Neon Auth
  // supplies them (T158, T259); only the profile tab does anything today.
  const [activeTab, setActiveTab] = useState<'profile'>('profile');

  // Profile form state
  const [firstName, setFirstName] = useState(user?.first_name || '');
  const [lastName, setLastName] = useState(user?.last_name || '');
  const [isSaving, setIsSaving] = useState(false);
  const [saveMessage, setSaveMessage] = useState<{
    type: 'success' | 'error';
    text: string;
  } | null>(null);

  // Ensure user data is loaded
  useEffect(() => {
    fetchCurrentUser();
  }, [fetchCurrentUser]);

  // Update form when user data changes
  useEffect(() => {
    if (user) {
      setFirstName(user.first_name || '');
      setLastName(user.last_name || '');
    }
  }, [user]);

  const handleSaveProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setSaveMessage(null);

    try {
      const updatedUser = await userService.updateProfile({
        first_name: firstName || null,
        last_name: lastName || null,
      });
      setUser(updatedUser);
      setSaveMessage({ type: 'success', text: 'Profile updated successfully' });
    } catch (error) {
      console.error('Failed to update profile:', error);
      setSaveMessage({ type: 'error', text: 'Failed to update profile' });
    } finally {
      setIsSaving(false);
    }
  };

  const handleCancelProfile = () => {
    setFirstName(user?.first_name || '');
    setLastName(user?.last_name || '');
    setSaveMessage(null);
  };

  const tabs = [{ id: 'profile' as const, label: 'Profile', icon: User }];

  return (
    <DashboardLayout
      breadcrumbs={[{ label: 'Account settings' }, { label: 'Profile' }]}
    >
      <div className="mx-auto max-w-[1440px] p-4 sm:p-6 lg:p-8">
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[280px_minmax(0,1fr)]">
          {/* Sidebar */}
          <div className="lg:sticky lg:top-24 lg:self-start">
            <Card variant="elevated" padding="sm">
              <nav className="space-y-1">
                {tabs.map((tab) => {
                  const Icon = tab.icon;
                  return (
                    <button
                      key={tab.id}
                      onClick={() => setActiveTab(tab.id)}
                      aria-current={activeTab === tab.id ? 'page' : undefined}
                      className={`w-full flex items-center px-4 py-3 text-[15px] rounded-xl transition-colors ${
                        activeTab === tab.id
                          ? 'bg-primary-50 text-primary-600'
                          : 'text-neutral-800 hover:bg-neutral-50'
                      }`}
                    >
                      <Icon className="h-5 w-5 mr-3" strokeWidth={1.75} />
                      {tab.label}
                    </button>
                  );
                })}
              </nav>
            </Card>
          </div>

          {/* Content */}
          <div className="min-w-0 space-y-6">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-neutral-900 md:text-3xl">
                Account settings
              </h1>
              <p className="mt-1 text-base text-neutral-600">
                Your own profile, the same in every organization.
              </p>
            </div>
            {activeTab === 'profile' && (
              <Card variant="elevated" padding="lg">
                <CardHeader>
                  <CardTitle>Profile Information</CardTitle>
                  <CardDescription>
                    Update your account profile information
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <ProfilePictureSection />
                  <form className="space-y-4 mt-6" onSubmit={handleSaveProfile}>
                    <Input
                      label="Email Address"
                      type="email"
                      value={user?.email || ''}
                      fullWidth
                      disabled
                      helperText="Contact support to change your email address"
                    />

                    <Input
                      label="First Name"
                      type="text"
                      placeholder="Enter your first name"
                      value={firstName}
                      onChange={(e) => setFirstName(e.target.value)}
                      fullWidth
                    />

                    <Input
                      label="Last Name"
                      type="text"
                      placeholder="Enter your last name"
                      value={lastName}
                      onChange={(e) => setLastName(e.target.value)}
                      fullWidth
                    />

                    {saveMessage && (
                      <div
                        className={`text-sm ${
                          saveMessage.type === 'success'
                            ? 'text-green-600'
                            : 'text-red-600'
                        }`}
                      >
                        {saveMessage.text}
                      </div>
                    )}

                    <div className="flex justify-end space-x-3 pt-4">
                      <Button
                        type="button"
                        variant="outline"
                        onClick={handleCancelProfile}
                      >
                        Cancel
                      </Button>
                      <Button
                        type="submit"
                        variant="primary"
                        disabled={isSaving}
                      >
                        {isSaving ? 'Saving...' : 'Save Changes'}
                      </Button>
                    </div>
                  </form>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}
