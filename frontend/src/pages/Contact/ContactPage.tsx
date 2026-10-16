import { useState } from 'react';
import { Link } from 'react-router-dom';
import { ArrowLeft, Mail } from 'lucide-react';
import { Button, Input } from '@components/shared';
import { Footer } from '@components/layout';
import { SUPPORT_EMAIL, buildMailto } from '@/config/support';

export function ContactPage() {
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    subject: '',
    message: '',
  });
  const [opened, setOpened] = useState(false);

  // There is no contact API yet: compose the message in the visitor's own
  // email app instead of pretending it was sent from this page.
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const body = `${formData.message}\n\n— ${formData.name} <${formData.email}>`;
    window.location.href = buildMailto(SUPPORT_EMAIL, {
      subject: formData.subject,
      body,
    });
    setOpened(true);
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

            <p className="mb-6 text-sm text-neutral-600">
              Submitting opens your email app with this message addressed to{' '}
              <a
                href={`mailto:${SUPPORT_EMAIL}`}
                className="text-primary-600 hover:underline"
              >
                {SUPPORT_EMAIL}
              </a>
              . Nothing is sent until you press send there.
            </p>

            {opened && (
              <div
                role="status"
                className="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg"
              >
                <p className="text-blue-800">
                  Your email app should have opened. If it didn't, email us at{' '}
                  {SUPPORT_EMAIL}.
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
              <Button type="submit" variant="primary" className="w-full">
                Open in email app
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
                  content={SUPPORT_EMAIL}
                  href={`mailto:${SUPPORT_EMAIL}`}
                />
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
