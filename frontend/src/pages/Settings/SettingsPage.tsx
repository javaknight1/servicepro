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

  const tabs = [
    { id: 'profile' as const, label: 'Profile', icon: User },
  ];

  return (
    <DashboardLayout>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-neutral-900">
            Account Settings
          </h1>
          <p className="text-neutral-600 mt-2">
            Manage your account profile.
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Sidebar */}
          <div className="lg:col-span-1">
            <Card variant="elevated" padding="sm">
              <nav className="space-y-1">
                {tabs.map((tab) => {
                  const Icon = tab.icon;
                  return (
                    <button
                      key={tab.id}
                      onClick={() => setActiveTab(tab.id)}
                      className={`w-full flex items-center px-4 py-3 text-sm font-medium rounded-lg transition-colors ${
                        activeTab === tab.id
                          ? 'bg-primary-50 text-primary-700'
                          : 'text-neutral-700 hover:bg-neutral-50'
                      }`}
                    >
                      <Icon className="h-5 w-5 mr-3" />
                      {tab.label}
                    </button>
                  );
                })}
              </nav>
            </Card>
          </div>

          {/* Content */}
          <div className="lg:col-span-3">
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
