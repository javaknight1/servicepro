import { BrowserRouter } from 'react-router-dom';
import { AppRoutes } from './routes';
import { StripeProvider } from './providers/StripeProvider';

function App() {
  return (
    <BrowserRouter>
      <StripeProvider>
        <AppRoutes />
      </StripeProvider>
    </BrowserRouter>
  );
}

export default App;
