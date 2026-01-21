import { Link } from 'react-router-dom';
import { MainLayout } from '@components/layout';
import { Button } from '@components/shared';
import { ShieldAlert, Home, ArrowLeft } from 'lucide-react';

export function UnauthorizedPage() {
  return (
    <MainLayout>
      <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center px-4">
        <div className="text-center">
          <div className="mx-auto flex items-center justify-center h-20 w-20 rounded-full bg-error-100 mb-6">
            <ShieldAlert className="h-10 w-10 text-error-600" />
          </div>
          <h1 className="text-6xl font-bold text-neutral-900">403</h1>
          <h2 className="text-3xl font-semibold text-neutral-900 mt-4">
            Access Denied
          </h2>
          <p className="text-neutral-600 mt-2 max-w-md mx-auto">
            You don't have permission to access this resource. Please contact
            your administrator if you believe this is an error.
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
