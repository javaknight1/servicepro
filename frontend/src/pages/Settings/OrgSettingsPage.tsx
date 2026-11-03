import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router';
import { DashboardLayout } from '@components/layout';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Button,
  Input,
  Modal,
  Badge,
  Avatar,
} from '@components/shared';
import { useTenantStore } from '@store';
import { tenantApi } from '@services/tenantService';
import { roleApi } from '@services/roleApi';
import {
  paymentReminderService,
  type PaymentReminderSettings,
} from '@services/paymentReminderService';
import { getTenantErrorMessage } from '@/utils/error';
import { useTenantPermissions } from '@/utils/tenantPermissions';
import {
  Building2,
  Users,
  Settings,
  Trash2,
  UserPlus,
  CreditCard,
  Mail,
  RefreshCw,
  X,
  Clock,
  Bell,
  Plus,
} from 'lucide-react';
import { MembershipTab } from '@components/membership';
import type {
  TenantMember,
  UpdateTenantRequest,
  Invitation,
} from '@/types/tenant';
import type { Role } from '@/types/role';

type TabId =
  'general' | 'members' | 'membership' | 'notifications' | 'timezone';

export function OrgSettingsPage() {
  const navigate = useNavigate();
  const {
    currentTenant,
    isLoading,
    error,
    updateTenant,
    deleteTenant,
    getMembers,
    removeMember,
    clearError,
  } = useTenantStore();

  const [activeTab, setActiveTab] = useState<TabId>('general');
  const [members, setMembers] = useState<TenantMember[]>([]);
  const [pendingInvitations, setPendingInvitations] = useState<Invitation[]>(
    []
  );
  const [roles, setRoles] = useState<Role[]>([]);
  const [membersLoading, setMembersLoading] = useState(false);
  const [membersError, setMembersError] = useState<string | null>(null);
  const [memberActionError, setMemberActionError] = useState<string | null>(
    null
  );

  // The caller's effective permissions in this tenant (GET /tenants/:id/me);
  // the API still enforces each one and answers 403.
  const {
    canManageMembers: canManage,
    canUpdateTenant: canEditSettings,
    canDeleteTenant: canDelete,
  } = useTenantPermissions(currentTenant);

  // Form state for general settings
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  // Modal state for inviting member
  const [showAddMemberModal, setShowAddMemberModal] = useState(false);
  const [newMemberEmail, setNewMemberEmail] = useState('');
  const [newMemberRoleId, setNewMemberRoleId] = useState('');
  const [isInviting, setIsInviting] = useState(false);
  const [inviteError, setInviteError] = useState<string | null>(null);

  // Modal state for delete confirmation
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteConfirmText, setDeleteConfirmText] = useState('');

  // Notification settings state
  const [reminderSettings, setReminderSettings] =
    useState<PaymentReminderSettings>({
      paymentRemindersEnabled: false,
      reminderDaysAfterDue: [7, 14, 30],
      maxRemindersPerInvoice: 3,
    });
  const [reminderSettingsLoading, setReminderSettingsLoading] = useState(false);
  const [reminderSettingsSaving, setReminderSettingsSaving] = useState(false);
  const [reminderSettingsError, setReminderSettingsError] = useState<
    string | null
  >(null);
  const [reminderSettingsSuccess, setReminderSettingsSuccess] = useState(false);

  // Timezone state. Operating hours are hidden until something reads them
  // (T099, T259); any saved businessHours are preserved on save.
  const [timezone, setTimezone] = useState('America/New_York');
  const [timezoneSaving, setTimezoneSaving] = useState(false);
  const [timezoneSuccess, setTimezoneSuccess] = useState(false);
  const [timezoneError, setTimezoneError] = useState<string | null>(null);

  const tabs = [
    { id: 'general' as const, label: 'Business', icon: Settings },
    { id: 'members' as const, label: 'Members', icon: Users },
    { id: 'membership' as const, label: 'Plan', icon: CreditCard },
    { id: 'notifications' as const, label: 'Notifications', icon: Bell },
    { id: 'timezone' as const, label: 'Time zone', icon: Clock },
  ];
  const activeTabLabel =
    tabs.find((tab) => tab.id === activeTab)?.label ?? 'Settings';

  // Load current tenant data into form
  useEffect(() => {
    if (currentTenant) {
      setName(currentTenant.name);
      setEmail(currentTenant.email || '');
      setPhone(currentTenant.phone || '');
    }
  }, [currentTenant]);

  const loadMembers = async () => {
    if (!currentTenant) return;
    setMembersLoading(true);
    setMembersError(null);
    setMemberActionError(null);
    try {
      const memberList = await getMembers(currentTenant.id);
      setMembers(memberList);

      // Invitations and roles are management data (tenants.manage)
      if (canManage) {
        try {
          const invitationsResponse = await tenantApi.getPendingInvitations(
            currentTenant.id
          );
          setPendingInvitations(invitationsResponse.data || []);
        } catch (invErr) {
          console.error('Failed to load invitations:', invErr);
          setPendingInvitations([]);
        }

        // Load roles for the invite modal
        try {
          const roleList = await roleApi.getRoles().then((res) => res.data);
          setRoles(roleList);
        } catch (roleErr) {
          console.error('Failed to load roles:', roleErr);
        }
      } else {
        setPendingInvitations([]);
      }
    } catch (err: unknown) {
      console.error('Failed to load members:', err);
      setMembers([]);
      setPendingInvitations([]);
      setMembersError(getTenantErrorMessage(err, 'Failed to load members'));
    } finally {
      setMembersLoading(false);
    }
  };

  // Load members when members tab is active
  useEffect(() => {
    if (activeTab === 'members' && currentTenant) {
      loadMembers();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab, currentTenant, canManage]);

  const loadReminderSettings = async () => {
    setReminderSettingsLoading(true);
    setReminderSettingsError(null);
    try {
      const settings = await paymentReminderService.getReminderSettings();
      setReminderSettings(settings);
    } catch (err) {
      console.error('Failed to load reminder settings:', err);
      setReminderSettingsError(
        getTenantErrorMessage(err, 'Failed to load reminder settings')
      );
    } finally {
      setReminderSettingsLoading(false);
    }
  };

  // Load notification settings when tab is active
  useEffect(() => {
    if (activeTab === 'notifications' && currentTenant) {
      loadReminderSettings();
    }
  }, [activeTab, currentTenant]);

  // Load the timezone when the tab is active
  useEffect(() => {
    if (activeTab === 'timezone' && currentTenant?.settings?.timezone) {
      setTimezone(currentTenant.settings.timezone);
    }
  }, [activeTab, currentTenant]);

  const handleSaveTimezone = async () => {
    if (!currentTenant) return;

    setTimezoneSaving(true);
    setTimezoneError(null);
    setTimezoneSuccess(false);
    try {
      const mergedSettings = { ...currentTenant.settings, timezone };
      await updateTenant(currentTenant.id, { settings: mergedSettings });
      setTimezoneSuccess(true);
      setTimeout(() => setTimezoneSuccess(false), 3000);
    } catch (err) {
      console.error('Failed to save timezone:', err);
      setTimezoneError(getTenantErrorMessage(err, 'Failed to save timezone'));
    } finally {
      setTimezoneSaving(false);
    }
  };

  const handleSaveReminderSettings = async () => {
    setReminderSettingsSaving(true);
    setReminderSettingsError(null);
    setReminderSettingsSuccess(false);
    try {
      const updated =
        await paymentReminderService.updateReminderSettings(reminderSettings);
      setReminderSettings(updated);
      setReminderSettingsSuccess(true);
      setTimeout(() => setReminderSettingsSuccess(false), 3000);
    } catch (err) {
      console.error('Failed to save reminder settings:', err);
      setReminderSettingsError(
        getTenantErrorMessage(err, 'Failed to save reminder settings')
      );
    } finally {
      setReminderSettingsSaving(false);
    }
  };

  const handleAddReminderDay = () => {
    const days = [...reminderSettings.reminderDaysAfterDue];
    const lastDay = days.length > 0 ? days[days.length - 1] : 0;
    days.push(lastDay + 7);
    setReminderSettings({ ...reminderSettings, reminderDaysAfterDue: days });
  };

  const handleRemoveReminderDay = (index: number) => {
    const days = reminderSettings.reminderDaysAfterDue.filter(
      (_, i) => i !== index
    );
    setReminderSettings({ ...reminderSettings, reminderDaysAfterDue: days });
  };

  const handleReminderDayChange = (index: number, value: number) => {
    const days = [...reminderSettings.reminderDaysAfterDue];
    days[index] = value;
    setReminderSettings({ ...reminderSettings, reminderDaysAfterDue: days });
  };

  const handleSaveGeneral = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentTenant) return;

    setIsSaving(true);
    clearError();

    try {
      const data: UpdateTenantRequest = {
        name: name.trim(),
        email: email.trim() || undefined,
        phone: phone.trim() || undefined,
      };
      await updateTenant(currentTenant.id, data);
    } catch (err) {
      console.error('Failed to save:', err);
    } finally {
      setIsSaving(false);
    }
  };

  const handleInviteMember = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentTenant || !newMemberEmail.trim() || !newMemberRoleId) return;

    setIsInviting(true);
    setInviteError(null);
    try {
      await tenantApi.inviteMember(currentTenant.id, {
        email: newMemberEmail.trim(),
        role_id: newMemberRoleId,
      });
      setShowAddMemberModal(false);
      setNewMemberEmail('');
      setNewMemberRoleId('');
      loadMembers();
    } catch (err) {
      console.error('Failed to invite member:', err);
      setInviteError(getTenantErrorMessage(err, 'Failed to send invitation'));
    } finally {
      setIsInviting(false);
    }
  };

  const handleCancelInvitation = async (invitation: Invitation) => {
    if (!currentTenant) return;
    if (
      !confirm(
        `Are you sure you want to cancel the invitation for ${invitation.email}?`
      )
    )
      return;

    try {
      await tenantApi.cancelInvitation(currentTenant.id, invitation.id);
      loadMembers();
    } catch (err: unknown) {
      console.error('Failed to cancel invitation:', err);
      setMemberActionError(
        getTenantErrorMessage(err, 'Failed to cancel invitation')
      );
    }
  };

  const handleResendInvitation = async (invitation: Invitation) => {
    if (!currentTenant) return;

    try {
      await tenantApi.resendInvitation(currentTenant.id, invitation.id);
      alert('Invitation resent successfully');
    } catch (err: unknown) {
      console.error('Failed to resend invitation:', err);
      setMemberActionError(
        getTenantErrorMessage(err, 'Failed to resend invitation')
      );
    }
  };

  const handleRemoveMember = async (userId: string) => {
    if (!currentTenant) return;
    if (!confirm('Are you sure you want to remove this member?')) return;

    try {
      await removeMember(currentTenant.id, userId);
      loadMembers();
    } catch (err: unknown) {
      console.error('Failed to remove member:', err);
      setMemberActionError(
        getTenantErrorMessage(err, 'Failed to remove member')
      );
    }
  };

  const handleDeleteOrganization = async () => {
    if (!currentTenant) return;
    if (deleteConfirmText !== currentTenant.name) return;

    try {
      await deleteTenant(currentTenant.id);
      navigate('/dashboard');
    } catch (err) {
      console.error('Failed to delete organization:', err);
    }
  };

  if (!currentTenant) {
    return (
      <DashboardLayout>
        <div className="mx-auto max-w-[1440px] p-4 sm:p-6 lg:p-8">
          <Card variant="elevated" padding="lg">
            <CardContent>
              <div className="text-center py-8">
                <Building2 className="h-12 w-12 text-neutral-400 mx-auto mb-4" />
                <h2 className="text-lg font-medium text-neutral-900 mb-2">
                  No Organization Selected
                </h2>
                <p className="text-neutral-600 mb-4">
                  Please select or create an organization to manage settings.
                </p>
                <Button
                  variant="primary"
                  onClick={() => navigate('/settings/organization/new')}
                >
                  Create Organization
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout
      breadcrumbs={[
        { label: 'Settings', to: '/settings/organization' },
        { label: activeTabLabel },
      ]}
    >
      <div className="mx-auto max-w-[1440px] p-4 sm:p-6 lg:p-8">
        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
            {error}
          </div>
        )}

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
                {activeTabLabel}
              </h1>
              <p className="mt-1 text-base text-neutral-600">
                Settings for{' '}
                <span className="font-medium">{currentTenant.name}</span>
              </p>
            </div>
            {activeTab === 'general' && (
              <>
                <Card variant="elevated" padding="lg">
                  <CardHeader>
                    <CardTitle>Organization Information</CardTitle>
                    <CardDescription>
                      Update your organization's basic information
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <form
                      onSubmit={handleSaveGeneral}
                      className="space-y-4 mt-6"
                    >
                      {!canEditSettings && (
                        <p className="text-sm text-neutral-600">
                          Only organization admins can change these settings.
                        </p>
                      )}

                      <Input
                        label="Organization Name"
                        type="text"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        required
                        fullWidth
                        disabled={!canEditSettings}
                      />

                      <Input
                        label="Email Address"
                        type="email"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        placeholder="contact@company.com"
                        fullWidth
                        disabled={!canEditSettings}
                      />

                      <Input
                        label="Phone Number"
                        type="tel"
                        value={phone}
                        onChange={(e) => setPhone(e.target.value)}
                        placeholder="+1 (555) 123-4567"
                        fullWidth
                        disabled={!canEditSettings}
                      />

                      {canEditSettings && (
                        <div className="flex justify-end space-x-3 pt-4">
                          <Button
                            type="button"
                            variant="outline"
                            onClick={() => {
                              setName(currentTenant.name);
                              setEmail(currentTenant.email || '');
                              setPhone(currentTenant.phone || '');
                            }}
                          >
                            Reset
                          </Button>
                          <Button
                            type="submit"
                            variant="primary"
                            disabled={isSaving || isLoading}
                          >
                            {isSaving ? 'Saving...' : 'Save Changes'}
                          </Button>
                        </div>
                      )}
                    </form>
                  </CardContent>
                </Card>

                {/* Danger Zone (tenants.delete) */}
                {canDelete && (
                  <Card
                    variant="elevated"
                    padding="lg"
                    className="border-red-200"
                  >
                    <CardHeader>
                      <CardTitle className="text-red-700">
                        Danger Zone
                      </CardTitle>
                      <CardDescription>
                        Irreversible and destructive actions
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="flex items-center justify-between py-4">
                        <div>
                          <p className="text-sm font-medium text-neutral-900">
                            Delete Organization
                          </p>
                          <p className="text-sm text-neutral-600">
                            Permanently delete this organization and all its
                            data
                          </p>
                        </div>
                        <Button
                          variant="outline"
                          className="text-red-600 border-red-300 hover:bg-red-50"
                          onClick={() => setShowDeleteModal(true)}
                        >
                          <Trash2 className="h-4 w-4 mr-2" />
                          Delete
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                )}
              </>
            )}

            {activeTab === 'members' && (
              <Card variant="elevated" padding="lg">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle>Team Members</CardTitle>
                      <CardDescription>
                        Manage who has access to this organization
                      </CardDescription>
                    </div>
                    {canManage && !membersError && (
                      <Button
                        variant="primary"
                        size="sm"
                        onClick={() => setShowAddMemberModal(true)}
                      >
                        <UserPlus className="h-4 w-4 mr-2" />
                        Invite Member
                      </Button>
                    )}
                  </div>
                </CardHeader>
                <CardContent>
                  {memberActionError && (
                    <div
                      role="alert"
                      className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm"
                    >
                      {memberActionError}
                    </div>
                  )}
                  {membersLoading ? (
                    <div className="py-8 text-center text-neutral-500">
                      Loading members...
                    </div>
                  ) : membersError ? (
                    <div
                      role="alert"
                      className="py-8 text-center text-neutral-600"
                    >
                      {membersError}
                    </div>
                  ) : members.length === 0 &&
                    pendingInvitations.length === 0 ? (
                    <div className="py-8 text-center text-neutral-500">
                      No members found
                    </div>
                  ) : (
                    <div className="space-y-6 mt-4">
                      {/* Active Members */}
                      {members.length > 0 && (
                        <div className="divide-y divide-neutral-200">
                          {members.map((member) => (
                            <div
                              key={member.id}
                              className="flex items-center justify-between py-4"
                            >
                              <div className="flex items-center">
                                <Avatar
                                  email={member.email}
                                  profilePictureUrl={member.profile_picture_url}
                                  size="md"
                                />
                                <div className="ml-4">
                                  <p className="text-sm font-medium text-neutral-900">
                                    {member.email}
                                  </p>
                                  <div className="flex items-center mt-1 space-x-2">
                                    <Badge variant="neutral">
                                      {member.role_name}
                                    </Badge>
                                  </div>
                                </div>
                              </div>
                              {canManage && (
                                <div className="flex items-center space-x-2">
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() =>
                                      handleRemoveMember(member.user_id)
                                    }
                                    className="text-red-600 hover:text-red-700 hover:bg-red-50"
                                    title="Remove member"
                                  >
                                    <Trash2 className="h-4 w-4" />
                                  </Button>
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      )}

                      {/* Pending Invitations */}
                      {pendingInvitations.length > 0 && (
                        <div>
                          <h3 className="text-sm font-medium text-neutral-700 mb-3 flex items-center">
                            <Clock className="h-4 w-4 mr-2" />
                            Pending Invitations ({pendingInvitations.length})
                          </h3>
                          <div className="divide-y divide-neutral-200 bg-neutral-50 rounded-lg">
                            {pendingInvitations.map((invitation) => (
                              <div
                                key={invitation.id}
                                className="flex items-center justify-between py-4 px-4"
                              >
                                <div className="flex items-center">
                                  <div className="h-10 w-10 rounded-full bg-neutral-200 flex items-center justify-center">
                                    <Mail className="h-5 w-5 text-neutral-500" />
                                  </div>
                                  <div className="ml-4">
                                    <p className="text-sm font-medium text-neutral-900">
                                      {invitation.email}
                                    </p>
                                    <div className="flex items-center mt-1 space-x-2">
                                      <Badge variant="neutral">
                                        {invitation.role_name}
                                      </Badge>
                                      <Badge variant="warning">
                                        {invitation.user_registered
                                          ? 'Awaiting Acceptance'
                                          : 'Awaiting Registration'}
                                      </Badge>
                                    </div>
                                    <p className="text-xs text-neutral-500 mt-1">
                                      Expires{' '}
                                      {new Date(
                                        invitation.expires_at
                                      ).toLocaleDateString()}
                                    </p>
                                  </div>
                                </div>
                                <div className="flex items-center space-x-2">
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() =>
                                      handleResendInvitation(invitation)
                                    }
                                    title="Resend invitation"
                                  >
                                    <RefreshCw className="h-4 w-4" />
                                  </Button>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() =>
                                      handleCancelInvitation(invitation)
                                    }
                                    className="text-red-600 hover:text-red-700 hover:bg-red-50"
                                    title="Cancel invitation"
                                  >
                                    <X className="h-4 w-4" />
                                  </Button>
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </CardContent>
              </Card>
            )}

            {activeTab === 'membership' && <MembershipTab />}

            {activeTab === 'notifications' && (
              <Card variant="elevated" padding="lg">
                <CardHeader>
                  <CardTitle>Payment Reminders</CardTitle>
                  <CardDescription>
                    Configure automatic payment reminders for overdue invoices
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  {reminderSettingsLoading ? (
                    <div className="py-8 text-center text-neutral-500">
                      Loading settings...
                    </div>
                  ) : (
                    <div className="space-y-6 mt-4">
                      {reminderSettingsError && (
                        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
                          {reminderSettingsError}
                        </div>
                      )}

                      {reminderSettingsSuccess && (
                        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm">
                          Settings saved successfully
                        </div>
                      )}

                      {/* Enable toggle */}
                      <div className="flex items-center justify-between">
                        <div>
                          <p className="text-sm font-medium text-neutral-900">
                            Enable automatic payment reminders
                          </p>
                          <p className="text-sm text-neutral-500">
                            Automatically send reminders when invoices become
                            overdue
                          </p>
                        </div>
                        <label className="relative inline-flex items-center cursor-pointer">
                          <input
                            type="checkbox"
                            checked={reminderSettings.paymentRemindersEnabled}
                            onChange={(e) =>
                              setReminderSettings({
                                ...reminderSettings,
                                paymentRemindersEnabled: e.target.checked,
                              })
                            }
                            className="sr-only peer"
                          />
                          <div className="w-11 h-6 bg-neutral-200 peer-focus:outline-hidden peer-focus:ring-4 peer-focus:ring-primary-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-neutral-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
                        </label>
                      </div>

                      {/* Reminder schedule */}
                      <div>
                        <div className="flex items-center justify-between mb-3">
                          <div>
                            <p className="text-sm font-medium text-neutral-900">
                              Reminder schedule
                            </p>
                            <p className="text-sm text-neutral-500">
                              Days after due date to send each reminder
                            </p>
                          </div>
                          <Button
                            type="button"
                            variant="secondary"
                            size="sm"
                            onClick={handleAddReminderDay}
                          >
                            <Plus className="h-4 w-4 mr-1" />
                            Add
                          </Button>
                        </div>
                        <div className="space-y-2">
                          {reminderSettings.reminderDaysAfterDue.map(
                            (day, index) => (
                              <div
                                key={index}
                                className="flex items-center gap-3"
                              >
                                <span className="text-sm text-neutral-500 w-24">
                                  Reminder {index + 1}:
                                </span>
                                <input
                                  type="number"
                                  value={day}
                                  onChange={(e) =>
                                    handleReminderDayChange(
                                      index,
                                      parseInt(e.target.value) || 0
                                    )
                                  }
                                  min="1"
                                  max="365"
                                  className="w-20 px-3 py-1.5 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 text-sm"
                                />
                                <span className="text-sm text-neutral-500">
                                  days after due date
                                </span>
                                {reminderSettings.reminderDaysAfterDue.length >
                                  1 && (
                                  <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    onClick={() =>
                                      handleRemoveReminderDay(index)
                                    }
                                    className="text-red-500 hover:text-red-700"
                                  >
                                    <X className="h-4 w-4" />
                                  </Button>
                                )}
                              </div>
                            )
                          )}
                        </div>
                      </div>

                      {/* Max reminders */}
                      <div>
                        <p className="text-sm font-medium text-neutral-900 mb-1">
                          Maximum reminders per invoice
                        </p>
                        <p className="text-sm text-neutral-500 mb-2">
                          Stop sending reminders after this many have been sent
                        </p>
                        <input
                          type="number"
                          value={reminderSettings.maxRemindersPerInvoice}
                          onChange={(e) =>
                            setReminderSettings({
                              ...reminderSettings,
                              maxRemindersPerInvoice:
                                parseInt(e.target.value) || 1,
                            })
                          }
                          min="1"
                          max="10"
                          className="w-20 px-3 py-1.5 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 text-sm"
                        />
                      </div>

                      {/* Save button */}
                      <div className="flex justify-end pt-4 border-t border-neutral-200">
                        <Button
                          type="button"
                          variant="primary"
                          onClick={handleSaveReminderSettings}
                          disabled={reminderSettingsSaving}
                        >
                          {reminderSettingsSaving
                            ? 'Saving...'
                            : 'Save Settings'}
                        </Button>
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            )}

            {activeTab === 'timezone' && (
              <Card variant="elevated" padding="lg">
                <CardHeader>
                  <CardTitle>Time Zone</CardTitle>
                  <CardDescription>
                    The time zone your business runs on
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-6 mt-4">
                    {timezoneError && (
                      <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
                        {timezoneError}
                      </div>
                    )}

                    {timezoneSuccess && (
                      <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm">
                        Time zone saved successfully
                      </div>
                    )}

                    <div>
                      <label
                        htmlFor="org-timezone"
                        className="block text-sm font-medium text-neutral-900 mb-1"
                      >
                        Timezone
                      </label>
                      <p className="text-sm text-neutral-500 mb-2">
                        Used for invoice due dates, payment terms and payment
                        reminders
                      </p>
                      <select
                        id="org-timezone"
                        value={timezone}
                        onChange={(e) => setTimezone(e.target.value)}
                        className="w-full max-w-xs px-3 py-2 border border-neutral-300 rounded-lg focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-primary-500 text-sm"
                      >
                        <option value="America/New_York">
                          Eastern (America/New_York)
                        </option>
                        <option value="America/Chicago">
                          Central (America/Chicago)
                        </option>
                        <option value="America/Denver">
                          Mountain (America/Denver)
                        </option>
                        <option value="America/Los_Angeles">
                          Pacific (America/Los_Angeles)
                        </option>
                      </select>
                    </div>

                    <div className="flex justify-end space-x-3 pt-4 border-t border-neutral-200">
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => {
                          setTimezone(
                            currentTenant?.settings?.timezone ??
                              'America/New_York'
                          );
                          setTimezoneError(null);
                        }}
                      >
                        Reset
                      </Button>
                      <Button
                        type="button"
                        variant="primary"
                        onClick={handleSaveTimezone}
                        disabled={timezoneSaving}
                      >
                        {timezoneSaving ? 'Saving...' : 'Save Time Zone'}
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>

      {/* Invite Member Modal */}
      <Modal
        isOpen={showAddMemberModal}
        onClose={() => {
          setShowAddMemberModal(false);
          setNewMemberEmail('');
          setNewMemberRoleId('');
          setInviteError(null);
        }}
        title="Invite Team Member"
      >
        <form onSubmit={handleInviteMember} className="space-y-4">
          <p className="text-sm text-neutral-600">
            Send an invitation to join your organization. If they don't have an
            account, they'll be prompted to register.
          </p>

          {inviteError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
              {inviteError}
            </div>
          )}

          <Input
            label="Email Address"
            type="email"
            value={newMemberEmail}
            onChange={(e) => setNewMemberEmail(e.target.value)}
            placeholder="member@company.com"
            required
            fullWidth
          />

          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">
              Role
            </label>
            <select
              value={newMemberRoleId}
              onChange={(e) => setNewMemberRoleId(e.target.value)}
              className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:outline-hidden focus:ring-2 focus:ring-primary-500"
              required
            >
              <option value="">Select a role</option>
              {roles.map((role) => (
                <option key={role.id} value={role.id}>
                  {role.name} - {role.description}
                </option>
              ))}
            </select>
          </div>

          <div className="flex justify-end space-x-3 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => setShowAddMemberModal(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={isInviting || !newMemberRoleId}
            >
              {isInviting ? 'Sending...' : 'Send Invitation'}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => {
          setShowDeleteModal(false);
          setDeleteConfirmText('');
        }}
        title="Delete Organization"
      >
        <div className="space-y-4">
          <div className="p-4 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-sm text-red-700">
              This action is irreversible. All data including customers, jobs,
              quotes, and invoices will be permanently deleted.
            </p>
          </div>

          <p className="text-sm text-neutral-600">
            Type{' '}
            <span className="font-mono font-bold">{currentTenant.name}</span> to
            confirm:
          </p>

          <Input
            type="text"
            value={deleteConfirmText}
            onChange={(e) => setDeleteConfirmText(e.target.value)}
            placeholder={currentTenant.name}
            fullWidth
          />

          <div className="flex justify-end space-x-3 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setShowDeleteModal(false);
                setDeleteConfirmText('');
              }}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="primary"
              className="bg-red-600 hover:bg-red-700"
              disabled={deleteConfirmText !== currentTenant.name}
              onClick={handleDeleteOrganization}
            >
              Delete Organization
            </Button>
          </div>
        </div>
      </Modal>
    </DashboardLayout>
  );
}
