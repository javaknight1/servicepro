import { BrowserRouter } from 'react-router-dom';
import { AppRoutes } from './routes';
import { StripeProvider } from './providers/StripeProvider';
import { QueryProvider } from './providers/QueryProvider';

function App() {
  return (
    <BrowserRouter>
      <QueryProvider>
        <StripeProvider>
          <AppRoutes />
        </StripeProvider>
      </QueryProvider>
    </BrowserRouter>
  );
}

export default App;
