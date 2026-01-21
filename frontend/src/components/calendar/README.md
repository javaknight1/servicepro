# Calendar Component

A comprehensive, feature-rich calendar component for managing job schedules in ServicePro.

## Features

- ✅ **Multiple Views**: Month, Week, Day, and Agenda views
- ✅ **Drag & Drop**: Intuitive event rescheduling
- ✅ **Event Resizing**: Adjust job duration directly
- ✅ **Status Colors**: Visual job status indicators
- ✅ **Priority Badges**: Priority level visibility
- ✅ **Responsive Design**: Mobile, tablet, and desktop support
- ✅ **Error Boundaries**: Graceful error handling
- ✅ **Loading States**: Skeleton loaders
- ✅ **Accessibility**: Full keyboard navigation and ARIA labels
- ✅ **TypeScript**: Full type safety

## Installation

### Required Dependencies

```bash
npm install react-big-calendar date-fns lucide-react
```

### Optional Dependencies (for testing)

```bash
npm install -D vitest @testing-library/react @testing-library/user-event
```

## Usage

### Basic Usage

```tsx
import { Calendar, JobEvent, JobStatus } from '@/components/calendar';

function MyCalendar() {
  const events: JobEvent[] = [
    {
      id: '1',
      title: 'Install HVAC System',
      start: new Date(2024, 0, 15, 9, 0),
      end: new Date(2024, 0, 15, 12, 0),
      jobNumber: 'JOB-001',
      status: JobStatus.SCHEDULED,
      location: '123 Main St',
    },
  ];

  return <Calendar events={events} />;
}
```

### With Event Handlers

```tsx
import { Calendar, JobEvent, DragDropEvent } from '@/components/calendar';

function MyCalendar() {
  const [events, setEvents] = useState<JobEvent[]>([...]);

  const handleEventClick = (event: JobEvent) => {
    console.log('Event clicked:', event);
    // Open event details modal
  };

  const handleEventDrop = async (data: DragDropEvent) => {
    console.log('Event moved:', data);
    // Update event in backend
    await updateJobSchedule(data.event.id, {
      start: data.start,
      end: data.end,
    });
    // Update local state
    setEvents(prev => /* update logic */);
  };

  const handleSelectSlot = (slotInfo: { start: Date; end: Date }) => {
    console.log('Slot selected:', slotInfo);
    // Open create job modal
  };

  return (
    <Calendar
      events={events}
      onEventClick={handleEventClick}
      onEventDrop={handleEventDrop}
      onSelectSlot={handleSelectSlot}
      defaultView="week"
      isLoading={isLoadingEvents}
    />
  );
}
```

### With React Query

```tsx
import { useQuery, useMutation } from '@tanstack/react-query';
import { Calendar } from '@/components/calendar';

function MyCalendar() {
  // Fetch events
  const { data: events, isLoading } = useQuery({
    queryKey: ['jobs', 'calendar'],
    queryFn: fetchCalendarJobs,
  });

  // Update event mutation
  const updateEventMutation = useMutation({
    mutationFn: updateJobSchedule,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs', 'calendar'] });
    },
  });

  const handleEventDrop = async (data: DragDropEvent) => {
    await updateEventMutation.mutateAsync({
      jobId: data.event.id,
      start: data.start,
      end: data.end,
    });
  };

  return (
    <Calendar
      events={events || []}
      isLoading={isLoading}
      onEventDrop={handleEventDrop}
    />
  );
}
```

## Props

### CalendarProps

| Prop            | Type                                     | Default      | Description                        |
| --------------- | ---------------------------------------- | ------------ | ---------------------------------- |
| `events`        | `JobEvent[]`                             | Required     | Array of job events to display     |
| `onEventClick`  | `(event: JobEvent) => void`              | -            | Called when event is clicked       |
| `onEventDrop`   | `(data: DragDropEvent) => Promise<void>` | -            | Called when event is dragged       |
| `onEventResize` | `(data: ResizeEvent) => Promise<void>`   | -            | Called when event is resized       |
| `onSelectSlot`  | `(slotInfo) => void`                     | -            | Called when empty slot is selected |
| `onNavigate`    | `(date: Date) => void`                   | -            | Called when navigating dates       |
| `onViewChange`  | `(view: CalendarView) => void`           | -            | Called when view changes           |
| `defaultView`   | `CalendarView`                           | `'month'`    | Initial view                       |
| `defaultDate`   | `Date`                                   | `new Date()` | Initial date                       |
| `isLoading`     | `boolean`                                | `false`      | Show loading skeleton              |
| `className`     | `string`                                 | -            | Additional CSS classes             |

### JobEvent

```typescript
interface JobEvent {
  id: string;
  title: string;
  start: Date;
  end: Date;
  jobNumber: string;
  status: JobStatus;
  priority?: JobPriority;
  description?: string;
  assignedTo?: string[];
  customerId?: string;
  customerName?: string;
  location?: string;
  resource?: any;
}
```

### Job Status

```typescript
enum JobStatus {
  SCHEDULED = 'scheduled',
  IN_PROGRESS = 'in_progress',
  ON_HOLD = 'on_hold',
  COMPLETED = 'completed',
  CANCELLED = 'cancelled',
}
```

### Job Priority

```typescript
enum JobPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  URGENT = 'urgent',
}
```

## Styling

### Custom Styles

The calendar uses Tailwind CSS. You can customize colors in `styles.ts`:

```typescript
import { getEventStyles } from '@/components/calendar/styles';

// Customize event styles
const customEventStyle = getEventStyles(JobStatus.SCHEDULED);
```

### Status Colors

- **Scheduled**: Blue (bg-blue-500)
- **In Progress**: Yellow (bg-yellow-500)
- **On Hold**: Orange (bg-orange-500)
- **Completed**: Green (bg-green-500)
- **Cancelled**: Gray (bg-gray-500)

### Priority Colors

- **Low**: Gray (bg-gray-200)
- **Medium**: Blue (bg-blue-200)
- **High**: Orange (bg-orange-200)
- **Urgent**: Red (bg-red-200)

## Accessibility

The calendar is fully accessible:

- ✅ Keyboard navigation
- ✅ ARIA labels
- ✅ Screen reader support
- ✅ Focus indicators
- ✅ Semantic HTML

### Keyboard Shortcuts

- `Arrow Keys`: Navigate calendar
- `Enter`: Select event/slot
- `Tab`: Navigate toolbar buttons

## Testing

### Unit Tests

```bash
npm test Calendar.test.tsx
npm test CalendarEvent.test.tsx
npm test CalendarToolbar.test.tsx
```

### Integration Tests

```bash
npm test Calendar.integration.test.tsx
```

### Coverage

Run all tests with coverage:

```bash
npm test -- --coverage
```

## Performance

### Optimizations

- ✅ Memoized components
- ✅ Efficient event rendering
- ✅ Lazy loading
- ✅ Debounced handlers

### Best Practices

```typescript
// Memoize event handlers
const handleEventClick = useCallback((event: JobEvent) => {
  // Handle click
}, []);

// Memoize events array
const memoizedEvents = useMemo(() => events, [events]);
```

## Error Handling

The calendar includes built-in error boundaries:

```typescript
// Errors are caught and displayed gracefully
<Calendar
  events={events}
  onEventDrop={async (data) => {
    try {
      await updateEvent(data);
    } catch (error) {
      // Error is logged and user is notified
      console.error('Failed to update event:', error);
    }
  }}
/>
```

## Responsive Design

The calendar automatically adapts to screen sizes:

- **Mobile**: Compact event display, vertical toolbar
- **Tablet**: Standard layout with reduced spacing
- **Desktop**: Full features, horizontal toolbar

## Examples

### With Custom Toolbar

```typescript
import { CalendarToolbar } from '@/components/calendar';

// Use standalone toolbar
<CalendarToolbar
  date={currentDate}
  view={currentView}
  onNavigate={handleNavigate}
  onViewChange={handleViewChange}
  label="January 2024"
/>
```

### With Custom Event Component

```typescript
// The calendar uses CalendarEvent by default
// You can customize it in styles.ts
```

## Troubleshooting

### Events not showing

Ensure events have valid `start` and `end` dates:

```typescript
const events = [
  {
    ...event,
    start: new Date(event.startDate), // Must be Date object
    end: new Date(event.endDate), // Must be Date object
  },
];
```

### Drag and drop not working

Ensure you provide the `onEventDrop` handler:

```typescript
<Calendar
  events={events}
  onEventDrop={handleEventDrop} // Required for drag and drop
/>
```

### Styling issues

Import React Big Calendar CSS:

```typescript
import 'react-big-calendar/lib/css/react-big-calendar.css';
```

## Contributing

See [CONTRIBUTING.md](../../../../CONTRIBUTING.md) for development guidelines.

## License

Part of ServicePro - Internal use only.
