# Role Management Components

Comprehensive React components for managing roles, permissions, and role assignments with full CRUD operations, audit trails, and bulk operations.

## Components

### 1. RoleManager

Main role management interface with role listing, search, selection, and bulk operations.

#### Features

- ✅ Role listing with search functionality
- ✅ Single and bulk role selection
- ✅ Create, edit, and delete roles
- ✅ Hierarchy level badges
- ✅ Permission count display
- ✅ System role protection (level >= 80 cannot be deleted)
- ✅ Loading and error states
- ✅ Modal-based forms

#### Usage

```tsx
import { RoleManager } from '@components/roles';

function RolesPage() {
  const handleRoleSelect = (role: Role) => {
    console.log('Selected role:', role);
  };

  return <RoleManager onRoleSelect={handleRoleSelect} />;
}
```

#### Props

| Prop           | Type                   | Description                              |
| -------------- | ---------------------- | ---------------------------------------- |
| `onRoleSelect` | `(role: Role) => void` | Optional callback when a role is clicked |

### 2. RoleForm

Form component for creating and editing roles with permission selection.

#### Features

- ✅ Create and edit modes
- ✅ Form validation with Zod
- ✅ Permission selection with search
- ✅ Permissions grouped by resource
- ✅ Select all/deselect all permissions
- ✅ Parent role selection
- ✅ Hierarchy level input
- ✅ Real-time validation feedback
- ✅ Loading states during submission

#### Usage

```tsx
import { RoleForm } from '@components/roles';

// Create mode
function CreateRoleModal() {
  return (
    <RoleForm
      onSuccess={() => console.log('Role created')}
      onCancel={() => console.log('Cancelled')}
    />
  );
}

// Edit mode
function EditRoleModal({ role }) {
  return (
    <RoleForm
      role={role}
      onSuccess={() => console.log('Role updated')}
      onCancel={() => console.log('Cancelled')}
    />
  );
}
```

#### Props

| Prop        | Type         | Description                                 |
| ----------- | ------------ | ------------------------------------------- |
| `role`      | `Role`       | Optional. If provided, form is in edit mode |
| `onSuccess` | `() => void` | Optional callback on successful submission  |
| `onCancel`  | `() => void` | Optional callback when cancel is clicked    |

#### Validation Rules

- **Name**: Required, max 100 characters, alphanumeric with underscores and hyphens
- **Description**: Optional, max 500 characters
- **Hierarchy Level**: Required, 0-100
- **Parent Role**: Optional

### 3. RoleAudit

Audit trail component for tracking all role and permission changes.

#### Features

- ✅ Comprehensive audit log display
- ✅ Action-based filtering
- ✅ Date range filtering
- ✅ Search functionality
- ✅ Export to CSV
- ✅ Expandable details
- ✅ Color-coded action badges
- ✅ Timestamp formatting
- ✅ Performer information

#### Usage

```tsx
import { RoleAudit } from '@components/roles';

function AuditPage() {
  return <RoleAudit />;
}
```

#### Audit Actions

| Action               | Description                  | Color            |
| -------------------- | ---------------------------- | ---------------- |
| `created`            | Role was created             | Green (Success)  |
| `updated`            | Role was modified            | Blue (Primary)   |
| `deleted`            | Role was deleted             | Red (Error)      |
| `assigned`           | Role was assigned to user    | Teal (Secondary) |
| `unassigned`         | Role was removed from user   | Yellow (Warning) |
| `permission_added`   | Permission added to role     | Green (Success)  |
| `permission_removed` | Permission removed from role | Red (Error)      |

## API Integration

All components use React Query for data fetching and caching. The following hooks are provided:

### Queries

```tsx
import {
  useRoles,
  useRole,
  usePermissions,
  useRolePermissions,
  useRoleAuditLogs,
} from '@hooks/useRoles';

// Get all roles with optional filters
const {
  data: roles,
  isLoading,
  error,
} = useRoles({
  search: 'admin',
  hierarchy_level: 90,
});

// Get single role
const { data: role } = useRole(roleId);

// Get all permissions
const { data: permissions } = usePermissions();

// Get role's permissions
const { data: rolePerms } = useRolePermissions(roleId);

// Get audit logs with filters
const { data: auditLogs } = useRoleAuditLogs({
  action: 'created',
  start_date: '2024-01-01',
  end_date: '2024-12-31',
});
```

### Mutations

```tsx
import {
  useCreateRole,
  useUpdateRole,
  useDeleteRole,
  useAssignRole,
  useUnassignRole,
  useBulkAssignRoles,
  useUpdateRolePermissions,
  useExportAuditLogs,
} from '@hooks/useRoles';

// Create role
const createMutation = useCreateRole();
await createMutation.mutateAsync({
  name: 'new_role',
  description: 'Description',
  hierarchy_level: 50,
  permission_ids: ['p1', 'p2'],
});

// Update role
const updateMutation = useUpdateRole();
await updateMutation.mutateAsync({
  id: 'role-id',
  data: { name: 'updated_name' },
});

// Delete role
const deleteMutation = useDeleteRole();
await deleteMutation.mutateAsync('role-id');

// Assign role to user
const assignMutation = useAssignRole();
await assignMutation.mutateAsync({
  user_id: 'user-id',
  role_id: 'role-id',
  expires_at: '2024-12-31T23:59:59Z', // Optional
});

// Bulk assign role to multiple users
const bulkAssignMutation = useBulkAssignRoles();
await bulkAssignMutation.mutateAsync({
  user_ids: ['user1', 'user2', 'user3'],
  role_id: 'role-id',
});

// Export audit logs
const exportMutation = useExportAuditLogs();
await exportMutation.mutateAsync({
  start_date: '2024-01-01',
  end_date: '2024-12-31',
});
```

## TypeScript Types

```typescript
interface Role {
  id: string;
  name: string;
  description: string;
  hierarchy_level: number;
  parent_role_id?: string | null;
  permissions?: Permission[];
  created_at: string;
  updated_at: string;
}

interface Permission {
  id: string;
  name: string;
  description: string;
  resource: string;
  action: string;
  created_at: string;
  updated_at: string;
}

interface RoleAuditLog {
  id: string;
  role_id: string;
  role_name: string;
  action:
    | 'created'
    | 'updated'
    | 'deleted'
    | 'assigned'
    | 'unassigned'
    | 'permission_added'
    | 'permission_removed';
  performed_by: string;
  performed_by_name?: string;
  details?: Record<string, any>;
  timestamp: string;
}
```

## Testing

Comprehensive test suites are provided for all components using React Testing Library.

### Running Tests

```bash
# Run all role component tests
npm test -- --testPathPattern=roles

# Run specific component tests
npm test -- RoleManager.test
npm test -- RoleForm.test
npm test -- RoleAudit.test

# Run with coverage
npm test -- --coverage --testPathPattern=roles
```

### Test Coverage

- ✅ Component rendering
- ✅ User interactions (clicks, typing, selections)
- ✅ Form validation
- ✅ API calls and mutations
- ✅ Loading and error states
- ✅ Search and filtering
- ✅ Bulk operations
- ✅ Modal interactions

## Styling

Components use Tailwind CSS with a custom design system featuring:

- **Primary**: Dark Orange (`#ea580c`)
- **Secondary**: Teal (`#0d9488`)
- **Success**: Emerald Green (`#059669`)
- **Warning**: Amber (`#d97706`)
- **Error**: Red (`#dc2626`)

All components are fully responsive and accessible.

## Backend Requirements

These components expect the following backend endpoints:

### Role Endpoints

- `GET /api/v1/roles` - List roles
- `GET /api/v1/roles/:id` - Get role by ID
- `POST /api/v1/roles` - Create role
- `PUT /api/v1/roles/:id` - Update role
- `DELETE /api/v1/roles/:id` - Delete role

### Permission Endpoints

- `GET /api/v1/permissions` - List all permissions
- `GET /api/v1/roles/:id/permissions` - Get role permissions
- `PUT /api/v1/roles/:id/permissions` - Update role permissions

### Assignment Endpoints

- `POST /api/v1/roles/assign` - Assign role to user
- `DELETE /api/v1/roles/unassign/:userId/:roleId` - Unassign role
- `POST /api/v1/roles/bulk-assign` - Bulk assign role

### Audit Endpoints

- `GET /api/v1/roles/audit` - Get audit logs
- `GET /api/v1/roles/audit/export` - Export audit logs as CSV

## Best Practices

1. **Always validate user permissions** before allowing role management operations
2. **Use hierarchy levels** to enforce role-based access control (higher levels can manage lower levels)
3. **Protect system roles** (level >= 80) from deletion
4. **Enable audit logging** to track all role changes
5. **Set expiration dates** for temporary role assignments
6. **Use parent roles** to implement role inheritance
7. **Group permissions by resource** for better organization

## Example Integration

```tsx
import { RoleManager, RoleAudit } from '@components/roles';
import { Tabs, TabList, Tab, TabPanels, TabPanel } from '@components/shared';

function RoleManagementPage() {
  return (
    <div className="container mx-auto p-6">
      <h1 className="text-3xl font-bold mb-6">Role Management</h1>

      <Tabs>
        <TabList>
          <Tab>Roles</Tab>
          <Tab>Audit Trail</Tab>
        </TabList>

        <TabPanels>
          <TabPanel>
            <RoleManager />
          </TabPanel>

          <TabPanel>
            <RoleAudit />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </div>
  );
}
```

## Dependencies

- `react` - ^18.0.0
- `react-query` - ^5.0.0
- `react-hook-form` - ^7.0.0
- `zod` - ^3.0.0
- `lucide-react` - ^0.300.0
- `date-fns` - ^3.0.0
- `axios` - ^1.0.0

## License

MIT
