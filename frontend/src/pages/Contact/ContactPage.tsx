import { useState } from 'react';
import { Link } from 'react-router-dom';
import { ArrowLeft, Mail, Phone, MapPin } from 'lucide-react';
import { Button, Input } from '@components/shared';
import { Footer } from '@components/layout';

export function ContactPage() {
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    subject: '',
    message: '',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState<
    'idle' | 'success' | 'error'
  >('idle');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setSubmitStatus('idle');

    // TODO: Implement contact form API submission
    // For now, simulate a successful submission
    try {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      setSubmitStatus('success');
      setFormData({ name: '', email: '', subject: '', message: '' });
    } catch {
      setSubmitStatus('error');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  return (
    <div className="min-h-screen bg-neutral-50 flex flex-col">
      {/* Header */}
      <header className="bg-white border-b border-neutral-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-4">
              <Link to="/" className="text-neutral-600 hover:text-neutral-900">
                <ArrowLeft className="h-5 w-5" />
              </Link>
              <Link to="/" className="text-xl font-bold text-neutral-900">
                ServicePro
              </Link>
            </div>
            <div className="flex items-center space-x-4">
              <Link to="/login">
                <Button variant="ghost">Sign In</Button>
              </Link>
              <Link to="/register">
                <Button variant="primary">Get Started</Button>
              </Link>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-neutral-900 mb-4">
            Contact Us
          </h1>
          <p className="text-xl text-neutral-600 max-w-2xl mx-auto">
            Have a question or need help? We'd love to hear from you.
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12">
          {/* Contact Form */}
          <div className="bg-white rounded-lg shadow-sm border border-neutral-200 p-8">
            <h2 className="text-2xl font-bold text-neutral-900 mb-6">
              Send us a message
            </h2>

            {submitStatus === 'success' && (
              <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg">
                <p className="text-green-800">
                  Thank you for your message! We'll get back to you soon.
                </p>
              </div>
            )}

            {submitStatus === 'error' && (
              <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
                <p className="text-red-800">
                  Something went wrong. Please try again later.
                </p>
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-6">
              <Input
                label="Name"
                name="name"
                value={formData.name}
                onChange={handleChange}
                required
                placeholder="Your name"
              />
              <Input
                label="Email"
                name="email"
                type="email"
                value={formData.email}
                onChange={handleChange}
                required
                placeholder="your@email.com"
              />
              <Input
                label="Subject"
                name="subject"
                value={formData.subject}
                onChange={handleChange}
                required
                placeholder="How can we help?"
              />
              <div>
                <label
                  htmlFor="message"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Message
                </label>
                <textarea
                  id="message"
                  name="message"
                  value={formData.message}
                  onChange={handleChange}
                  required
                  rows={5}
                  placeholder="Tell us more about your question or feedback..."
                  className="w-full px-3 py-2 border border-neutral-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
              <Button
                type="submit"
                variant="primary"
                className="w-full"
                disabled={isSubmitting}
              >
                {isSubmitting ? 'Sending...' : 'Send Message'}
              </Button>
            </form>
          </div>

          {/* Contact Information */}
          <div className="space-y-8">
            <div className="bg-white rounded-lg shadow-sm border border-neutral-200 p-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-6">
                Get in touch
              </h2>
              <div className="space-y-6">
                <ContactInfo
                  icon={<Mail className="h-6 w-6 text-primary-600" />}
                  title="Email"
                  content="support@servicepro.com"
                  href="mailto:support@servicepro.com"
                />
                <ContactInfo
                  icon={<Phone className="h-6 w-6 text-primary-600" />}
                  title="Phone"
                  content="(555) 123-4567"
                  href="tel:+15551234567"
                />
                <ContactInfo
                  icon={<MapPin className="h-6 w-6 text-primary-600" />}
                  title="Address"
                  content="123 Business Ave, Suite 100, San Francisco, CA 94102"
                />
              </div>
            </div>

            <div className="bg-white rounded-lg shadow-sm border border-neutral-200 p-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                Office Hours
              </h2>
              <div className="space-y-2 text-neutral-600">
                <p>Monday - Friday: 9:00 AM - 6:00 PM (PST)</p>
                <p>Saturday - Sunday: Closed</p>
              </div>
            </div>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  );
}

interface ContactInfoProps {
  icon: React.ReactNode;
  title: string;
  content: string;
  href?: string;
}

function ContactInfo({ icon, title, content, href }: ContactInfoProps) {
  const ContentWrapper = href ? 'a' : 'span';

  return (
    <div className="flex items-start space-x-4">
      <div className="flex-shrink-0">{icon}</div>
      <div>
        <h3 className="text-sm font-medium text-neutral-500">{title}</h3>
        <ContentWrapper
          {...(href && { href, className: 'hover:text-primary-600' })}
          className="text-neutral-900"
        >
          {content}
        </ContentWrapper>
      </div>
    </div>
  );
}

export default ContactPage;
