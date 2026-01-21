import React from 'react';
import {
  Document,
  Page,
  Text,
  View,
  StyleSheet,
  PDFDownloadLink,
  pdf,
} from '@react-pdf/renderer';
import { Quote, LineItem } from '../../../types/quote';

/**
 * PDF Styles
 */
const styles = StyleSheet.create({
  page: {
    padding: 40,
    fontSize: 10,
    fontFamily: 'Helvetica',
  },
  header: {
    marginBottom: 20,
  },
  companyName: {
    fontSize: 20,
    fontWeight: 'bold',
    marginBottom: 5,
  },
  quoteTitle: {
    fontSize: 16,
    fontWeight: 'bold',
    marginBottom: 10,
  },
  section: {
    marginBottom: 15,
  },
  sectionTitle: {
    fontSize: 12,
    fontWeight: 'bold',
    marginBottom: 8,
    color: '#333',
  },
  row: {
    flexDirection: 'row',
    marginBottom: 4,
  },
  label: {
    width: '30%',
    fontWeight: 'bold',
  },
  value: {
    width: '70%',
  },
  table: {
    marginTop: 10,
    marginBottom: 15,
  },
  tableHeader: {
    flexDirection: 'row',
    borderBottomWidth: 2,
    borderBottomColor: '#000',
    paddingBottom: 5,
    marginBottom: 5,
    fontWeight: 'bold',
  },
  tableRow: {
    flexDirection: 'row',
    borderBottomWidth: 1,
    borderBottomColor: '#ddd',
    paddingVertical: 5,
  },
  tableColDescription: {
    width: '45%',
  },
  tableColQuantity: {
    width: '15%',
    textAlign: 'right',
  },
  tableColPrice: {
    width: '20%',
    textAlign: 'right',
  },
  tableColTotal: {
    width: '20%',
    textAlign: 'right',
  },
  totalsSection: {
    marginTop: 15,
    alignItems: 'flex-end',
  },
  totalRow: {
    flexDirection: 'row',
    width: '40%',
    marginBottom: 5,
  },
  totalLabel: {
    width: '60%',
    textAlign: 'right',
    paddingRight: 10,
  },
  totalValue: {
    width: '40%',
    textAlign: 'right',
  },
  grandTotalRow: {
    flexDirection: 'row',
    width: '40%',
    marginTop: 5,
    paddingTop: 5,
    borderTopWidth: 2,
    borderTopColor: '#000',
    fontWeight: 'bold',
  },
  notes: {
    marginTop: 20,
    padding: 10,
    backgroundColor: '#f5f5f5',
    borderRadius: 5,
  },
  footer: {
    position: 'absolute',
    bottom: 30,
    left: 40,
    right: 40,
    textAlign: 'center',
    color: '#666',
    fontSize: 8,
  },
});

/**
 * Quote PDF Document Component
 */
interface QuotePDFDocumentProps {
  quote: Quote;
  companyName?: string;
  companyAddress?: string;
  companyPhone?: string;
  companyEmail?: string;
}

export const QuotePDFDocument: React.FC<QuotePDFDocumentProps> = ({
  quote,
  companyName = 'ServicePro',
  companyAddress = '123 Business St, City, ST 12345',
  companyPhone = '(555) 123-4567',
  companyEmail = 'info@servicepro.com',
}) => {
  const formatCurrency = (value: string | number): string => {
    const num = typeof value === 'string' ? parseFloat(value) : value;
    return `$${num.toLocaleString('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })}`;
  };

  const formatDate = (dateString: string): string => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  return (
    <Document>
      <Page size="A4" style={styles.page}>
        {/* Header */}
        <View style={styles.header}>
          <Text style={styles.companyName}>{companyName}</Text>
          <Text>{companyAddress}</Text>
          <Text>
            {companyPhone} | {companyEmail}
          </Text>
        </View>

        {/* Quote Title */}
        <Text style={styles.quoteTitle}>Quote #{quote.quote_number}</Text>

        {/* Quote Information */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Quote Information</Text>
          <View style={styles.row}>
            <Text style={styles.label}>Date:</Text>
            <Text style={styles.value}>{formatDate(quote.created_at)}</Text>
          </View>
          <View style={styles.row}>
            <Text style={styles.label}>Valid Until:</Text>
            <Text style={styles.value}>{formatDate(quote.valid_until)}</Text>
          </View>
          <View style={styles.row}>
            <Text style={styles.label}>Status:</Text>
            <Text style={styles.value}>
              {quote.status.charAt(0).toUpperCase() + quote.status.slice(1)}
            </Text>
          </View>
        </View>

        {/* Customer Information */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Customer Information</Text>
          <View style={styles.row}>
            <Text style={styles.label}>Name:</Text>
            <Text style={styles.value}>{quote.customer_name}</Text>
          </View>
          <View style={styles.row}>
            <Text style={styles.label}>Email:</Text>
            <Text style={styles.value}>{quote.customer_email}</Text>
          </View>
        </View>

        {/* Line Items Table */}
        <View style={styles.table}>
          <Text style={styles.sectionTitle}>Items</Text>

          {/* Table Header */}
          <View style={styles.tableHeader}>
            <Text style={styles.tableColDescription}>Description</Text>
            <Text style={styles.tableColQuantity}>Quantity</Text>
            <Text style={styles.tableColPrice}>Unit Price</Text>
            <Text style={styles.tableColTotal}>Total</Text>
          </View>

          {/* Table Rows */}
          {quote.items.map((item: LineItem, index: number) => (
            <View key={item.id || index} style={styles.tableRow}>
              <Text style={styles.tableColDescription}>{item.description}</Text>
              <Text style={styles.tableColQuantity}>{item.quantity}</Text>
              <Text style={styles.tableColPrice}>
                {formatCurrency(item.unit_price)}
              </Text>
              <Text style={styles.tableColTotal}>
                {formatCurrency(item.total)}
              </Text>
            </View>
          ))}
        </View>

        {/* Totals */}
        <View style={styles.totalsSection}>
          <View style={styles.totalRow}>
            <Text style={styles.totalLabel}>Subtotal:</Text>
            <Text style={styles.totalValue}>
              {formatCurrency(quote.subtotal)}
            </Text>
          </View>
          <View style={styles.totalRow}>
            <Text style={styles.totalLabel}>
              Tax ({(parseFloat(quote.tax_rate) * 100).toFixed(2)}%):
            </Text>
            <Text style={styles.totalValue}>
              {formatCurrency(quote.tax_amount)}
            </Text>
          </View>
          <View style={styles.grandTotalRow}>
            <Text style={styles.totalLabel}>Total:</Text>
            <Text style={styles.totalValue}>{formatCurrency(quote.total)}</Text>
          </View>
        </View>

        {/* Notes */}
        {quote.notes && (
          <View style={styles.notes}>
            <Text style={styles.sectionTitle}>Notes</Text>
            <Text>{quote.notes}</Text>
          </View>
        )}

        {/* Footer */}
        <Text style={styles.footer}>
          Thank you for your business! | This quote is valid until{' '}
          {formatDate(quote.valid_until)}
        </Text>
      </Page>
    </Document>
  );
};

/**
 * Generate PDF blob from quote data
 */
export const generateQuotePDF = async (
  quote: Quote,
  companyInfo?: {
    name?: string;
    address?: string;
    phone?: string;
    email?: string;
  }
): Promise<Blob> => {
  const doc = (
    <QuotePDFDocument
      quote={quote}
      companyName={companyInfo?.name}
      companyAddress={companyInfo?.address}
      companyPhone={companyInfo?.phone}
      companyEmail={companyInfo?.email}
    />
  );

  const blob = await pdf(doc).toBlob();
  return blob;
};

/**
 * Download quote as PDF
 */
export const downloadQuotePDF = async (
  quote: Quote,
  companyInfo?: {
    name?: string;
    address?: string;
    phone?: string;
    email?: string;
  }
): Promise<void> => {
  const blob = await generateQuotePDF(quote, companyInfo);
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `quote-${quote.quote_number}.pdf`;
  link.click();
  URL.revokeObjectURL(url);
};

/**
 * PDF Download Button Component
 */
interface PDFDownloadButtonProps {
  quote: Quote;
  companyInfo?: {
    name?: string;
    address?: string;
    phone?: string;
    email?: string;
  };
  className?: string;
  children?: React.ReactNode;
}

export const PDFDownloadButton: React.FC<PDFDownloadButtonProps> = ({
  quote,
  companyInfo,
  className = '',
  children = 'Download PDF',
}) => {
  return (
    <PDFDownloadLink
      document={
        <QuotePDFDocument
          quote={quote}
          companyName={companyInfo?.name}
          companyAddress={companyInfo?.address}
          companyPhone={companyInfo?.phone}
          companyEmail={companyInfo?.email}
        />
      }
      fileName={`quote-${quote.quote_number}.pdf`}
      className={className}
    >
      {({ blob, url, loading, error }) => {
        if (loading) return 'Generating PDF...';
        if (error) return 'Error generating PDF';
        return children;
      }}
    </PDFDownloadLink>
  );
};

/**
 * Preview PDF in new window
 */
export const previewQuotePDF = async (
  quote: Quote,
  companyInfo?: {
    name?: string;
    address?: string;
    phone?: string;
    email?: string;
  }
): Promise<void> => {
  const blob = await generateQuotePDF(quote, companyInfo);
  const url = URL.createObjectURL(blob);
  window.open(url, '_blank');
  // Note: URL will be revoked when the window is closed
};

/**
 * Email quote PDF
 * This would integrate with your email service
 */
export const emailQuotePDF = async (
  quote: Quote,
  recipientEmail: string,
  companyInfo?: {
    name?: string;
    address?: string;
    phone?: string;
    email?: string;
  }
): Promise<void> => {
  const blob = await generateQuotePDF(quote, companyInfo);

  // Convert blob to base64
  const reader = new FileReader();
  reader.readAsDataURL(blob);

  return new Promise((resolve, reject) => {
    reader.onloadend = async () => {
      const base64data = reader.result as string;

      try {
        // This would call your email service API
        const response = await fetch('/api/v1/quotes/email', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            quote_id: quote.id,
            recipient_email: recipientEmail,
            pdf_data: base64data,
          }),
        });

        if (!response.ok) {
          throw new Error('Failed to send email');
        }

        resolve();
      } catch (error) {
        reject(error);
      }
    };

    reader.onerror = () => {
      reject(new Error('Failed to read PDF blob'));
    };
  });
};
