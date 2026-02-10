import api from './api';

export interface PaymentReminderResponse {
  id: string;
  invoice_id: string;
  invoice_number: string;
  customer_name: string;
  reminder_number: number;
  channel: 'email' | 'sms';
  status: 'pending' | 'sent' | 'failed';
  tone: 'friendly' | 'firm' | 'final';
  recipient: string;
  sent_at?: number;
  error_message?: string;
  created_at: string;
}

export interface PaymentReminderSettings {
  paymentRemindersEnabled: boolean;
  reminderDaysAfterDue: number[];
  maxRemindersPerInvoice: number;
}

class PaymentReminderService {
  async getReminderHistory(
    invoiceId: string
  ): Promise<PaymentReminderResponse[]> {
    const response = await api.get<PaymentReminderResponse[]>(
      `/v1/invoices/${invoiceId}/reminders`
    );
    return response.data;
  }

  async sendManualReminder(
    invoiceId: string
  ): Promise<PaymentReminderResponse> {
    const response = await api.post<PaymentReminderResponse>(
      `/v1/invoices/${invoiceId}/reminders`
    );
    return response.data;
  }

  async getReminderSettings(): Promise<PaymentReminderSettings> {
    const response = await api.get<PaymentReminderSettings>(
      '/v1/settings/payment-reminders'
    );
    return response.data;
  }

  async updateReminderSettings(
    settings: PaymentReminderSettings
  ): Promise<PaymentReminderSettings> {
    const response = await api.put<PaymentReminderSettings>(
      '/v1/settings/payment-reminders',
      settings
    );
    return response.data;
  }
}

export const paymentReminderService = new PaymentReminderService();
