import { Link } from 'react-router-dom';
import { MainLayout } from '@components/layout';
import { Button, Card } from '@components/shared';
import { CheckCircle, Shield, Zap } from 'lucide-react';

export function LandingPage() {
  return (
    <MainLayout>
      {/* Hero Section */}
      <section className="bg-gradient-to-b from-primary-50 to-white py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h1 className="text-5xl md:text-6xl font-bold text-neutral-900 mb-6">
              Welcome to <span className="text-primary-600">ServicePro</span>
            </h1>
            <p className="text-xl md:text-2xl text-neutral-600 mb-8 max-w-3xl mx-auto">
              The professional services platform built for modern businesses.
              Streamline your operations and focus on what matters most.
            </p>
            <div className="flex justify-center gap-4">
              <Link to="/register">
                <Button size="lg">Get Started Free</Button>
              </Link>
              <Link to="/login">
                <Button variant="outline" size="lg">
                  Sign In
                </Button>
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold text-neutral-900 mb-4">
              Why Choose ServicePro?
            </h2>
            <p className="text-xl text-neutral-600">
              Everything you need to manage your professional services
            </p>
          </div>

          <div className="grid md:grid-cols-3 gap-8">
            <Card variant="elevated" padding="lg">
              <div className="text-primary-600 mb-4">
                <Zap className="h-12 w-12" />
              </div>
              <h3 className="text-xl font-semibold mb-3">Fast & Efficient</h3>
              <p className="text-neutral-600">
                Streamline your workflow with our intuitive platform designed
                for speed and efficiency.
              </p>
            </Card>

            <Card variant="elevated" padding="lg">
              <div className="text-primary-600 mb-4">
                <Shield className="h-12 w-12" />
              </div>
              <h3 className="text-xl font-semibold mb-3">Secure & Reliable</h3>
              <p className="text-neutral-600">
                Enterprise-grade security with role-based access control and
                comprehensive audit trails.
              </p>
            </Card>

            <Card variant="elevated" padding="lg">
              <div className="text-primary-600 mb-4">
                <CheckCircle className="h-12 w-12" />
              </div>
              <h3 className="text-xl font-semibold mb-3">Easy to Use</h3>
              <p className="text-neutral-600">
                Clean, modern interface that your team will love. Get started in
                minutes, not hours.
              </p>
            </Card>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="bg-primary-600 py-16">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">
            Ready to get started?
          </h2>
          <p className="text-xl text-primary-100 mb-8">
            Join thousands of professionals using ServicePro
          </p>
          <Link to="/register">
            <Button variant="secondary" size="lg">
              Create Free Account
            </Button>
          </Link>
        </div>
      </section>
    </MainLayout>
  );
}
