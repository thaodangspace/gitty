import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClientProvider } from '@tanstack/react-query';
import { Provider as JotaiProvider } from 'jotai';
import { BrowserRouter } from 'react-router-dom';
import './index.css';
import App from './App.tsx';
import { queryClient } from './store/queries';

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <QueryClientProvider client={queryClient}>
            <JotaiProvider>
                <BrowserRouter>
                    <App />
                </BrowserRouter>
            </JotaiProvider>
        </QueryClientProvider>
    </StrictMode>
);
