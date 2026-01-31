import { z } from 'zod';

/**
 * Line Item Validation Schema
 */
export const lineItemSchema = z.object({
  id: z.string(),
  description: z
    .string()
    .min(1, 'Description is required')
    .max(500, 'Description must be less than 500 characters'),
  quantity: z
    .number()
    .positive('Quantity must be positive')
    .max(999999, 'Quantity is too large'),
  unit_price: z
    .number()
    .nonnegative('Unit price cannot be negative')
    .max(999999.99, 'Unit price is too large'),
  total: z.number(),
  sort_order: z.number(),
});

/**
 * Customer Information Schema
 */
export const customerSchema = z.object({
  customer_id: z.string().optional(),
  customer_name: z
    .string()
    .min(1, 'Customer name is required')
    .max(200, 'Name is too long'),
  customer_email: z
    .string()
    .email('Invalid email address')
    .max(255, 'Email is too long'),
  customer_phone: z
    .string()
    .regex(/^[\d\s\-\(\)\+]+$/, 'Invalid phone number format')
    .min(10, 'Phone number too short')
    .max(20, 'Phone number too long')
    .optional()
    .or(z.literal('')),
  customer_address: z
    .string()
    .max(500, 'Address is too long')
    .optional()
    .or(z.literal('')),
  customer_city: z
    .string()
    .max(100, 'City is too long')
    .optional()
    .or(z.literal('')),
  customer_state: z
    .string()
    .length(2, 'State must be 2 characters')
    .regex(/^[A-Z]{2}$/, 'State must be uppercase letters')
    .optional()
    .or(z.literal('')),
  customer_zip: z
    .string()
    .regex(/^\d{5}(-\d{4})?$/, 'Invalid ZIP code format')
    .optional()
    .or(z.literal('')),
});

/**
 * Terms and Conditions Schema
 */
export const termsSchema = z.object({
  valid_until: z
    .string()
    .datetime('Invalid date format')
    .refine(
      (date) => new Date(date) > new Date(),
      'Valid until date must be in the future'
    ),
  payment_terms: z
    .string()
    .max(500, 'Payment terms too long')
    .optional()
    .or(z.literal('')),
  delivery_timeline: z
    .string()
    .max(200, 'Delivery timeline too long')
    .optional()
    .or(z.literal('')),
  warranty_info: z
    .string()
    .max(500, 'Warranty info too long')
    .optional()
    .or(z.literal('')),
});

/**
 * Quote Form Schema
 */
export const quoteFormSchema = z
  .object({
    // Customer Information
    customer_id: z.string().optional(),
    customer_name: z
      .string()
      .min(1, 'Customer name is required')
      .max(200, 'Name is too long'),
    customer_email: z
      .string()
      .email('Invalid email address')
      .max(255, 'Email is too long'),
    customer_phone: z
      .string()
      .regex(/^[\d\s\-\(\)\+]+$/, 'Invalid phone number format')
      .min(10, 'Phone number too short')
      .max(20, 'Phone number too long')
      .optional()
      .or(z.literal('')),
    customer_address: z
      .string()
      .max(500, 'Address is too long')
      .optional()
      .or(z.literal('')),
    customer_city: z
      .string()
      .max(100, 'City is too long')
      .optional()
      .or(z.literal('')),
    customer_state: z
      .string()
      .length(2, 'State must be 2 characters')
      .regex(/^[A-Z]{2}$/, 'State must be uppercase letters')
      .optional()
      .or(z.literal('')),
    customer_zip: z
      .string()
      .regex(/^\d{5}(-\d{4})?$/, 'Invalid ZIP code format')
      .optional()
      .or(z.literal('')),

    // Line Items
    items: z
      .array(lineItemSchema)
      .min(1, 'At least one line item is required')
      .max(100, 'Too many line items'),

    // Tax Information
    tax_zip_code: z
      .string()
      .regex(/^\d{5}(-\d{4})?$/, 'Invalid ZIP code format')
      .optional()
      .or(z.literal('')),
    tax_rate: z
      .string()
      .regex(/^\d+(\.\d{1,4})?$/, 'Invalid tax rate format')
      .optional()
      .or(z.literal('')),
    tax_exemption_type: z
      .enum([
        'none',
        'nonprofit',
        'government',
        'resale',
        'manufacture',
        'educational',
      ])
      .optional(),
    tax_exemption_number: z.string().optional().or(z.literal('')),

    // Calculated Fields (read-only)
    subtotal: z.string().optional(),
    tax_amount: z.string().optional(),
    total: z.string().optional(),

    // Terms and Notes
    valid_until: z.string().datetime('Invalid date format'),
    payment_terms: z
      .string()
      .max(500, 'Payment terms too long')
      .optional()
      .or(z.literal('')),
    delivery_timeline: z
      .string()
      .max(200, 'Delivery timeline too long')
      .optional()
      .or(z.literal('')),
    warranty_info: z
      .string()
      .max(500, 'Warranty info too long')
      .optional()
      .or(z.literal('')),
    notes: z.string().max(2000, 'Notes too long').optional().or(z.literal('')),

    // Status
    status: z
      .enum(['draft', 'sent', 'viewed', 'accepted', 'declined', 'expired'])
      .default('draft'),
  })
  .refine(
    (data) => {
      // If tax exemption type is set, exemption number is required
      if (
        data.tax_exemption_type &&
        data.tax_exemption_type !== 'none' &&
        !data.tax_exemption_number
      ) {
        return false;
      }
      return true;
    },
    {
      message: 'Exemption number is required when exemption type is selected',
      path: ['tax_exemption_number'],
    }
  )
  .refine(
    (data) => {
      // Either customer_id or customer details must be provided
      if (!data.customer_id && !data.customer_name) {
        return false;
      }
      return true;
    },
    {
      message: 'Customer information is required',
      path: ['customer_name'],
    }
  );

/**
 * Type inference from schemas
 */
export type LineItemFormData = z.infer<typeof lineItemSchema>;
export type CustomerFormData = z.infer<typeof customerSchema>;
export type TermsFormData = z.infer<typeof termsSchema>;
export type QuoteFormData = z.infer<typeof quoteFormSchema>;

/**
 * Default values for new quote
 */
export const getDefaultQuoteValues = (): Partial<QuoteFormData> => {
  const validUntil = new Date();
  validUntil.setDate(validUntil.getDate() + 30); // 30 days from now

  return {
    customer_name: '',
    customer_email: '',
    customer_phone: '',
    customer_address: '',
    customer_city: '',
    customer_state: '',
    customer_zip: '',
    items: [],
    tax_zip_code: '',
    tax_rate: '',
    tax_exemption_type: 'none',
    tax_exemption_number: '',
    subtotal: '0.00',
    tax_amount: '0.00',
    total: '0.00',
    valid_until: validUntil.toISOString(),
    payment_terms: 'Net 30',
    delivery_timeline: '',
    warranty_info: '',
    notes: '',
    status: 'draft',
  };
};

/**
 * Validation error messages
 */
export const validationMessages = {
  required: 'This field is required',
  email: 'Please enter a valid email address',
  phone: 'Please enter a valid phone number',
  zipCode: 'Please enter a valid ZIP code (12345 or 12345-6789)',
  state: 'Please enter a 2-letter state code (e.g., CA)',
  positiveNumber: 'Value must be positive',
  futureDate: 'Date must be in the future',
  minItems: 'At least one line item is required',
  maxLength: (max: number) => `Maximum ${max} characters allowed`,
  minLength: (min: number) => `Minimum ${min} characters required`,
} as const;
