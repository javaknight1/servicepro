import { Link } from 'react-router-dom';
import { MainLayout } from '@components/layout';
import { Button } from '@components/shared';
import { Home, ArrowLeft } from 'lucide-react';

export function NotFoundPage() {
  return (
    <MainLayout>
      <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center px-4">
        <div className="text-center">
          <h1 className="text-9xl font-bold text-primary-600">404</h1>
          <h2 className="text-3xl font-semibold text-neutral-900 mt-4">
            Page not found
          </h2>
          <p className="text-neutral-600 mt-2 max-w-md mx-auto">
            Sorry, we couldn't find the page you're looking for. It may have
            been moved or deleted.
          </p>
          <div className="flex justify-center gap-4 mt-8">
            <Link to="/">
              <Button variant="primary" className="inline-flex items-center">
                <Home className="h-4 w-4 mr-2" />
                Go Home
              </Button>
            </Link>
            <Button
              variant="outline"
              onClick={() => window.history.back()}
              className="inline-flex items-center"
            >
              <ArrowLeft className="h-4 w-4 mr-2" />
              Go Back
            </Button>
          </div>
        </div>
      </div>
    </MainLayout>
  );
}
