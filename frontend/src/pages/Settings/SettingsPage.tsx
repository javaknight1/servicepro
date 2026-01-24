import { useState } from 'react';
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
import { User, Lock, Bell, Shield } from 'lucide-react';

export function SettingsPage() {
  const user = useAuthStore((state) => state.user);
  const [activeTab, setActiveTab] = useState<
    'profile' | 'security' | 'notifications'
  >('profile');

  const tabs = [
    { id: 'profile' as const, label: 'Profile', icon: User },
    { id: 'security' as const, label: 'Security', icon: Lock },
    { id: 'notifications' as const, label: 'Notifications', icon: Bell },
  ];

  return (
    <DashboardLayout>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-neutral-900">
            Account Settings
          </h1>
          <p className="text-neutral-600 mt-2">
            Manage your account preferences and security settings.
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
                  <form className="space-y-4 mt-6">
                    <Input
                      label="Email Address"
                      type="email"
                      defaultValue={user?.email || ''}
                      fullWidth
                      disabled
                      helperText="Contact support to change your email address"
                    />

                    <Input
                      label="Full Name"
                      type="text"
                      placeholder="Enter your full name"
                      fullWidth
                    />

                    <Input
                      label="Company"
                      type="text"
                      placeholder="Enter your company name"
                      fullWidth
                    />

                    <div className="flex justify-end space-x-3 pt-4">
                      <Button variant="outline">Cancel</Button>
                      <Button variant="primary">Save Changes</Button>
                    </div>
                  </form>
                </CardContent>
              </Card>
            )}

            {activeTab === 'security' && (
              <Card variant="elevated" padding="lg">
                <CardHeader>
                  <CardTitle>Security Settings</CardTitle>
                  <CardDescription>
                    Manage your password and security preferences
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <form className="space-y-4 mt-6">
                    <Input
                      label="Current Password"
                      type="password"
                      autoComplete="current-password"
                      fullWidth
                    />

                    <Input
                      label="New Password"
                      type="password"
                      autoComplete="new-password"
                      fullWidth
                      helperText="Must be at least 8 characters"
                    />

                    <Input
                      label="Confirm New Password"
                      type="password"
                      autoComplete="new-password"
                      fullWidth
                    />

                    <div className="flex justify-end space-x-3 pt-4">
                      <Button variant="outline">Cancel</Button>
                      <Button variant="primary">Update Password</Button>
                    </div>
                  </form>

                  <div className="mt-8 pt-8 border-t border-neutral-200">
                    <div className="flex items-start">
                      <Shield className="h-5 w-5 text-primary-600 mt-0.5" />
                      <div className="ml-3">
                        <h4 className="text-sm font-medium text-neutral-900">
                          Two-Factor Authentication
                        </h4>
                        <p className="text-sm text-neutral-600 mt-1">
                          Add an extra layer of security to your account (coming
                          soon)
                        </p>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}

            {activeTab === 'notifications' && (
              <Card variant="elevated" padding="lg">
                <CardHeader>
                  <CardTitle>Notification Preferences</CardTitle>
                  <CardDescription>
                    Choose what updates you want to receive
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4 mt-6">
                    <div className="flex items-center justify-between py-3">
                      <div>
                        <p className="text-sm font-medium text-neutral-900">
                          Email Notifications
                        </p>
                        <p className="text-sm text-neutral-600">
                          Receive email about your account activity
                        </p>
                      </div>
                      <input
                        type="checkbox"
                        className="h-4 w-4 text-primary-600 rounded"
                        defaultChecked
                      />
                    </div>

                    <div className="flex items-center justify-between py-3">
                      <div>
                        <p className="text-sm font-medium text-neutral-900">
                          Security Alerts
                        </p>
                        <p className="text-sm text-neutral-600">
                          Get notified of suspicious activity
                        </p>
                      </div>
                      <input
                        type="checkbox"
                        className="h-4 w-4 text-primary-600 rounded"
                        defaultChecked
                      />
                    </div>

                    <div className="flex items-center justify-between py-3">
                      <div>
                        <p className="text-sm font-medium text-neutral-900">
                          Product Updates
                        </p>
                        <p className="text-sm text-neutral-600">
                          News about product features and updates
                        </p>
                      </div>
                      <input
                        type="checkbox"
                        className="h-4 w-4 text-primary-600 rounded"
                      />
                    </div>

                    <div className="flex justify-end space-x-3 pt-4">
                      <Button variant="outline">Cancel</Button>
                      <Button variant="primary">Save Preferences</Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}
