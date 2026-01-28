import { Link } from 'react-router-dom';

export function Footer() {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="bg-white border-t border-neutral-200 mt-auto">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
          <div>
            <h3 className="text-lg font-semibold text-neutral-900 mb-4">
              ServicePro
            </h3>
            <p className="text-sm text-neutral-600">
              Professional services platform for modern businesses.
            </p>
          </div>

          <div>
            <h4 className="text-sm font-semibold text-neutral-900 mb-4">
              Product
            </h4>
            <ul className="space-y-2">
              <li>
                <Link
                  to="/features"
                  className="text-sm text-neutral-600 hover:text-neutral-900"
                >
                  Features
                </Link>
              </li>
              <li>
                <Link
                  to="/pricing"
                  className="text-sm text-neutral-600 hover:text-neutral-900"
                >
                  Pricing
                </Link>
              </li>
            </ul>
          </div>

          <div>
            <h4 className="text-sm font-semibold text-neutral-900 mb-4">
              Company
            </h4>
            <ul className="space-y-2">
              <li>
                <Link
                  to="/about"
                  className="text-sm text-neutral-600 hover:text-neutral-900"
                >
                  About
                </Link>
              </li>
              <li>
                <Link
                  to="/contact"
                  className="text-sm text-neutral-600 hover:text-neutral-900"
                >
                  Contact
                </Link>
              </li>
            </ul>
          </div>

          <div>
            <h4 className="text-sm font-semibold text-neutral-900 mb-4">
              Legal
            </h4>
            <ul className="space-y-2">
              <li>
                <Link
                  to="/privacy"
                  className="text-sm text-neutral-600 hover:text-neutral-900"
                >
                  Privacy Policy
                </Link>
              </li>
              <li>
                <Link
                  to="/terms"
                  className="text-sm text-neutral-600 hover:text-neutral-900"
                >
                  Terms of Service
                </Link>
              </li>
            </ul>
          </div>
        </div>

        <div className="mt-8 pt-8 border-t border-neutral-200">
          <p className="text-sm text-neutral-600 text-center">
            © {currentYear} ServicePro. All rights reserved.
          </p>
        </div>
      </div>
    </footer>
  );
}
