import './api';
import { createRoot } from 'react-dom/client';
import './styles/main.css';
import './styles/antd-override.css';
import App from './App';

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(<App />);
}
