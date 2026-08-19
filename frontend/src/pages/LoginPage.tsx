import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import { Input } from '@/primitives/Input';
import { useToast } from '@/primitives/ToastProvider';
import { CompassLogo } from '@/theme/logo';
import { login as apiLogin } from '@/api/auth/auth';
import { useAuthStore } from '@/stores/authStore';
import { Field } from '@/features/auth/fields';
import styles from './LoginPage.module.css';

export default function LoginPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const toast = useToast();
  const token = useAuthStore((s) => s.token);
  const loginStore = useAuthStore((s) => s.login);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [remember, setRemember] = useState(true);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (token) navigate('/', { replace: true });
  }, [token, navigate]);

  async function onLogin(e: React.FormEvent) {
    e.preventDefault();
    if (loading) return;
    setLoading(true);
    try {
      const res = await apiLogin({ email, password });
      loginStore({ token: res.token, user: res.user, remember: remember !== false });
      navigate('/', { replace: true });
    } catch (err: unknown) {
      const code = (err as { code?: string })?.code;
      toast.error(code === 'UNAUTHORIZED' ? t('auth.error.invalid') : t('error.generic'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className={styles.root}>
      <div className={styles.card}>
        <div className={styles.head}>
          <CompassLogo size={44} />
          <h1 className={styles.title}>{t('auth.login.title')}</h1>
          <p className={styles.subtitle}>{t('auth.login.subtitle')}</p>
        </div>
        <form onSubmit={(e) => void onLogin(e)}>
          <Field label={t('auth.field.email')}>
            <Input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t('auth.field.email')}
              required
            />
          </Field>
          <Field label={t('auth.field.password')}>
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t('auth.field.password')}
              required
              minLength={8}
              maxLength={72}
            />
          </Field>
          <label className={styles.remember}>
            <input
              type="checkbox"
              checked={remember}
              onChange={(e) => setRemember(e.target.checked)}
            />
            <span>{t('auth.login.remember')}</span>
          </label>
          <Button type="submit" variant="primary" block size="md" loading={loading}>
            {t('auth.login.submit')}
          </Button>
        </form>
        <div className={styles.switch}>
          <Button variant="ghost" size="sm" onClick={() => navigate('/register')}>
            {t('auth.login.toRegister')}
          </Button>
        </div>
      </div>
    </div>
  );
}
