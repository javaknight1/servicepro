import {
  ReactNode,
  createContext,
  useContext,
  useEffect,
  useState,
} from 'react';
import { Elements } from '@stripe/react-stripe-js';
import { loadStripe, Stripe } from '@stripe/stripe-js';
import { billingApi } from '@/services';

interface StripeContextValue {
  isLoading: boolean;
  isAvailable: boolean;
}

const StripeContext = createContext<StripeContextValue>({
  isLoading: true,
  isAvailable: false,
});

// eslint-disable-next-line react-refresh/only-export-components
export const useStripeContext = () => useContext(StripeContext);

interface StripeProviderProps {
  children: ReactNode;
}

// Singleton promise for loading Stripe
let stripePromise: Promise<Stripe | null> | null = null;

const getStripePromise = async (): Promise<Stripe | null> => {
  try {
    // First, check if Stripe is enabled and get the publishable key
    const response = await billingApi.getConfig();

    if (!response.data.stripe_enabled) {
      return null;
    }

    // Use the publishable key from environment if available
    const publishableKey = import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY;

    if (!publishableKey) {
      return null;
    }

    return loadStripe(publishableKey);
  } catch {
    return null;
  }
};

export const StripeProvider = ({ children }: StripeProviderProps) => {
  const [stripe, setStripe] = useState<Stripe | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const initStripe = async () => {
      if (!stripePromise) {
        stripePromise = getStripePromise();
      }
      const stripeInstance = await stripePromise;
      setStripe(stripeInstance);
      setIsLoading(false);
    };

    initStripe();
  }, []);

  const contextValue: StripeContextValue = {
    isLoading,
    isAvailable: !isLoading && stripe !== null,
  };

  // Always wrap in Elements when stripe is available
  // This ensures useStripe() and useElements() work correctly
  if (stripe) {
    return (
      <StripeContext.Provider value={contextValue}>
        <Elements stripe={stripe}>{children}</Elements>
      </StripeContext.Provider>
    );
  }

  // When stripe is not available, just provide context without Elements
  return (
    <StripeContext.Provider value={contextValue}>
      {children}
    </StripeContext.Provider>
  );
};

export default StripeProvider;
